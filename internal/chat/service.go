package chat

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Service struct {
	db      *gorm.DB
	mongoDB *mongo.Database
	redis   *redis.Client
	cfg     *config.Config
	logger  *zap.Logger
}

func NewService(db *gorm.DB, mongoDB *mongo.Database, rds *redis.Client, cfg *config.Config, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		db:      db,
		mongoDB: mongoDB,
		redis:   rds,
		cfg:     cfg,
		logger:  logger,
	}
}

func (s *Service) ListConversations(ctx context.Context, userID int64) ([]Conversation, error) {
	var ids []int64
	if err := s.db.WithContext(ctx).Model(&ConversationMember{}).Where("userId = ?", userID).Pluck("conversationId", &ids).Error; err != nil {
		return nil, fmt.Errorf("load conversation ids: %w", err)
	}
	if len(ids) == 0 {
		return []Conversation{}, nil
	}

	var conversations []Conversation
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Order("lastMessageSentAt DESC").Find(&conversations).Error; err != nil {
		return nil, fmt.Errorf("load conversations: %w", err)
	}
	return conversations, nil
}

func (s *Service) EnterConversation(ctx context.Context, userID, conversationID int64) error {
	if err := s.db.WithContext(ctx).
		Model(&ConversationMember{}).
		Where("userId = ? AND conversationId = ?", userID, conversationID).
		Updates(map[string]interface{}{"unreadCount": 0}).Error; err != nil {
		return fmt.Errorf("enter conversation: %w", err)
	}
	return nil
}

func (s *Service) GetUnreadCount(ctx context.Context, userID, conversationID int64) (int, error) {
	var m ConversationMember
	err := s.db.WithContext(ctx).Where("userId = ? AND conversationId = ?", userID, conversationID).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get unread count: %w", err)
	}
	return m.UnreadCount, nil
}

func (s *Service) QueryConversation(ctx context.Context, userID, targetUserID int64) (*Conversation, error) {
	var myMembers []ConversationMember
	if err := s.db.WithContext(ctx).Where("userId = ?", userID).Find(&myMembers).Error; err != nil {
		return nil, fmt.Errorf("query my members: %w", err)
	}
	for _, member := range myMembers {
		var count int64
		if err := s.db.WithContext(ctx).
			Model(&ConversationMember{}).
			Where("conversationId = ? AND userId = ?", member.ConversationID, targetUserID).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("query target member: %w", err)
		}
		if count > 0 {
			var c Conversation
			if err := s.db.WithContext(ctx).First(&c, member.ConversationID).Error; err != nil {
				return nil, fmt.Errorf("query conversation: %w", err)
			}
			return &c, nil
		}
	}
	return nil, nil
}

func (s *Service) DeleteConversation(ctx context.Context, userID, conversationID int64) error {
	if err := s.db.WithContext(ctx).Where("userId = ? AND conversationId = ?", userID, conversationID).Delete(&ConversationMember{}).Error; err != nil {
		return fmt.Errorf("delete conversation member: %w", err)
	}
	return nil
}

func (s *Service) GetOfflineMessages(ctx context.Context, userID, lastMessageID int64) ([]Message, error) {
	filter := bson.M{"receiver_id": userID}
	if lastMessageID > 0 {
		filter["message_id"] = bson.M{"$gt": lastMessageID}
	}
	cur, err := s.messageColl().Find(ctx, filter, options.Find().SetSort(bson.M{"message_id": 1}))
	if err != nil {
		return nil, fmt.Errorf("find offline messages: %w", err)
	}
	defer cur.Close(ctx)

	var msgs []Message
	if err := cur.All(ctx, &msgs); err != nil {
		return nil, fmt.Errorf("decode offline messages: %w", err)
	}
	return msgs, nil
}

