package chat

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Conversation struct {
	ID                  int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Type                int       `gorm:"column:type" json:"type"`
	LastMessageContent  string    `gorm:"column:lastMessageContent" json:"lastMessageContent"`
	LastMessageSenderID int64     `gorm:"column:lastMessageSenderId" json:"lastMessageSenderId"`
	LastMessageSentAt   time.Time `gorm:"column:lastMessageSentAt" json:"lastMessageSentAt"`
	CreatedAt           time.Time `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt           time.Time `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
}

func (Conversation) TableName() string {
	return "conversations"
}

type ConversationMember struct {
	ID                int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ConversationID    int64 `gorm:"column:conversationId" json:"conversationId"`
	UserID            int64 `gorm:"column:userId" json:"userId"`
	LastReadMessageID int64 `gorm:"column:lastReadMessageId" json:"lastReadMessageId"`
	UnreadCount       int   `gorm:"column:unreadCount" json:"unreadCount"`
}

func (ConversationMember) TableName() string {
	return "conversation_members"
}

type Message struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MessageID      int64              `bson:"message_id" json:"messageId"`
	ConversationID int64              `bson:"conversation_id" json:"conversationId"`
	ReceiverID     int64              `bson:"receiver_id" json:"receiverId"`
	SenderID       int64              `bson:"sender_id" json:"senderId"`
	Content        string             `bson:"content" json:"content"`
	SentAt         time.Time          `bson:"sentAt" json:"sentAt"`
}

type Notification struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ReceiverID  string             `bson:"receiver_id" json:"receiverId"`
	SenderID    string             `bson:"sender_id" json:"senderId"`
	Type        string             `bson:"type" json:"type"`
	Content     string             `bson:"content" json:"content"`
	TopicID     string             `bson:"topic_id" json:"topicId"`
	CommentID   string             `bson:"comment_id" json:"commentId"`
	CreatedTime time.Time          `bson:"created_time" json:"createdTime"`
	IsRead      bool               `bson:"is_read" json:"isRead"`
}
