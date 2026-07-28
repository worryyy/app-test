package chat

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (h *Handler) PushNotification(ctx context.Context, targetUserID string, payload interface{}) error {
	_ = ctx

	userID, err := strconv.ParseInt(targetUserID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid target user id: %w", err)
	}
	session, ok := h.sessions.Get(userID)
	if !ok || session == nil {
		return nil
	}

	notification, err := normalizeNotificationPayload(payload)
	if err != nil {
		return err
	}
	return session.WriteJSON(newRealtimeNotification(notification))
}

func normalizeNotificationPayload(payload interface{}) (*Notification, error) {
	switch value := payload.(type) {
	case *Notification:
		return value, nil
	case Notification:
		notification := value
		return &notification, nil
	case map[string]interface{}:
		notification := &Notification{
			ReceiverID:  mapString(value, "receiverId", "receiver_id"),
			SenderID:    mapString(value, "senderId", "sender_id"),
			Type:        mapString(value, "type"),
			Content:     mapString(value, "content"),
			TopicID:     mapString(value, "topicId", "topic_id"),
			CommentID:   mapString(value, "commentId", "comment_id"),
			CreatedTime: mapTime(value, "createdTime", "created_time"),
			IsRead:      mapBool(value, "isRead", "is_read"),
		}
		if id := mapString(value, "id", "_id"); id != "" {
			notification.IDString = id
			if oid, err := primitive.ObjectIDFromHex(id); err == nil {
				notification.ID = oid
			}
		}
		return notification, nil
	default:
		return nil, fmt.Errorf("unsupported notification payload type %T", payload)
	}
}

type realtimeNotification struct {
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

func newRealtimeNotification(notification *Notification) realtimeNotification {
	if notification == nil {
		return realtimeNotification{}
	}
	return realtimeNotification{
		ID:          notificationIDString(*notification),
		ReceiverID:  notification.ReceiverID,
		SenderID:    notification.SenderID,
		Type:        notification.Type,
		Content:     notification.Content,
		TopicID:     notification.TopicID,
		CommentID:   notification.CommentID,
		CreatedTime: formatChatDate(notification.CreatedTime),
		IsRead:      notification.IsRead,
	}
}

func objectIDString(id primitive.ObjectID) string {
	if id.IsZero() {
		return ""
	}
	return id.Hex()
}

func notificationIDString(notification Notification) string {
	if notification.IDString != "" {
		return notification.IDString
	}
	return objectIDString(notification.ID)
}

func mapString(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value := stringField(data, key); value != "" {
			return value
		}
	}
	return ""
}

func mapTime(data map[string]interface{}, keys ...string) time.Time {
	for _, key := range keys {
		if value := timeField(data, key); !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func mapBool(data map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		raw, ok := data[key]
		if !ok || raw == nil {
			continue
		}
		if value, ok := raw.(bool); ok {
			return value
		}
	}
	return false
}
