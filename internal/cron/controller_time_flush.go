package cron

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/monitor"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
)

type ControllerTimeFlushJob struct {
	db     *gorm.DB
	rds    *redis.Client
	logger *zap.Logger
}

func NewControllerTimeFlushJob(db *gorm.DB, rds *redis.Client, logger *zap.Logger) *ControllerTimeFlushJob {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ControllerTimeFlushJob{db: db, rds: rds, logger: logger}
}

func (j *ControllerTimeFlushJob) Run(ctx context.Context) error {
	if j.db == nil || j.rds == nil {
		return nil
	}

	keys, err := j.scanKeys(ctx)
	if err != nil || len(keys) == 0 {
		return err
	}

	list, err := j.loadRecords(ctx, keys)
	if err != nil || len(list) == 0 {
		return err
	}

	for i := 0; i < len(list); i += 1000 {
		end := i + 1000
		if end > len(list) {
			end = len(list)
		}
		if err := j.db.WithContext(ctx).CreateInBatches(list[i:end], 1000).Error; err != nil {
			return fmt.Errorf("insert controller times: %w", err)
		}
	}
	return nil
}

func (j *ControllerTimeFlushJob) scanKeys(ctx context.Context) ([]string, error) {
	keys := make([]string, 0, 32)
	iter := j.rds.Scan(ctx, 0, rediskey.ControllerTimePrefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("scan controller time keys: %w", err)
	}
	return keys, nil
}

func (j *ControllerTimeFlushJob) loadRecords(ctx context.Context, keys []string) ([]monitor.ControllerTime, error) {
	list := make([]monitor.ControllerTime, 0, len(keys))
	for _, key := range keys {
		items, err := j.rds.LRange(ctx, key, 0, -1).Result()
		if err != nil {
			return nil, fmt.Errorf("load controller time list: %w", err)
		}
		if len(items) == 0 {
			continue
		}

		for _, item := range items {
			var record monitor.ControllerTime
			if err := json.Unmarshal([]byte(item), &record); err != nil {
				j.logger.Warn("skip invalid controller time payload", zap.Error(err), zap.String("key", key))
				continue
			}
			list = append(list, record)
		}
		if err := j.rds.Del(ctx, key).Err(); err != nil {
			return nil, fmt.Errorf("delete controller time key: %w", err)
		}
	}
	return list, nil
}
