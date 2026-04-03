package main

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
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

func (a *topicProducerAdapter) SendUpdateTopicSearch(ctx context.Context, msg topic.TopicSearchMsg) error {
	return a.producer.SendUpdateTopicSearch(ctx, mq.AddTopicSearchMsg(msg))
}

func (a *topicProducerAdapter) SendDeleteTopicSearch(ctx context.Context, msg topic.TopicSearchMsg) error {
	return a.producer.SendDelTopicSearch(ctx, mq.AddTopicSearchMsg(msg))
}

func (a *topicProducerAdapter) SendDeleteTopic(ctx context.Context, msg topic.TopicDeleteMsg) error {
	return a.producer.SendDeleteTopic(ctx, mq.TopicDeleteMsg(msg))
}

func (a *topicProducerAdapter) SendNotifyUser(ctx context.Context, msg topic.NotifyMsg) error {
	return a.producer.SendNotifyUser(ctx, mq.NotifyMsg(msg))
}
