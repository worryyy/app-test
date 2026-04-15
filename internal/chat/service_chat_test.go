package chat

import "testing"

func TestResolveInitConversationIDPrefersProvidedID(t *testing.T) {
	got := resolveInitConversationID(map[string]any{
		"id":             "conversation-from-id",
		"conversationId": "conversation-from-alias",
	})

	if got != "conversation-from-id" {
		t.Fatalf("expected id field to win, got %q", got)
	}
}

func TestResolveInitConversationIDSupportsConversationIDAlias(t *testing.T) {
	got := resolveInitConversationID(map[string]any{
		"conversationId": "conversation-42",
	})

	if got != "conversation-42" {
		t.Fatalf("expected conversationId alias to be used, got %q", got)
	}
}

func TestResolveInitConversationIDGeneratesWhenMissing(t *testing.T) {
	got := resolveInitConversationID(map[string]any{
		"receiverId": "42",
		"content":    "hello",
	})

	if got == "" {
		t.Fatal("expected generated conversation id when request does not provide one")
	}
}
