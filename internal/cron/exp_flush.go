package cron

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/level"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
)

type ExpFlushJob struct {
	db     *gorm.DB
	rds    *redis.Client
	logger *zap.Logger
}

func NewExpFlushJob(db *gorm.DB, rds *redis.Client, logger *zap.Logger) *ExpFlushJob {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ExpFlushJob{db: db, rds: rds, logger: logger}
}

func (j *ExpFlushJob) Run(ctx context.Context) error {
	if j.db == nil || j.rds == nil {
		return nil
	}
	items, err := j.rds.LRange(ctx, rediskey.ExpDetailKey, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("load exp details from redis: %w", err)
	}
	if len(items) == 0 {
		return nil
	}

	list := make([]level.ExpDetail, 0, len(items))
	for _, raw := range items {
		var detail level.ExpDetail
		if err := json.Unmarshal([]byte(raw), &detail); err != nil {
			j.logger.Warn("skip invalid exp detail payload", zap.Error(err))
			continue
		}
		list = append(list, detail)
	}
	if len(list) == 0 {
		_, _ = j.rds.Del(ctx, rediskey.ExpDetailKey).Result()
		return nil
	}

	for i := 0; i < len(list); i += 1000 {
		end := i + 1000
		if end > len(list) {
			end = len(list)
		}
		if err := j.db.WithContext(ctx).CreateInBatches(list[i:end], 1000).Error; err != nil {
			return fmt.Errorf("batch insert exp details: %w", err)
		}
	}

	if err := j.rds.Del(ctx, rediskey.ExpDetailKey).Err(); err != nil {
		return fmt.Errorf("clear exp detail list: %w", err)
	}
	return nil
}
