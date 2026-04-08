package ecampus

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

type topicProducerAdapter struct {
	producer *mq.Producer
}

func newTopicProducerAdapter(producer *mq.Producer) topic.EventProducer {
	if producer == nil {
		return nil
	}
	return &topicProducerAdapter{producer: producer}
}

func (a *topicProducerAdapter) SendTopicCheck(ctx context.Context, msg topic.TopicCheckMsg) error {
	return a.producer.SendTopicCheck(ctx, mq.TopicCheckMsg(msg))
}

func (a *topicProducerAdapter) SendDeleteTopic(ctx context.Context, msg topic.TopicDeleteMsg) error {
	return a.producer.SendDeleteTopic(ctx, mq.TopicDeleteMsg(msg))
}

func (a *topicProducerAdapter) SendNotifyUser(ctx context.Context, msg topic.NotifyMsg) error {
	return a.producer.SendNotifyUser(ctx, mq.NotifyMsg(msg))
}

type userProducerAdapter struct {
	producer *mq.Producer
}

func newUserProducerAdapter(producer *mq.Producer) user.EventProducer {
	if producer == nil {
		return nil
	}
	return &userProducerAdapter{producer: producer}
}

func (a *userProducerAdapter) SendTopicUserUpdate(ctx context.Context, msg user.TopicUserUpdateMsg) error {
	return a.producer.SendUpdateTopicUser(ctx, mq.TopicUserUpdateMsg(msg))
}

func (a *userProducerAdapter) SendCommentUserUpdate(ctx context.Context, msg user.CommentUserUpdateMsg) error {
	return a.producer.SendUpdateCommentUser(ctx, mq.CommentUserUpdateMsg(msg))
}
