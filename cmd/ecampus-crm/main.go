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

	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/file"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/snowflake"
	"github.com/Milchstrassse/Ecampus-go/internal/school"
	"github.com/Milchstrassse/Ecampus-go/internal/theme"
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
	cfg := config.Load("configs/ecampus-crm")
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

	jwtHelper := jwtutil.NewHelper(cfg.JWT, rds)
	userSvc := user.NewService(db, mongoDB, rds, cfg, logger)
	commentSvc := comment.NewService(db, mongoDB, rds, cfg, logger, nil)
	themeSvc := theme.NewService(db, mongoDB, rds, cfg, logger)
	fileSvc := file.NewService(db, mongoDB, rds, cfg, logger)
	schoolSvc := school.NewService(db, mongoDB, rds, cfg, logger, nil)

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.CORS())

	server.RegisterCommonRoutes(engine)

	registerAdminRoutes(engine, logger, db, jwtHelper, rds, AdminHandlers{
		User:    user.NewAdminHandler(userSvc),
		Comment: comment.NewAdminHandler(commentSvc),
		Theme:   theme.NewAdminHandler(themeSvc),
		File:    file.NewAdminHandler(fileSvc),
		School:  school.NewAdminHandler(schoolSvc),
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("start ecampus-crm server failed", zap.Error(err))
		}
	}()
	logger.Info("ecampus-crm started", zap.Int("port", cfg.Server.Port))

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
