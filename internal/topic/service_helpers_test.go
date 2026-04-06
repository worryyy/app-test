package topic

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestPrepareTopicFormatsCreatedTime(t *testing.T) {
	ts := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	topic := &Topic{
		ID:   primitive.NewObjectIDFromTimestamp(ts),
		Imgs: nil,
	}

	svc := &Service{}
	svc.prepareTopic(topic)

	if topic.CreatedTime != "2026-03-20 12:00:00" {
		t.Fatalf("unexpected createdTime: %q", topic.CreatedTime)
	}
	if topic.Imgs == nil {
		t.Fatal("expected imgs to be normalized to non-nil slice")
	}
}
