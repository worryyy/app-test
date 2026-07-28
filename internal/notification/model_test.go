package notification

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestDocumentResponsesSupportLegacyAndNewIDs(t *testing.T) {
	createdAt := time.Date(2026, 7, 28, 8, 30, 0, 0, time.UTC)
	legacyID := primitive.NewObjectID()
	legacy := Document{ID: legacyID, Type: "TOPIC_LIKE", Content: "x", CreatedTime: createdAt}
	if got := legacy.response(); got.ID != legacyID.Hex() || got.Category != CategorySocial || got.EventType != "TOPIC_LIKE" {
		t.Fatalf("unexpected legacy response: %+v", got)
	}
	if got := legacy.legacyResponse(); got.ID != legacyID.Hex() || got.CreatedTime != "2026-07-28" {
		t.Fatalf("unexpected legacy compatibility response: %+v", got)
	}

	current := Document{ID: "123456789", Category: CategoryAcademic, EventType: "material.created", CreatedTime: createdAt}
	if got := current.response(); got.ID != "123456789" || got.CreatedTime != "2026-07-28T16:30:00+08:00" {
		t.Fatalf("unexpected current response: %+v", got)
	}
}

func TestReceiverFilterIncludesRootAndAllIdentities(t *testing.T) {
	filter := receiverFilter(100, []int64{100, 101})
	raw, err := bson.MarshalExtJSON(filter, false, false)
	if err != nil {
		t.Fatalf("marshal filter: %v", err)
	}
	value := string(raw)
	for _, expected := range []string{"receiver_root_user_id", "receiver_id", "100", "101"} {
		if !contains(value, expected) {
			t.Fatalf("filter %s does not contain %q", value, expected)
		}
	}
}

func contains(value, target string) bool {
	for i := 0; i+len(target) <= len(value); i++ {
		if value[i:i+len(target)] == target {
			return true
		}
	}
	return false
}
