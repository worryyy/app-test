package monitor

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
)

type Service struct {
	db      *gorm.DB
	mongoDB *mongo.Database
	redis   *redis.Client
	cfg     *config.Config
	logger  *zap.Logger
}

func NewService(db *gorm.DB, mongoDB *mongo.Database, rds *redis.Client, cfg *config.Config, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		db:      db,
		mongoDB: mongoDB,
		redis:   rds,
		cfg:     cfg,
		logger:  logger,
	}
}

func (s *Service) ListCacheNames(ctx context.Context) ([]string, error) {
	iter := s.redis.Scan(ctx, 0, "*", 500).Iterator()
	keys := make([]string, 0)
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("scan cache keys: %w", err)
	}
	return keys, nil
}

func (s *Service) GetCacheStats(ctx context.Context, cacheName string) (*CacheStats, error) {
	var size int64
	if cacheName == "" {
		v, err := s.redis.DBSize(ctx).Result()
		if err != nil {
			return nil, fmt.Errorf("query redis dbsize: %w", err)
		}
		size = v
	} else {
		iter := s.redis.Scan(ctx, 0, cacheName+"*", 500).Iterator()
		for iter.Next(ctx) {
			size++
		}
		if err := iter.Err(); err != nil {
			return nil, fmt.Errorf("scan cache by prefix: %w", err)
		}
	}

	hitCount, missCount, err := s.redisHitMiss(ctx)
	if err != nil {
		return nil, err
	}

	stats := &CacheStats{
		CacheName: cacheName,
		Size:      size,
		HitCount:  hitCount,
		MissCount: missCount,
	}
	if hitCount+missCount > 0 {
		stats.HitRate = float64(hitCount) / float64(hitCount+missCount)
	}
	return stats, nil
}

func (s *Service) redisHitMiss(ctx context.Context) (int64, int64, error) {
	info, err := s.redis.Info(ctx, "stats").Result()
	if err != nil {
		return 0, 0, fmt.Errorf("query redis info stats: %w", err)
	}

	var hitCount int64
	var missCount int64
	scanner := bufio.NewScanner(strings.NewReader(info))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "keyspace_hits:") {
			v := strings.TrimPrefix(line, "keyspace_hits:")
			n, convErr := strconv.ParseInt(v, 10, 64)
			if convErr == nil {
				hitCount = n
			}
		}
		if strings.HasPrefix(line, "keyspace_misses:") {
			v := strings.TrimPrefix(line, "keyspace_misses:")
			n, convErr := strconv.ParseInt(v, 10, 64)
			if convErr == nil {
				missCount = n
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("scan redis info: %w", err)
	}
	return hitCount, missCount, nil
}
