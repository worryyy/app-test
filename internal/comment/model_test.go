package comment

import (
	"encoding/json"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCommentMarshalJSONFormatsCreatedTimeAsLocalDateTime(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.FixedZone("UTC+8", 8*60*60)
	defer func() {
		time.Local = originalLocal
	}()

	// 21:30 UTC 经 UTC+8 转换后跨天到次日 05:30，覆盖跨天场景。
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

	if payload["createdTime"] != "2026-04-07 05:30:00" {
		t.Fatalf("unexpected createdTime: %v", payload["createdTime"])
	}
}

func TestCommentTopicMarshalJSONFormatsCreatedTimeAsLocalDateTime(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.FixedZone("UTC+8", 8*60*60)
	defer func() {
		time.Local = originalLocal
	}()

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

	if payload["createdTime"] != "2026-04-06 16:00:00" {
		t.Fatalf("unexpected createdTime: %v", payload["createdTime"])
	}
}
