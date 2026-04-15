package chat

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const chatDateLayout = "2006-01-02"

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

func (c Conversation) MarshalJSON() ([]byte, error) {
	type conversationJSON struct {
		ID                  string `json:"id"`
		Type                int    `json:"type"`
		LastMessageContent  string `json:"lastMessageContent"`
		LastMessageSenderID string `json:"lastMessageSenderId"`
		LastMessageSentAt   string `json:"lastMessageSentAt"`
		CreatedAt           string `json:"createdAt"`
		UpdatedAt           string `json:"updatedAt"`
	}

	return json.Marshal(conversationJSON{
		ID:                  c.ID,
		Type:                c.Type,
		LastMessageContent:  c.LastMessageContent,
		LastMessageSenderID: c.LastMessageSenderID,
		LastMessageSentAt:   formatChatDate(c.LastMessageSentAt),
		CreatedAt:           formatChatDate(c.CreatedAt),
		UpdatedAt:           formatChatDate(c.UpdatedAt),
	})
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

func (m ConversationMember) MarshalJSON() ([]byte, error) {
	type conversationMemberJSON struct {
		ConversationID    string `json:"conversationId"`
		UserID            string `json:"userId"`
		LastReadMessageID *int64 `json:"lastReadMessageId"`
		UnreadCount       int    `json:"unreadCount"`
		CreatedAt         string `json:"createdAt"`
	}

	return json.Marshal(conversationMemberJSON{
		ConversationID:    m.ConversationID,
		UserID:            m.UserID,
		LastReadMessageID: m.LastReadMessageID,
		UnreadCount:       m.UnreadCount,
		CreatedAt:         formatChatDate(m.CreatedAt),
	})
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

func (m Message) MarshalJSON() ([]byte, error) {
	type messageJSON struct {
		MessageID      int64          `json:"messageId"`
		ConversationID string         `json:"conversationId"`
		ReceiverID     string         `json:"receiverId"`
		SenderID       string         `json:"senderId"`
		Content        string         `json:"content"`
		MessageType    *int           `json:"messageType,omitempty"`
		SentAt         string         `json:"sentAt"`
		Metadata       map[string]any `json:"metadata,omitempty"`
		HandleType     string         `json:"handleType,omitempty"`
	}

	return json.Marshal(messageJSON{
		MessageID:      m.MessageID,
		ConversationID: m.ConversationID,
		ReceiverID:     m.ReceiverID,
		SenderID:       m.SenderID,
		Content:        m.Content,
		MessageType:    m.MessageType,
		SentAt:         formatChatDate(m.SentAt),
		Metadata:       m.Metadata,
		HandleType:     m.HandleType,
	})
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

func (n Notification) MarshalJSON() ([]byte, error) {
	type notificationJSON struct {
		ID          string `json:"id"`
		ReceiverID  string `json:"receiverId"`
		SenderID    string `json:"senderId"`
		Type        string `json:"type"`
		Content     string `json:"content"`
		TopicID     string `json:"topicId"`
		CommentID   string `json:"commentId"`
		CreatedTime string `json:"createdTime"`
		IsRead      bool   `json:"isRead"`
	}

	return json.Marshal(notificationJSON{
		ID:          objectIDString(n.ID),
		ReceiverID:  n.ReceiverID,
		SenderID:    n.SenderID,
		Type:        n.Type,
		Content:     n.Content,
		TopicID:     n.TopicID,
		CommentID:   n.CommentID,
		CreatedTime: formatChatDate(n.CreatedTime),
		IsRead:      n.IsRead,
	})
}

type ConversationUnreadCount struct {
	UnreadCount       int    `gorm:"column:unread_count" json:"unreadCount"`
	LastReadMessageID *int64 `gorm:"column:last_read_message_id" json:"lastReadMessageId"`
}

type ConversationProfile struct {
	Avatar    string `json:"avatar"`
	Nickname  string `json:"nickname"`
	UserID    string `json:"userId"`
	Gender    string `json:"gender"`
	StuCla    string `json:"stuCla"`
	Signature string `json:"signature"`
}

func formatChatDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(chatDateLayout)
}
