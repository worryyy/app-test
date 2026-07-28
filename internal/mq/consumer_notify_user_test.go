package mq

import (
	"context"
	"encoding/json"
	"testing"
)

type notificationWriterStub struct {
	calls int
	last  NotifyMsg
}

func (s *notificationWriterStub) PersistLegacyNotification(_ context.Context, msg NotifyMsg) error {
	s.calls++
	s.last = msg
	return nil
}

func TestHandleNotifyUserDelegatesPersistence(t *testing.T) {
	writer := &notificationWriterStub{}
	consumer := &Consumers{notificationWriter: writer}
	payload, err := json.Marshal(NotifyMsg{EventID: 9, TargetUserID: "100", Type: "TOPIC_LIKE", Content: "liked"})
	if err != nil {
		t.Fatal(err)
	}
	if err := consumer.handleNotifyUser(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	if writer.calls != 1 || writer.last.EventID != 9 || writer.last.TargetUserID != "100" {
		t.Fatalf("writer = %+v", writer)
	}
	if err := consumer.handleNotifyUser(context.Background(), json.RawMessage(`{"targetUserId":""}`)); err != nil {
		t.Fatal(err)
	}
	if writer.calls != 1 {
		t.Fatalf("empty target should not call writer: %d", writer.calls)
	}
}
