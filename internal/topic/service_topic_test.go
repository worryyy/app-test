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

func TestApplyTopicUserCertificationFillsFalseAndExpiry(t *testing.T) {
	expiresAt := time.Date(2026, 5, 30, 10, 30, 0, 0, time.UTC)
	topics := []Topic{
		{UserID: "42"},
		{UserID: "invalid"},
	}

	applyTopicUserCertification(topics, map[int64]topicAuthor{
		42: {
			ID:                   42,
			StuIsCheck:           false,
			ProvisionalExpiresAt: &expiresAt,
		},
	})

	if topics[0].StuIsCheck == nil {
		t.Fatal("expected stuIsCheck to be filled")
	}
	if *topics[0].StuIsCheck {
		t.Fatal("expected stuIsCheck=false to be preserved")
	}
	if topics[0].ProvisionalExpiresAt == nil || !topics[0].ProvisionalExpiresAt.Equal(expiresAt) {
		t.Fatalf("unexpected provisionalExpiresAt: %v", topics[0].ProvisionalExpiresAt)
	}
	if topics[1].StuIsCheck != nil || topics[1].ProvisionalExpiresAt != nil {
		t.Fatalf("expected invalid user id to be skipped, got %+v", topics[1])
	}
}
