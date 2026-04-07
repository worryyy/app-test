package comment

import (
	"encoding/json"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCommentMarshalJSONFormatsCreatedTimeAsDate(t *testing.T) {
	raw, err := json.Marshal(Comment{
		ID:          primitive.NewObjectID(),
		TopicID:     "topic-1",
		Comment:     "hi",
		CreatedTime: time.Date(2026, 4, 6, 21, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("marshal comment: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal comment json: %v", err)
	}

	if payload["createdTime"] != "2026-04-06" {
		t.Fatalf("unexpected createdTime: %v", payload["createdTime"])
	}
}

func TestCommentTopicMarshalJSONFormatsCreatedTimeAsDate(t *testing.T) {
	createdAt := time.Date(2026, 4, 6, 8, 0, 0, 0, time.UTC)
	raw, err := json.Marshal(CommentTopic{
		ID:          primitive.NewObjectID(),
		ThemeID:     "10001",
		Title:       "topic",
		CreatedTime: &createdAt,
	})
	if err != nil {
		t.Fatalf("marshal comment topic: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal comment topic json: %v", err)
	}

	if payload["createdTime"] != "2026-04-06" {
		t.Fatalf("unexpected createdTime: %v", payload["createdTime"])
	}
}
