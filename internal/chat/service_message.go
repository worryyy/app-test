package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/snowflake"
)

func (s *Service) GetOfflineMessages(ctx context.Context, userID int64, lastMessageID int64) ([]Message, error) {
	if userID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	conversationIDs, err := s.repo.FindConversationIDsByUserID(ctx, userIDString(userID))
	if err != nil {
		return nil, bizerr.InternalWrap("查询离线消息失败", err)
	}
	if len(conversationIDs) == 0 {
		return []Message{}, nil
	}

	messages, err := s.repo.FindMessagesAfter(ctx, conversationIDs, lastMessageID)
	if err != nil {
		return nil, bizerr.InternalWrap("查询离线消息失败", err)
	}
	return messages, nil
}

func (s *Service) GetHistoryMessages(
	ctx context.Context,
	userID int64,
	conversationID string,
	oldestMessageID *int64,
	page, size int,
) (*PageResult[Message], error) {
	if userID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}
	if strings.TrimSpace(conversationID) == "" {
		return nil, ErrConversationIDRequired
	}
	page, size = normalizePage(page, size, s.defaultPageSize())

	member, err := s.repo.FindConversationMember(ctx, conversationID, userIDString(userID))
	if err != nil {
		return nil, bizerr.InternalWrap("查询历史消息失败", err)
	}
	if member == nil {
		return nil, ErrConversationAccessDenied
	}

	messages, err := s.repo.FindConversationMessagesBefore(ctx, conversationID, oldestMessageID, int64(size))
	if err != nil {
		return nil, bizerr.InternalWrap("查询历史消息失败", err)
	}
	return NewPageResult(messages, int64(len(messages)), page, size), nil
}

func (s *Service) HasUnreadMessages(ctx context.Context, userID int64) (bool, error) {
	if userID <= 0 {
		return false, bizerr.Param(errMsgInvalidParam)
	}

	total, err := s.repo.SumUnreadCount(ctx, userIDString(userID))
	if err != nil {
		return false, bizerr.InternalWrap("查询未读消息失败", err)
	}
	return total != 0, nil
}

func (s *Service) HandleWSMessage(ctx context.Context, senderID int64, payload []byte) (*Message, error) {
	body := make(map[string]any)
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, ErrMessageParseFailed
	}

	handleType := strings.ToUpper(strings.TrimSpace(stringField(body, "handleType")))
	if handleType == "INIT" {
		return s.handleInitMessage(ctx, userIDString(senderID), body)
	}
	return s.handleChatMessage(ctx, userIDString(senderID), body)
}

func (s *Service) handleInitMessage(ctx context.Context, senderID string, body map[string]any) (*Message, error) {
	conversationID := stringField(body, "id")
	if conversationID == "" {
		conversationID = stringField(body, "conversationId")
	}
	if strings.TrimSpace(conversationID) == "" {
		return nil, ErrConversationIDRequired
	}

	receiverID := stringField(body, "receiverId")
	if strings.TrimSpace(receiverID) == "" {
		return nil, ErrReceiverIDRequired
	}

	content := stringField(body, "content")
	sentAt := timeField(body, "sentAt")
	now := time.Now()
	if sentAt.IsZero() {
		sentAt = now
	}

	message := &Message{
		MessageID:      nextMessageID(),
		ConversationID: conversationID,
		ReceiverID:     receiverID,
		SenderID:       senderID,
		Content:        content,
		SentAt:         sentAt,
	}
	if err := s.repo.InsertMessage(ctx, message); err != nil {
		return nil, bizerr.InternalWrap("保存消息失败", err)
	}

	senderLastRead := message.MessageID
	conversation := &Conversation{
		ID:                  conversationID,
		Type:                1,
		LastMessageContent:  content,
		LastMessageSenderID: senderID,
		LastMessageSentAt:   sentAt,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	members := []ConversationMember{
		{
			ConversationID:    conversationID,
			UserID:            senderID,
			LastReadMessageID: &senderLastRead,
			UnreadCount:       0,
			CreatedAt:         now,
		},
		{
			ConversationID:    conversationID,
			UserID:            receiverID,
			LastReadMessageID: nil,
			UnreadCount:       1,
			CreatedAt:         now,
		},
	}
	if err := s.repo.CreateConversationWithMembers(ctx, conversation, members); err != nil {
		return nil, bizerr.InternalWrap("创建会话失败", err)
	}
	return message, nil
}

func (s *Service) handleChatMessage(ctx context.Context, senderID string, body map[string]any) (*Message, error) {
	conversationID := stringField(body, "conversationId")
	if strings.TrimSpace(conversationID) == "" {
		return nil, ErrConversationIDRequired
	}

	receiverID := stringField(body, "receiverId")
	if strings.TrimSpace(receiverID) == "" {
		return nil, ErrReceiverIDRequired
	}

	content := stringField(body, "content")
	sentAt := timeField(body, "sentAt")
	if sentAt.IsZero() {
		sentAt = time.Now()
	}

	message := &Message{
		MessageID:      nextMessageID(),
		ConversationID: conversationID,
		ReceiverID:     receiverID,
		SenderID:       senderID,
		Content:        content,
		SentAt:         sentAt,
	}
	if err := s.repo.InsertMessage(ctx, message); err != nil {
		return nil, bizerr.InternalWrap("保存消息失败", err)
	}

	if err := s.repo.UpdateConversationAfterMessage(ctx, conversationID, senderID, receiverID, content, sentAt, message.MessageID); err != nil {
		switch {
		case errors.Is(err, errRepoConversationMemberMiss):
			return nil, ErrConversationUpdateFailed
		case errors.Is(err, errRepoConversationNotFound):
			return nil, ErrConversationUpdateFailed
		case errors.Is(err, errRepoConversationUpdateFailed):
			return nil, ErrConversationUpdateFailed
		default:
			return nil, bizerr.InternalWrap("更新会话失败", err)
		}
	}

	return message, nil
}

func nextMessageID() int64 {
	return snowflake.Generate().Int64()
}

func stringField(body map[string]any, key string) string {
	v, ok := body[key]
	if !ok || v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case primitive.ObjectID:
		return val.Hex()
	case float64:
		return strconv.FormatInt(int64(val), 10)
	case json.Number:
		return val.String()
	default:
		return strings.TrimSpace(fmt.Sprint(val))
	}
}

func timeField(body map[string]any, key string) time.Time {
	v, ok := body[key]
	if !ok || v == nil {
		return time.Time{}
	}
	switch val := v.(type) {
	case string:
		layouts := []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, val); err == nil {
				return t
			}
		}
	case float64:
		return time.UnixMilli(int64(val))
	case json.Number:
		if n, err := val.Int64(); err == nil {
			return time.UnixMilli(n)
		}
	}
	return time.Time{}
}
