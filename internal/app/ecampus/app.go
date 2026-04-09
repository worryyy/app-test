package ecampus

import (
	"context"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/app/bootstrap"
	"github.com/Milchstrassse/Ecampus-go/internal/chat"
	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/file"
	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/sensitive"
	"github.com/Milchstrassse/Ecampus-go/internal/theme"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
	"go.uber.org/zap"
)

func Run() error {
	infra, err := bootstrap.LoadInfrastructure(bootstrap.Options{
		ConfigDir:     "configs/ecampus",
		WithRabbitMQ:  true,
		SnowflakeNode: 1,
	})
	if err != nil {
		return err
	}
	defer infra.Close(context.Background())

	jwtHelper := jwtutil.NewHelper(infra.Config.JWT, infra.Redis)
	producer, err := mq.NewProducer(infra.RabbitMQ, infra.Redis, infra.Mongo, infra.Logger)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := producer.Close(); closeErr != nil {
			infra.Logger.Warn("close producer failed", zap.Error(closeErr))
		}
	}()

	userSvc := user.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	userSvc.SetProducer(newUserProducerAdapter(producer))
	topicSvc := topic.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	topicSvc.SetProducer(newTopicProducerAdapter(producer))
	commentSvc := comment.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger, producer)
	sensitiveSvc := sensitive.NewService(infra.MySQL, infra.Logger)
	topicSvc.SetSensitiveFilter(sensitiveSvc)
	commentSvc.SetSensitiveFilter(sensitiveSvc)
	themeSvc := theme.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	fileSvc := file.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	chatSvc := chat.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger)
	schoolSvc := school.NewService(infra.MySQL, infra.Mongo, infra.Redis, infra.Config, infra.Logger, producer)

	userH := user.NewHandler(userSvc)
	topicH := topic.NewHandler(topicSvc)
	commentH := comment.NewHandler(commentSvc)
	themeH := theme.NewHandler(themeSvc)
	fileH := file.NewHandler(fileSvc)
	chatH := chat.NewHandler(chatSvc, userSvc, jwtHelper, infra.Redis)
	schoolH := school.NewHandler(schoolSvc)

	engine := bootstrap.NewEngine()
	registerRoutes(engine, infra.Logger, infra.MySQL, jwtHelper, infra.Redis, userHandlers{
		User:    userH,
		Topic:   topicH,
		Comment: commentH,
		Theme:   themeH,
		File:    fileH,
		Chat:    chatH,
		School:  schoolH,
	})

	consumers, err := mq.NewConsumers(infra.RabbitMQ, infra.Redis, infra.Mongo, infra.MySQL, infra.Config, infra.Logger, producer, sensitiveSvc)
	if err != nil {
		return err
	}
	consumers.SetNotifyPusher(chatH.PushNotification)
	if err := consumers.Start(); err != nil {
		_ = consumers.Close()
		return err
	}
	defer func() {
		if closeErr := consumers.Close(); closeErr != nil {
			infra.Logger.Warn("close consumers failed", zap.Error(closeErr))
		}
	}()

	server := bootstrap.NewHTTPServer(infra.Config.Server.Port, engine)
	return bootstrap.RunHTTPServer(server, infra.Logger, "ecampus", 10*time.Second)
}
