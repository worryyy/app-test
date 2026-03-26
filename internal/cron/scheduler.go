package cron

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	robcron "github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Scheduler struct {
	cron      *robcron.Cron
	logger    *zap.Logger
	suggest   *SuggestJob
	event     *EventFlushJob
	metrics   *MetricsJob
	expDetail *ExpFlushJob
}

func NewScheduler(db *gorm.DB, mongoDB *mongo.Database, rds *redis.Client, logger *zap.Logger) *Scheduler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Scheduler{
		cron:      robcron.New(robcron.WithSeconds()),
		logger:    logger,
		suggest:   NewSuggestJob(mongoDB, rds, logger),
		event:     NewEventFlushJob(db, rds, logger),
		metrics:   NewMetricsJob(rds, logger),
		expDetail: NewExpFlushJob(db, rds, logger),
	}
}

func (s *Scheduler) Start() error {
	if s.cron == nil {
		return fmt.Errorf("cron scheduler not initialized")
	}

	_, err := s.cron.AddFunc("0 1 2 * * *", func() {
		if _, runErr := s.suggest.Generate(context.Background()); runErr != nil {
			s.logger.Error("run suggest job failed", zap.Error(runErr))
		}
	})
	if err != nil {
		return fmt.Errorf("register suggest cron: %w", err)
	}

	_, err = s.cron.AddFunc("0 2 2 * * *", func() {
		if runErr := s.suggest.CleanupOldAllRanks(context.Background()); runErr != nil {
			s.logger.Error("cleanup suggest keys failed", zap.Error(runErr))
		}
	})
	if err != nil {
		return fmt.Errorf("register suggest cleanup cron: %w", err)
	}

	_, err = s.cron.AddFunc("0 */15 * * * *", func() {
		if runErr := s.event.Run(context.Background()); runErr != nil {
			s.logger.Error("run event flush job failed", zap.Error(runErr))
		}
	})
	if err != nil {
		return fmt.Errorf("register event flush cron: %w", err)
	}

	_, err = s.cron.AddFunc("0 * * * * *", func() {
		if runErr := s.metrics.Run(context.Background()); runErr != nil {
			s.logger.Error("run metrics job failed", zap.Error(runErr))
		}
	})
	if err != nil {
		return fmt.Errorf("register metrics cron: %w", err)
	}

	_, err = s.cron.AddFunc("0 */5 * * * *", func() {
		if runErr := s.expDetail.Run(context.Background()); runErr != nil {
			s.logger.Error("run exp flush job failed", zap.Error(runErr))
		}
	})
	if err != nil {
		return fmt.Errorf("register exp flush cron: %w", err)
	}

	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() {
	if s.cron == nil {
		return
	}
	stopCtx := s.cron.Stop()
	<-stopCtx.Done()
}
