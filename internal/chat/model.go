package chat

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Conversation struct {
	ID                  string    `gorm:"column:id;primaryKey" json:"id"`
	Type                int       `gorm:"column:type" json:"type"`
	LastMessageContent  string    `gorm:"column:last_message_content" json:"lastMessageContent"`
	LastMessageSenderID string    `gorm:"column:last_message_sender_id" json:"lastMessageSenderId"`
	LastMessageSentAt   time.Time `gorm:"column:last_message_sent_at" json:"lastMessageSentAt"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (Conversation) TableName() string {
	return "conversations"
}

type ConversationMember struct {
	ConversationID    string    `gorm:"column:conversation_id;primaryKey" json:"conversationId"`
	UserID            string    `gorm:"column:user_id;primaryKey" json:"userId"`
	LastReadMessageID *int64    `gorm:"column:last_read_message_id" json:"lastReadMessageId"`
	UnreadCount       int       `gorm:"column:unread_count" json:"unreadCount"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
}

func (ConversationMember) TableName() string {
	return "conversation_members"
}

type Message struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	MessageID      int64              `bson:"message_id" json:"messageId"`
	ConversationID string             `bson:"conversation_id" json:"conversationId"`
	ReceiverID     string             `bson:"receiver_id" json:"receiverId"`
	SenderID       string             `bson:"sender_id" json:"senderId"`
	Content        string             `bson:"content" json:"content"`
	MessageType    *int               `bson:"message_type,omitempty" json:"messageType,omitempty"`
	SentAt         time.Time          `bson:"sentAt" json:"sentAt"`
	Metadata       map[string]any     `bson:"metadata,omitempty" json:"metadata,omitempty"`
	HandleType     string             `bson:"-" json:"handleType,omitempty"`
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

type ConversationUnreadCount struct {
	UnreadCount int `gorm:"column:unread_count" json:"unreadCount"`
}

type ConversationProfile struct {
	Avatar    string `json:"avatar"`
	Nickname  string `json:"nickname"`
	UserID    string `json:"userId"`
	Gender    string `json:"gender"`
	StuCla    string `json:"stuCla"`
	Signature string `json:"signature"`
}
