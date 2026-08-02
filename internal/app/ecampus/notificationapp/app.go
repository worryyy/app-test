package notificationapp

import (
	"context"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusproducer"
	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusruntime"
	"github.com/Milchstrassse/Ecampus-go/internal/app/notificationadapter"
	"github.com/Milchstrassse/Ecampus-go/internal/app/useradapter"
	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/notification"
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

	svc := notification.NewService(app.Infra.Mongo, app.Infra.Redis, app.Infra.Logger)
	svc.SetIdentityResolver(useradapter.Adapter{Service: app.Users})
	if err := svc.EnsureIndexes(context.Background()); err != nil {
		return err
	}
	handler := notification.NewHandler(svc, app.JWTHelper)
	notification.RegisterInfraRoutes(app.Engine, handler)
	notification.RegisterProtectedRoutes(app.ProtectedAPI(), handler)

	closeSubscriber, err := svc.StartSubscriber(context.Background(), handler.Broadcast)
	if err != nil {
		return err
	}
	defer func() {
		if err := closeSubscriber(); err != nil {
			app.Infra.Logger.Warn("close notification subscriber failed", zap.Error(err))
		}
	}()

	consumers, err := mq.NewConsumers(
		app.Infra.RabbitMQ,
		app.Infra.Redis,
		app.Infra.Mongo,
		app.Infra.MySQL,
		app.Infra.Config,
		app.Infra.Logger,
		producer,
		nil,
	)
	if err != nil {
		return err
	}
	consumers.SetNotificationWriter(notificationadapter.Adapter{Service: svc})
	if err := consumers.StartNotification(); err != nil {
		_ = consumers.Close()
		return err
	}
	if err := consumers.StartDeadLetters(); err != nil {
		_ = consumers.Close()
		return err
	}
	defer func() {
		if err := consumers.Close(); err != nil {
			app.Infra.Logger.Warn("close consumers failed", zap.Error(err))
		}
	}()

	return app.Run("ecampus-notification")
}
