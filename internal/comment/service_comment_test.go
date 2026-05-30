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

func TestApplyCommentUserCertificationFillsUserAndParent(t *testing.T) {
	expiresAt := time.Date(2026, 5, 30, 10, 30, 0, 0, time.UTC)
	comments := []Comment{
		{
			User:   CommentUser{UserID: "11"},
			Parent: &CommentUser{UserID: "22"},
		},
	}

	applyCommentUserCertification(comments, map[int64]userRecord{
		11: {
			ID:         11,
			StuIsCheck: true,
		},
		22: {
			ID:                   22,
			StuIsCheck:           false,
			ProvisionalExpiresAt: &expiresAt,
		},
	})

	if comments[0].User.StuIsCheck == nil || !*comments[0].User.StuIsCheck {
		t.Fatalf("expected user stuIsCheck=true, got %v", comments[0].User.StuIsCheck)
	}
	if comments[0].Parent == nil || comments[0].Parent.StuIsCheck == nil {
		t.Fatal("expected parent stuIsCheck to be filled")
	}
	if *comments[0].Parent.StuIsCheck {
		t.Fatal("expected parent stuIsCheck=false to be preserved")
	}
	if comments[0].Parent.ProvisionalExpiresAt == nil || !comments[0].Parent.ProvisionalExpiresAt.Equal(expiresAt) {
		t.Fatalf("unexpected parent provisionalExpiresAt: %v", comments[0].Parent.ProvisionalExpiresAt)
	}
}
