package chat

import (
	"encoding/json"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestRealtimeNotificationMarshal(t *testing.T) {
	notification := &Notification{
		ID:          primitive.NewObjectID(),
		ReceiverID:  "42",
		SenderID:    "7",
		Type:        "TOPIC_LIKE",
		Content:     "x",
		TopicID:     "t1",
		CommentID:   "c1",
		CreatedTime: time.UnixMilli(1710000000000),
		IsRead:      false,
	}

	raw, err := json.Marshal(newRealtimeNotification(notification))
	if err != nil {
		t.Fatalf("marshal realtime notification: %v", err)
	}

	want := "{\"id\":\"" + notification.ID.Hex() + "\",\"receiverId\":\"42\",\"senderId\":\"7\",\"type\":\"TOPIC_LIKE\",\"content\":\"x\",\"topicId\":\"t1\",\"commentId\":\"c1\",\"createdTime\":1710000000000,\"isRead\":false}"
	if string(raw) != want {
		t.Fatalf("unexpected json: got %s want %s", raw, want)
	}
}
