package commentapp

import (
	"context"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/app/commentproducer"
	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusproducer"
	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusruntime"
	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/sensitive"
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
	svc := comment.NewService(app.Infra.MySQL, app.Infra.Mongo, app.Infra.Redis, app.Infra.Config, app.Infra.Logger, commentproducer.New(producer))
	svc.SetSensitiveFilter(filter)
	svc.SetCapabilityChecker(app.Capabilities)
	comment.RegisterProtectedRoutes(app.ProtectedAPI(), comment.NewHandler(svc))

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
	if err := consumers.StartComment(); err != nil {
		_ = consumers.Close()
		return err
	}
	defer func() {
		if err := consumers.Close(); err != nil {
			app.Infra.Logger.Warn("close comment consumers failed", zap.Error(err))
		}
	}()

	return app.Run("ecampus-comment")
}
