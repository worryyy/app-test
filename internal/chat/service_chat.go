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
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/snowflake"
)

func (s *Service) handleInitMessage(ctx context.Context, senderID string, body map[string]any) (*Message, error) {
	conversationID := stringField(body, "id", "conversationId", "conversation_id")
	if strings.TrimSpace(conversationID) == "" {
		return nil, ErrConversationIDRequired
	}

	receiverID := stringField(body, "receiverId", "receiver_id")
	if strings.TrimSpace(receiverID) == "" {
		return nil, ErrReceiverIDRequired
	}

	content := stringField(body, "content")
	sentAt := timeField(body, "sentAt", "sent_at")
	now := time.Now()
	if sentAt.IsZero() {
		sentAt = now
	}

	messageID := nextMessageID()
	senderLastRead := messageID
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

	message := &Message{
		MessageID:      messageID,
		ConversationID: conversationID,
		ReceiverID:     receiverID,
		SenderID:       senderID,
		Content:        content,
		MessageType:    intField(body, "messageType", "message_type"),
		SentAt:         sentAt,
		Metadata:       mapField(body, "metadata"),
		HandleType:     "INIT",
	}
	if err := s.repo.InsertMessage(ctx, message); err != nil {
		if cleanupErr := s.repo.DeleteConversationCascade(ctx, conversationID); cleanupErr != nil && s.logger != nil {
			s.logger.Warn(
				"rollback init conversation failed",
				zap.Error(cleanupErr),
				zap.String("conversationID", conversationID),
			)
		}
		return nil, bizerr.InternalWrap("保存消息失败", err)
	}
	return message, nil
}

func (s *Service) handleChatMessage(ctx context.Context, senderID string, body map[string]any) (*Message, error) {
	conversationID := stringField(body, "conversationId", "conversation_id")
	if strings.TrimSpace(conversationID) == "" {
		return nil, ErrConversationIDRequired
	}

	receiverID := stringField(body, "receiverId", "receiver_id")
	if strings.TrimSpace(receiverID) == "" {
		return nil, ErrReceiverIDRequired
	}

	senderMember, err := s.repo.FindConversationMember(ctx, conversationID, senderID)
	if err != nil {
		return nil, bizerr.InternalWrap("查询会话成员失败", err)
	}
	if senderMember == nil {
		return nil, ErrConversationAccessDenied
	}

	receiverMember, err := s.repo.FindConversationMember(ctx, conversationID, receiverID)
	if err != nil {
		return nil, bizerr.InternalWrap("查询会话成员失败", err)
	}
	if receiverMember == nil {
		return nil, ErrConversationPeerNotFound
	}

	content := stringField(body, "content")
	sentAt := timeField(body, "sentAt", "sent_at")
	if sentAt.IsZero() {
		sentAt = time.Now()
	}

	message := &Message{
		MessageID:      nextMessageID(),
		ConversationID: conversationID,
		ReceiverID:     receiverID,
		SenderID:       senderID,
		Content:        content,
		MessageType:    intField(body, "messageType", "message_type"),
		SentAt:         sentAt,
		Metadata:       mapField(body, "metadata"),
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

func stringField(body map[string]any, keys ...string) string {
	for _, key := range keys {
		v, ok := body[key]
		if !ok || v == nil {
			continue
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
	return ""
}

func timeField(body map[string]any, keys ...string) time.Time {
	for _, key := range keys {
		v, ok := body[key]
		if !ok || v == nil {
			continue
		}
		if value := normalizeTimeValue(v); !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func normalizeTimeValue(v any) time.Time {
	switch val := v.(type) {
	case time.Time:
		return val
	case primitive.DateTime:
		return val.Time()
	case string:
		raw := strings.TrimSpace(val)
		if unixMillis, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return time.UnixMilli(unixMillis)
		}
		layouts := []string{
			time.RFC3339,
			time.RFC3339Nano,
			chatDateLayout,
			"2006-01-02T15:04:05",
			"2006-01-02T15:04:05.999",
			"2006-01-02T15:04:05.999999999",
			"2006-01-02 15:04:05",
			"2006-01-02 15:04:05.999",
			"2006-01-02 15:04:05.999999999",
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, raw); err == nil {
				return t
			}
		}
	case float64:
		return time.UnixMilli(int64(val))
	case int:
		return time.UnixMilli(int64(val))
	case int32:
		return time.UnixMilli(int64(val))
	case int64:
		return time.UnixMilli(val)
	case json.Number:
		if n, err := val.Int64(); err == nil {
			return time.UnixMilli(n)
		}
	}
	return time.Time{}
}

func intField(body map[string]any, keys ...string) *int {
	for _, key := range keys {
		v, ok := body[key]
		if !ok || v == nil {
			continue
		}
		switch val := v.(type) {
		case int:
			value := val
			return &value
		case int32:
			value := int(val)
			return &value
		case int64:
			value := int(val)
			return &value
		case float64:
			value := int(val)
			return &value
		case json.Number:
			if n, err := val.Int64(); err == nil {
				value := int(n)
				return &value
			}
		case string:
			if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				value := n
				return &value
			}
		}
	}
	return nil
}

func mapField(body map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		v, ok := body[key]
		if !ok || v == nil {
			continue
		}
		switch val := v.(type) {
		case map[string]any:
			return val
		case primitive.M:
			return map[string]any(val)
		}
	}
	return nil
}
