package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/snowflake"
)

func (s *Service) GetOfflineMessages(ctx context.Context, userID int64, lastMessageID int64) ([]Message, error) {
	conversationIDs, err := s.getConversationIDs(ctx, userIDString(userID))
	if err != nil {
		return nil, err
	}
	if len(conversationIDs) == 0 {
		return []Message{}, nil
	}

	filter := bson.M{"conversation_id": bson.M{"$in": conversationIDs}}
	if lastMessageID > 0 {
		filter["message_id"] = bson.M{"$gt": lastMessageID}
	}

	cur, err := s.messageColl().Find(ctx, filter, options.Find().SetSort(bson.M{"message_id": 1}))
	if err != nil {
		return nil, fmt.Errorf("find offline messages: %w", err)
	}
	defer closeCursor(ctx, s.logger, cur, "close offline message cursor failed")

	var msgs []Message
	if err := cur.All(ctx, &msgs); err != nil {
		return nil, fmt.Errorf("decode offline messages: %w", err)
	}
	if msgs == nil {
		return []Message{}, nil
	}
	return msgs, nil
}

func (s *Service) GetHistoryMessages(
	ctx context.Context,
	userID int64,
	conversationID string,
	oldestMessageID *int64,
	page, size int,
) (*result.CusPage[Message], error) {
	if strings.TrimSpace(conversationID) == "" {
		return nil, newFail("会话ID不能为空")
	}
	page, size = normalizePage(page, size, s.defaultPageSize())

	var member ConversationMember
	err := s.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userIDString(userID)).
		First(&member).Error
	if err == gorm.ErrRecordNotFound {
		return nil, newFail("无权访问该会话历史")
	}
	if err != nil {
		return nil, fmt.Errorf("load history conversation member: %w", err)
	}

	filter := bson.M{"conversation_id": conversationID}
	if oldestMessageID != nil {
		filter["message_id"] = bson.M{"$lt": *oldestMessageID}
	}

	cur, err := s.messageColl().Find(ctx, filter, options.Find().
		SetSort(bson.M{"message_id": 1}).
		SetLimit(50))
	if err != nil {
		return nil, fmt.Errorf("find history messages: %w", err)
	}
	defer closeCursor(ctx, s.logger, cur, "close history message cursor failed")

	var msgs []Message
	if err := cur.All(ctx, &msgs); err != nil {
		return nil, fmt.Errorf("decode history messages: %w", err)
	}
	if msgs == nil {
		msgs = []Message{}
	}
	return result.NewCusPage(msgs, int64(len(msgs)), page, size), nil
}

func (s *Service) HasUnreadMessages(ctx context.Context, userID int64) (bool, error) {
	var total int64
	row := s.db.WithContext(ctx).
		Model(&ConversationMember{}).
		Select("COALESCE(SUM(unread_count), 0)").
		Where("user_id = ?", userIDString(userID)).
		Row()
	if err := row.Scan(&total); err != nil {
		return false, fmt.Errorf("sum unread count: %w", err)
	}
	return total != 0, nil
}

func (s *Service) HandleWSMessage(ctx context.Context, senderID int64, payload []byte) (*Message, error) {
	body := make(map[string]interface{})
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, newFail("消息解析失败")
	}

	handleType := strings.ToUpper(strings.TrimSpace(stringField(body, "handleType")))
	if handleType == "INIT" {
		return s.handleInitMessage(ctx, userIDString(senderID), body)
	}
	return s.handleChatMessage(ctx, userIDString(senderID), body)
}

func (s *Service) handleInitMessage(ctx context.Context, senderID string, body map[string]interface{}) (*Message, error) {
	conversationID := stringField(body, "id")
	if conversationID == "" {
		conversationID = stringField(body, "conversationId")
	}
	if strings.TrimSpace(conversationID) == "" {
		return nil, newFail("会话ID不能为空")
	}

	receiverID := stringField(body, "receiverId")
	if strings.TrimSpace(receiverID) == "" {
		return nil, newFail("接收者ID不能为空")
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
	if _, err := s.messageColl().InsertOne(ctx, message); err != nil {
		return nil, fmt.Errorf("insert init message: %w", err)
	}

	senderLastRead := message.MessageID
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		conversation := Conversation{
			ID:                  conversationID,
			Type:                1,
			LastMessageContent:  content,
			LastMessageSenderID: senderID,
			LastMessageSentAt:   sentAt,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		if err := tx.Create(&conversation).Error; err != nil {
			return fmt.Errorf("create conversation: %w", err)
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
		if err := tx.Create(&members).Error; err != nil {
			return fmt.Errorf("create conversation members: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (s *Service) handleChatMessage(ctx context.Context, senderID string, body map[string]interface{}) (*Message, error) {
	conversationID := stringField(body, "conversationId")
	if strings.TrimSpace(conversationID) == "" {
		return nil, newFail("会话ID不能为空")
	}

	receiverID := stringField(body, "receiverId")
	if strings.TrimSpace(receiverID) == "" {
		return nil, newFail("接收者ID不能为空")
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
	if _, err := s.messageColl().InsertOne(ctx, message); err != nil {
		return nil, fmt.Errorf("insert chat message: %w", err)
	}

	senderUpdate := s.db.WithContext(ctx).
		Model(&ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, senderID).
		Update("last_read_message_id", message.MessageID)
	if senderUpdate.Error != nil {
		return nil, fmt.Errorf("update sender last read message: %w", senderUpdate.Error)
	}
	if senderUpdate.RowsAffected == 0 {
		return nil, newFail("更新会话成员未读数失败")
	}

	receiverUpdate := s.db.WithContext(ctx).
		Model(&ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, receiverID).
		Update("unread_count", gorm.Expr("unread_count + 1"))
	if receiverUpdate.Error != nil {
		return nil, fmt.Errorf("increase unread count: %w", receiverUpdate.Error)
	}
	if receiverUpdate.RowsAffected == 0 {
		return nil, newFail("更新会话成员未读数失败")
	}

	conversationUpdate := s.db.WithContext(ctx).
		Model(&Conversation{}).
		Where("id = ?", conversationID).
		Updates(map[string]interface{}{
			"last_message_content":   content,
			"last_message_sender_id": senderID,
			"last_message_sent_at":   sentAt,
			"updated_at":             time.Now(),
		})
	if conversationUpdate.Error != nil {
		return nil, fmt.Errorf("update conversation last message: %w", conversationUpdate.Error)
	}
	if conversationUpdate.RowsAffected == 0 {
		return nil, newFail("更新会话信息失败")
	}

	return message, nil
}

func nextMessageID() int64 {
	return snowflake.Generate().Int64()
}

func stringField(body map[string]interface{}, key string) string {
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

func timeField(body map[string]interface{}, key string) time.Time {
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
