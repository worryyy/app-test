package mq

import (
	"context"

	"go.uber.org/zap"
)

type Producer struct {
	logger *zap.Logger
}

func NewProducer(logger *zap.Logger) *Producer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Producer{logger: logger}
}

func (p *Producer) SendTopicCheck(ctx context.Context, msg TopicCheckMsg) error {
	_ = ctx
	p.logger.Info("mq send topic check", zap.String("topicID", msg.TopicID))
	return nil
}

func (p *Producer) SendAddComment(ctx context.Context, msg AddCommentMsg) error {
	_ = ctx
	p.logger.Info("mq send add comment")
	return nil
}

func (p *Producer) SendAddTopicSearch(ctx context.Context, msg AddTopicSearchMsg) error {
	_ = ctx
	p.logger.Info("mq send add topic search", zap.String("topicID", msg.TopicID))
	return nil
}

func (p *Producer) SendUpdateTopicSearch(ctx context.Context, msg AddTopicSearchMsg) error {
	_ = ctx
	p.logger.Info("mq send update topic search", zap.String("topicID", msg.TopicID))
	return nil
}

func (p *Producer) SendDelTopicSearch(ctx context.Context, msg AddTopicSearchMsg) error {
	_ = ctx
	p.logger.Info("mq send del topic search", zap.String("topicID", msg.TopicID))
	return nil
}

func (p *Producer) SendGetCourse(ctx context.Context, msg CourseMsg) error {
	_ = ctx
	p.logger.Info("mq send get course", zap.Int64("userID", msg.UserID), zap.String("term", msg.Term))
	return nil
}

func (p *Producer) SendNotifyUser(ctx context.Context, msg NotifyMsg) error {
	_ = ctx
	p.logger.Info("mq send notify", zap.String("targetUserID", msg.TargetUserID), zap.String("type", msg.Type))
	return nil
}
