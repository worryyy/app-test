package chat

import "testing"

func TestSessionManagerRemoveIfSameKeepsNewSession(t *testing.T) {
	manager := NewSessionManager()
	first := &Session{UserID: 7}
	second := &Session{UserID: 7}

	manager.Set(7, first)
	manager.Set(7, second)
	manager.RemoveIfSame(7, first)

	got, ok := manager.Get(7)
	if !ok {
		t.Fatal("expected session to remain")
	}
	if got != second {
		t.Fatalf("expected latest session to remain, got %#v", got)
	}
}

func TestNewWSAuthSuccessIncludesCompatUserFields(t *testing.T) {
	payload := newWSAuthSuccess(42)

	if payload["type"] != "auth_success" {
		t.Fatalf("unexpected type: %v", payload["type"])
	}
	if payload["userId"] != "42" {
		t.Fatalf("expected userId field, got %v", payload["userId"])
	}
	if payload["user_id"] != "42" {
		t.Fatalf("expected user_id field, got %v", payload["user_id"])
	}
}
