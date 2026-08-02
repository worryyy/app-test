package ecampusproducer

import (
	"github.com/Milchstrassse/Ecampus-go/internal/app/bootstrap"
	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"go.uber.org/zap"
)

func New(infra *bootstrap.Infra) (*mq.Producer, error) {
	return mq.NewProducer(infra.RabbitMQ, infra.Redis, infra.Mongo, infra.Logger)
}

func Close(infra *bootstrap.Infra, producer *mq.Producer) {
	if producer == nil {
		return
	}
	if err := producer.Close(); err != nil && infra != nil && infra.Logger != nil {
		infra.Logger.Warn("close producer failed", zap.Error(err))
	}
}