func (s *Service) GetHistoryMessages(ctx context.Context, conversationID int64, page, size int) (*result.CusPage[Message], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}

	filter := bson.M{"conversation_id": conversationID}
	total, err := s.messageColl().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count history messages: %w", err)
	}

	cur, err := s.messageColl().Find(ctx, filter, options.Find().
		SetSort(bson.M{"message_id": -1}).
		SetSkip(int64((page-1)*size)).
		SetLimit(int64(size)))
	if err != nil {
		return nil, fmt.Errorf("find history messages: %w", err)
	}
	defer cur.Close(ctx)

	var msgs []Message
	if err := cur.All(ctx, &msgs); err != nil {
		return nil, fmt.Errorf("decode history messages: %w", err)
	}
	return result.NewCusPage(msgs, total, page, size), nil
}

func (s *Service) GetUnreadMessages(ctx context.Context, userID int64) ([]Message, error) {
	cur, err := s.messageColl().Find(ctx, bson.M{"receiver_id": userID}, options.Find().SetSort(bson.M{"message_id": -1}))
	if err != nil {
		return nil, fmt.Errorf("find unread messages: %w", err)
	}
	defer cur.Close(ctx)

	var msgs []Message
	if err := cur.All(ctx, &msgs); err != nil {
		return nil, fmt.Errorf("decode unread messages: %w", err)
	}
	return msgs, nil
}

func (s *Service) ListNotifications(ctx context.Context, userID int64, typ string, page, size int) (*result.CusPage[Notification], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	filter := bson.M{"userId": fmt.Sprintf("%d", userID)}
	if typ != "" {
		filter["type"] = typ
	}

	total, err := s.notifyColl().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count notifications: %w", err)
	}

	cur, err := s.notifyColl().Find(ctx, filter, options.Find().
		SetSort(bson.M{"createdAt": -1}).
		SetSkip(int64((page-1)*size)).
		SetLimit(int64(size)))
	if err != nil {
		return nil, fmt.Errorf("find notifications: %w", err)
	}
	defer cur.Close(ctx)

	var list []Notification
	if err := cur.All(ctx, &list); err != nil {
		return nil, fmt.Errorf("decode notifications: %w", err)
	}
	return result.NewCusPage(list, total, page, size), nil
}

func (s *Service) HaveUnreadNotification(ctx context.Context, userID int64, typ string) (bool, error) {
	filter := bson.M{"userId": fmt.Sprintf("%d", userID), "isRead": false}
	if typ != "" {
		filter["type"] = typ
	}
	count, err := s.notifyColl().CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("count unread notifications: %w", err)
	}
	return count > 0, nil
}

func (s *Service) LatestNotification(ctx context.Context, userID int64, typ string) (*Notification, error) {
	filter := bson.M{"userId": fmt.Sprintf("%d", userID)}
	if typ != "" {
		filter["type"] = typ
	}

	var n Notification
	err := s.notifyColl().FindOne(ctx, filter, options.FindOne().SetSort(bson.M{"createdAt": -1})).Decode(&n)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find latest notification: %w", err)
	}
	return &n, nil
}

func (s *Service) HandleMessage(ctx context.Context, conversationID, senderID, receiverID int64, content string) (*Message, error) {
	msg := &Message{
		MessageID:      time.Now().UnixNano(),
		ConversationID: conversationID,
		ReceiverID:     receiverID,
		SenderID:       senderID,
		Content:        content,
		SentAt:         time.Now(),
	}
	if _, err := s.messageColl().InsertOne(ctx, msg); err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	if err := s.db.WithContext(ctx).Model(&Conversation{}).Where("id = ?", conversationID).Updates(map[string]interface{}{
		"lastMessageContent":  content,
		"lastMessageSenderId": senderID,
		"lastMessageSentAt":   msg.SentAt,
	}).Error; err != nil {
		return nil, fmt.Errorf("update conversation last message: %w", err)
	}

	if err := s.db.WithContext(ctx).Model(&ConversationMember{}).
		Where("conversationId = ? AND userId = ?", conversationID, receiverID).
		Update("unreadCount", gorm.Expr("unreadCount + 1")).Error; err != nil {
		return nil, fmt.Errorf("increase unread count: %w", err)
	}

	return msg, nil
}

func (s *Service) messageColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_messages")
}

func (s *Service) notifyColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_notifications")
}
