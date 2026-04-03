package main

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

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
