package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/chat"
	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/file"
	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/snowflake"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/theme"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/server"
	"github.com/Milchstrassse/Ecampus-go/internal/middleware"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	_, _ = time.LoadLocation("Asia/Shanghai")
	cfg := config.Load("configs/ecampus")
	logger, err := config.InitLogger(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if syncErr := logger.Sync(); syncErr != nil {
			logger.Warn("sync logger failed", zap.Error(syncErr))
		}
	}()

	if err := snowflake.Init(1); err != nil {
		return fmt.Errorf("init snowflake: %w", err)
	}
	db, err := config.InitMySQL(cfg)
	if err != nil {
		return err
	}
	mongoDB, err := config.InitMongo(cfg)
	if err != nil {
		return err
	}
	rds, err := config.InitRedis(cfg)
	if err != nil {
		return err
	}
	defer closeRedis(logger, rds)

	amqpConn, err := config.InitRabbitMQ(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := amqpConn.Close(); closeErr != nil {
			logger.Warn("close rabbitmq connection failed", zap.Error(closeErr))
		}
	}()

	jwtHelper := jwtutil.NewHelper(cfg.JWT, rds)
	producer, err := mq.NewProducer(amqpConn, rds, mongoDB, logger)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := producer.Close(); closeErr != nil {
			logger.Warn("close producer failed", zap.Error(closeErr))
		}
	}()

	userSvc := user.NewService(db, mongoDB, rds, cfg, logger)
	userSvc.SetProducer(newUserProducerAdapter(producer))
	topicSvc := topic.NewService(db, mongoDB, rds, cfg, logger)
	topicSvc.SetProducer(newTopicProducerAdapter(producer))
	commentSvc := comment.NewService(db, mongoDB, rds, cfg, logger, producer)
	themeSvc := theme.NewService(db, mongoDB, rds, cfg, logger)
	fileSvc := file.NewService(db, mongoDB, rds, cfg, logger)
	chatSvc := chat.NewService(db, mongoDB, rds, cfg, logger)
	schoolSvc := school.NewService(db, mongoDB, rds, cfg, logger, producer)

	userH := user.NewHandler(userSvc)
	topicH := topic.NewHandler(topicSvc)
	commentH := comment.NewHandler(commentSvc)
	themeH := theme.NewHandler(themeSvc)
	fileH := file.NewHandler(fileSvc)
	chatH := chat.NewHandler(chatSvc, userSvc, jwtHelper, rds)
	schoolH := school.NewHandler(schoolSvc)

engine := gin.New()
engine.Use(gin.Recovery())
engine.Use(middleware.CORS())

server.RegisterCommonRoutes(engine)

registerUserRoutes(engine, logger, db, jwtHelper, rds, UserHandlers{
	User:    userH,
	Topic:   topicH,
	Comment: commentH,
	Theme:   themeH,
	File:    fileH,
	Chat:    chatH,
	School:  schoolH,
})

	consumers, err := mq.NewConsumers(amqpConn, rds, mongoDB, db, cfg, logger, producer)
	if err != nil {
		return err
	}
	consumers.SetNotifyPusher(chatH.PushNotification)
	if err := consumers.Start(); err != nil {
		return err
	}
	defer func() {
		if closeErr := consumers.Close(); closeErr != nil {
			logger.Warn("close consumers failed", zap.Error(closeErr))
		}
	}()

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("start ecampus server failed", zap.Error(err))
		}
	}()
	logger.Info("ecampus started", zap.Int("port", cfg.Server.Port))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func closeRedis(logger *zap.Logger, rds *redis.Client) {
	if rds == nil {
		return
	}
	if err := rds.Close(); err != nil {
		logger.Warn("close redis failed", zap.Error(err))
	}
}
