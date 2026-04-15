package chat

import "testing"

func TestReverseMessages(t *testing.T) {
	messages := []Message{
		{MessageID: 3},
		{MessageID: 2},
		{MessageID: 1},
	}

	reverseMessages(messages)

	if len(messages) != 3 {
		t.Fatalf("unexpected length: %d", len(messages))
	}
	if messages[0].MessageID != 1 || messages[1].MessageID != 2 || messages[2].MessageID != 3 {
		t.Fatalf("unexpected order after reverse: %#v", messages)
	}
}
