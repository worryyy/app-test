package cron

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
)

var (
	activeUsersGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "campus_active_users",
		Help: "Active users in different time windows",
	}, []string{"window"})
)

type MetricsJob struct {
	rds    *redis.Client
	logger *zap.Logger
}

func NewMetricsJob(rds *redis.Client, logger *zap.Logger) *MetricsJob {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &MetricsJob{rds: rds, logger: logger}
}

func (j *MetricsJob) Run(ctx context.Context) error {
	if j.rds == nil {
		return nil
	}

	now := time.Now()
	dau, err := j.rds.PFCount(ctx, rediskey.ActiveDay(now.Format("20060102"))).Result()
	if err != nil {
		return fmt.Errorf("load dau: %w", err)
	}
	activeUsersGauge.WithLabelValues("dau").Set(float64(dau))

	weekKeys := recentDayKeys(now, 7)
	wau, err := j.rds.PFCount(ctx, weekKeys...).Result()
	if err != nil {
		return fmt.Errorf("load wau: %w", err)
	}
	activeUsersGauge.WithLabelValues("wau").Set(float64(wau))

	monthKeys := recentDayKeys(now, 30)
	mau, err := j.rds.PFCount(ctx, monthKeys...).Result()
	if err != nil {
		return fmt.Errorf("load mau: %w", err)
	}
	activeUsersGauge.WithLabelValues("mau").Set(float64(mau))
	return nil
}

func recentDayKeys(now time.Time, days int) []string {
	keys := make([]string, 0, days)
	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i).Format("20060102")
		keys = append(keys, rediskey.ActiveDay(date))
	}
	return keys
}
