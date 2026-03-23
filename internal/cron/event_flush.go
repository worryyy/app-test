package cron

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/event"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
)

type EventFlushJob struct {
	db     *gorm.DB
	rds    *redis.Client
	logger *zap.Logger
}

func NewEventFlushJob(db *gorm.DB, rds *redis.Client, logger *zap.Logger) *EventFlushJob {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &EventFlushJob{db: db, rds: rds, logger: logger}
}

func (j *EventFlushJob) Run(ctx context.Context) error {
	if j.db == nil || j.rds == nil {
		return nil
	}

	items, err := j.rds.LRange(ctx, rediskey.EventKey, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("load events from redis: %w", err)
	}
	if len(items) == 0 {
		return nil
	}

	events := make([]event.Event, 0, len(items))
	for _, raw := range items {
		var e event.Event
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			j.logger.Warn("skip invalid event payload", zap.Error(err))
			continue
		}
		events = append(events, e)
	}
	if len(events) == 0 {
		_, _ = j.rds.Del(ctx, rediskey.EventKey).Result()
		return nil
	}

	for i := 0; i < len(events); i += 1000 {
		end := i + 1000
		if end > len(events) {
			end = len(events)
		}
		if err := j.db.WithContext(ctx).CreateInBatches(events[i:end], 1000).Error; err != nil {
			return fmt.Errorf("batch insert events: %w", err)
		}
	}

	if err := j.rds.Del(ctx, rediskey.EventKey).Err(); err != nil {
		return fmt.Errorf("clear event redis list: %w", err)
	}
	return nil
}
