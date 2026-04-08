package chat

import (
	"encoding/json"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNormalizeNotificationPayloadSupportsTimeValue(t *testing.T) {
	createdAt := time.UnixMilli(1710000000000).UTC()
	notification, err := normalizeNotificationPayload(map[string]any{
		"_id":          primitive.NewObjectID(),
		"receiver_id":  "42",
		"sender_id":    "7",
		"type":         "TOPIC_LIKE",
		"content":      "x",
		"topic_id":     "t1",
		"comment_id":   "c1",
		"created_time": createdAt,
		"is_read":      true,
	})
	if err != nil {
		t.Fatalf("normalize notification payload: %v", err)
	}

	if !notification.CreatedTime.Equal(createdAt) {
		t.Fatalf("unexpected created time: got %v want %v", notification.CreatedTime, createdAt)
	}
	if !notification.IsRead {
		t.Fatalf("expected notification to stay read")
	}
}

func TestTimeFieldSupportsCommonChatTimeFormats(t *testing.T) {
	expected := time.UnixMilli(1710000000000).UTC()

	cases := []struct {
		name string
		body map[string]any
	}{
		{
			name: "time.Time",
			body: map[string]any{"sentAt": expected},
		},
		{
			name: "primitive.DateTime",
			body: map[string]any{"sentAt": primitive.NewDateTimeFromTime(expected)},
		},
		{
			name: "unixMillisString",
			body: map[string]any{"sentAt": "1710000000000"},
		},
		{
			name: "javaLocalDateTime",
			body: map[string]any{"sentAt": "2024-03-09T16:00:00"},
		},
		{
			name: "formattedDate",
			body: map[string]any{"sentAt": "2024-03-09"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := timeField(tc.body, "sentAt")
			if got.IsZero() {
				t.Fatalf("expected non-zero time")
			}

			switch tc.name {
			case "javaLocalDateTime":
				want := time.Date(2024, 3, 9, 16, 0, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Fatalf("unexpected java time: got %v want %v", got, want)
				}
			case "formattedDate":
				want := time.Date(2024, 3, 9, 0, 0, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Fatalf("unexpected formatted date: got %v want %v", got, want)
				}
			default:
				if !got.Equal(expected) {
					t.Fatalf("unexpected time: got %v want %v", got, expected)
				}
			}
		})
	}
}

func TestConversationMarshalFormatsDates(t *testing.T) {
	raw, err := json.Marshal(Conversation{
		ID:                  "c1",
		Type:                1,
		LastMessageContent:  "hi",
		LastMessageSenderID: "7",
		LastMessageSentAt:   time.Date(2024, 3, 9, 16, 0, 0, 0, time.UTC),
		CreatedAt:           time.Date(2024, 3, 1, 8, 0, 0, 0, time.UTC),
		UpdatedAt:           time.Date(2024, 3, 10, 9, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("marshal conversation: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal conversation payload: %v", err)
	}

	if payload["lastMessageSentAt"] != "2024-03-09" {
		t.Fatalf("expected formatted lastMessageSentAt, got %v", payload["lastMessageSentAt"])
	}
	if payload["createdAt"] != "2024-03-01" {
		t.Fatalf("expected formatted createdAt, got %v", payload["createdAt"])
	}
	if payload["updatedAt"] != "2024-03-10" {
		t.Fatalf("expected formatted updatedAt, got %v", payload["updatedAt"])
	}
}

func TestConversationMemberMarshalFormatsDate(t *testing.T) {
	lastReadMessageID := int64(123)
	raw, err := json.Marshal(ConversationMember{
		ConversationID:    "c1",
		UserID:            "7",
		LastReadMessageID: &lastReadMessageID,
		UnreadCount:       2,
		CreatedAt:         time.Date(2024, 3, 11, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("marshal conversation member: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal conversation member payload: %v", err)
	}

	if payload["createdAt"] != "2024-03-11" {
		t.Fatalf("expected formatted createdAt, got %v", payload["createdAt"])
	}
}

func TestMessageMarshalIncludesJavaCompatFields(t *testing.T) {
	messageType := 2
	raw, err := json.Marshal(Message{
		MessageID:      1,
		ConversationID: "9001",
		ReceiverID:     "42",
		SenderID:       "7",
		Content:        "hi",
		MessageType:    &messageType,
		SentAt:         time.Unix(1710000000, 0).UTC(),
		Metadata: map[string]any{
			"kind": "text",
		},
		HandleType: "INIT",
	})
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal message payload: %v", err)
	}

	if payload["handleType"] != "INIT" {
		t.Fatalf("expected handleType to be present, got %v", payload["handleType"])
	}
	if payload["messageType"] != float64(2) {
		t.Fatalf("expected messageType to be present, got %v", payload["messageType"])
	}
	if payload["sentAt"] != "2024-03-09" {
		t.Fatalf("expected formatted sentAt, got %v", payload["sentAt"])
	}

	metadata, ok := payload["metadata"].(map[string]any)
	if !ok || metadata["kind"] != "text" {
		t.Fatalf("expected metadata to be preserved, got %v", payload["metadata"])
	}
}

func TestNotificationMarshalFormatsDate(t *testing.T) {
	notification := Notification{
		ID:          primitive.NewObjectID(),
		ReceiverID:  "42",
		SenderID:    "7",
		Type:        "TOPIC_LIKE",
		Content:     "x",
		TopicID:     "t1",
		CommentID:   "c1",
		CreatedTime: time.Date(2024, 3, 12, 9, 0, 0, 0, time.UTC),
		IsRead:      true,
	}

	raw, err := json.Marshal(notification)
	if err != nil {
		t.Fatalf("marshal notification: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal notification payload: %v", err)
	}

	if payload["id"] != notification.ID.Hex() {
		t.Fatalf("expected notification id to be hex string, got %v", payload["id"])
	}
	if payload["createdTime"] != "2024-03-12" {
		t.Fatalf("expected formatted createdTime, got %v", payload["createdTime"])
	}
}
