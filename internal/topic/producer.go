package topic

import (
	"context"
	"time"
)

type EventProducer interface {
	SendTopicCheck(ctx context.Context, msg TopicCheckMsg) error
	SendDeleteTopic(ctx context.Context, msg TopicDeleteMsg) error
	SendNotifyUser(ctx context.Context, msg NotifyMsg) error
}

type TopicCheckMsg struct {
	TopicID string `json:"topicId"`
}

type TopicDeleteMsg struct {
	TopicID string `json:"topicId"`
}

type NotifyMsg struct {
	TargetUserID string    `json:"targetUserId"`
	SenderUserID string    `json:"senderUserId"`
	Type         string    `json:"type"`
	Content      string    `json:"content"`
	TopicID      string    `json:"topicId"`
	CommentID    string    `json:"commentId"`
	CreatedTime  time.Time `json:"createdTime"`
}
