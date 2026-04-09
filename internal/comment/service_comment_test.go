package comment

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestObjectIDTimestampUsesLocalDateAfterConversion(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.FixedZone("UTC+8", 8*60*60)
	defer func() {
		time.Local = originalLocal
	}()

	createdAt := primitive.NewObjectIDFromTimestamp(time.Date(2026, 4, 9, 16, 30, 0, 0, time.UTC)).Timestamp().Local()
	if got := formatCommentDate(createdAt); got != "2026-04-10" {
		t.Fatalf("unexpected local date: %s", got)
	}
}
