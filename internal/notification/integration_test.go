package notification

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestRedisBroadcastCrossesServiceInstances(t *testing.T) {
	server := miniredis.RunT(t)
	publisherClient := redis.NewClient(&redis.Options{Addr: server.Addr()})
	subscriberClient := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = publisherClient.Close()
		_ = subscriberClient.Close()
	})
	publisher := NewService(nil, publisherClient, nil)
	subscriber := NewService(nil, subscriberClient, nil)
	received := make(chan Broadcast, 1)
	closeSubscriber, err := subscriber.StartSubscriber(context.Background(), func(event Broadcast) { received <- event })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = closeSubscriber() })
	document := &Document{ID: "1001", ReceiverRootUserID: 99, ReceiverID: "99", Category: CategorySystem, EventType: "system.test", CreatedTime: time.Now()}
	publisher.publish(context.Background(), document)
	select {
	case event := <-received:
		if event.RootUserID != 99 || event.Notification.ID != "1001" || event.Notification.Category != CategorySystem {
			t.Fatalf("broadcast = %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for redis broadcast")
	}
}

func TestMongoCompatibilityAndEventIdempotency(t *testing.T) {
	uri := os.Getenv("ECAMPUS_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("ECAMPUS_TEST_MONGO_URI is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	database := client.Database("ecampus_notification_test_" + strconv.FormatInt(time.Now().UnixNano(), 10))
	t.Cleanup(func() { _ = database.Drop(context.Background()) })
	repository := NewRepository(database)
	if err := repository.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	legacyID := primitive.NewObjectID()
	legacy := &Document{ID: legacyID, ReceiverID: "101", Type: "TOPIC_LIKE", Content: "legacy", CreatedTime: now.Add(-time.Minute), IsRead: false}
	current := &Document{ID: "2001", EventID: "event-1", ReceiverRootUserID: 100, ReceiverID: "100", Category: CategoryAcademic, EventType: "material.created", Content: "current", CreatedTime: now, IsRead: false}
	if inserted, err := repository.Insert(ctx, legacy); err != nil || !inserted {
		t.Fatalf("insert legacy: inserted=%t err=%v", inserted, err)
	}
	if inserted, err := repository.Insert(ctx, current); err != nil || !inserted {
		t.Fatalf("insert current: inserted=%t err=%v", inserted, err)
	}
	duplicate := *current
	duplicate.ID = "2002"
	if inserted, err := repository.Insert(ctx, &duplicate); err != nil || inserted {
		t.Fatalf("duplicate event: inserted=%t err=%v", inserted, err)
	}
	documents, total, err := repository.List(ctx, 100, []int64{101}, "", 1, 10)
	if err != nil || total != 2 || len(documents) != 2 {
		t.Fatalf("combined list total=%d len=%d err=%v", total, len(documents), err)
	}
	social, total, err := repository.List(ctx, 100, []int64{101}, CategorySocial, 1, 10)
	if err != nil || total != 1 || len(social) != 1 || social[0].ID != legacyID {
		t.Fatalf("legacy social list total=%d data=%+v err=%v", total, social, err)
	}
	if ok, err := repository.MarkOneRead(ctx, 100, []int64{101}, legacyID.Hex()); err != nil || !ok {
		t.Fatalf("mark legacy read: ok=%t err=%v", ok, err)
	}
	counts, err := repository.UnreadCategories(ctx, 100, []int64{101})
	if err != nil || counts[CategoryAcademic] != 1 || counts[CategorySocial] != 0 {
		t.Fatalf("unread counts=%+v err=%v", counts, err)
	}
}
