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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := timeField(tc.body, "sentAt")
			if got.IsZero() {
				t.Fatalf("expected non-zero time")
			}

			if tc.name == "javaLocalDateTime" {
				want := time.Date(2024, 3, 9, 16, 0, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Fatalf("unexpected java time: got %v want %v", got, want)
				}
				return
			}

			if !got.Equal(expected) {
				t.Fatalf("unexpected time: got %v want %v", got, expected)
			}
		})
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
	metadata, ok := payload["metadata"].(map[string]any)
	if !ok || metadata["kind"] != "text" {
		t.Fatalf("expected metadata to be preserved, got %v", payload["metadata"])
	}
}
