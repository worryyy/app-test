package topic

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestPrepareTopicFormatsObjectIDTimestampInLocalTime(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.FixedZone("UTC+8", 8*60*60)
	defer func() {
		time.Local = originalLocal
	}()

	svc := &Service{}
	topic := &Topic{
		ID: primitive.NewObjectIDFromTimestamp(time.Date(2026, 4, 9, 16, 30, 0, 0, time.UTC)),
	}

	svc.prepareTopic(topic)

	if topic.CreatedTime != "2026-04-10 00:30:00" {
		t.Fatalf("unexpected createdTime: %s", topic.CreatedTime)
	}
}
