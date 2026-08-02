package topicapp

import (
	"context"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusproducer"
	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusruntime"
	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/sensitive"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
)

func Run() error {
	app, err := ecampusruntime.New(true)
	if err != nil {
		return err
	}
	defer app.Close(context.Background())
	producer, err := ecampusproducer.New(app.Infra)
	if err != nil {
		return err
	}
	defer ecampusproducer.Close(app.Infra, producer)
	filter := sensitive.NewService(app.Infra.MySQL, app.Infra.Logger)
	defer filter.Close()
	svc := topic.NewService(app.Infra.MySQL, app.Infra.Mongo, app.Infra.Redis, app.Infra.Config, app.Infra.Logger)
	svc.SetProducer(producerAdapter{producer: producer})
	svc.SetSensitiveFilter(filter)
	svc.SetCapabilityChecker(app.Capabilities)
	topic.RegisterProtectedRoutes(app.ProtectedAPI(), topic.NewHandler(svc))

	consumers, err := mq.NewConsumers(
		app.Infra.RabbitMQ,
		app.Infra.Redis,
		app.Infra.Mongo,
		app.Infra.MySQL,
		app.Infra.Config,
		app.Infra.Logger,
		producer,
		filter,
	)
	if err != nil {
		return err
	}
	if err := consumers.StartTopic(); err != nil {
		_ = consumers.Close()
		return err
	}
	defer func() {
		if err := consumers.Close(); err != nil {
			app.Infra.Logger.Warn("close topic consumers failed", zap.Error(err))
		}
	}()

	return app.Run("ecampus-topic")
}

type producerAdapter struct{ producer *mq.Producer }

func (a producerAdapter) SendTopicCheck(ctx context.Context, msg topic.TopicCheckMsg) error {
	return a.producer.SendTopicCheck(ctx, mq.TopicCheckMsg(msg))
}
func (a producerAdapter) SendDeleteTopic(ctx context.Context, msg topic.TopicDeleteMsg) error {
	return a.producer.SendDeleteTopic(ctx, mq.TopicDeleteMsg(msg))
}
func (a producerAdapter) SendNotifyUser(ctx context.Context, msg topic.NotifyMsg) error {
	return a.producer.SendNotifyUser(ctx, mq.NotifyMsg{TargetUserID: msg.TargetUserID, SenderUserID: msg.SenderUserID, Type: msg.Type, Content: msg.Content, TopicID: msg.TopicID, CommentID: msg.CommentID, CreatedTime: msg.CreatedTime})
}
