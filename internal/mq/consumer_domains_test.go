package mq

import "testing"

func TestDomainConsumersCoverEveryQueueOnce(t *testing.T) {
	consumers := &Consumers{}
	groups := []map[string]any{
		keys(consumers.topicHandlers()),
		keys(consumers.commentHandlers()),
		keys(consumers.notificationHandlers()),
		keys(consumers.deadLetterHandlers()),
	}
	want := []string{
		QueueTopicCheck,
		QueueTopicUpdate,
		QueueTopicDelete,
		QueueCommentAdd,
		QueueCommentUpdate,
		QueueCommentDelete,
		QueueNotifyUser,
		QueueDie,
	}
	seen := make(map[string]int, len(want))
	for _, group := range groups {
		for queue := range group {
			seen[queue]++
		}
	}
	for _, queue := range want {
		if seen[queue] != 1 {
			t.Fatalf("queue %q is assigned %d times", queue, seen[queue])
		}
		delete(seen, queue)
	}
	if len(seen) != 0 {
		t.Fatalf("unexpected queues are assigned: %v", seen)
	}
}

func keys[T any](values map[string]T) map[string]any {
	result := make(map[string]any, len(values))
	for key := range values {
		result[key] = nil
	}
	return result
}
