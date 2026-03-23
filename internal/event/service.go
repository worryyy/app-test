package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
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

func (s *Service) AddEvent(ctx context.Context, e *Event) error {
	if e.TriggerTime.IsZero() {
		e.TriggerTime = time.Now()
	}
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if err := s.redis.LPush(ctx, rediskey.EventKey, string(data)).Err(); err != nil {
		return fmt.Errorf("push event to redis: %w", err)
	}
	return nil
}

func (s *Service) DeleteEvent(ctx context.Context, id int64) error {
	if err := s.db.WithContext(ctx).Where("eventId = ?", id).Delete(&Event{}).Error; err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	return nil
}

func (s *Service) UpdateEvent(ctx context.Context, id int64, e *Event) error {
	if err := s.db.WithContext(ctx).Model(&Event{}).Where("eventId = ?", id).Updates(e).Error; err != nil {
		return fmt.Errorf("update event: %w", err)
	}
	return nil
}

func (s *Service) GetEvent(ctx context.Context, id int64) (*Event, error) {
	var e Event
	if err := s.db.WithContext(ctx).Where("eventId = ?", id).First(&e).Error; err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	return &e, nil
}

func (s *Service) ListEvents(ctx context.Context, page, size int, eventType string) (*result.PageResult[Event], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	q := s.db.WithContext(ctx).Model(&Event{})
	if eventType != "" {
		q = q.Where("eventType = ?", eventType)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count events: %w", err)
	}

	var list []Event
	if err := q.Offset((page - 1) * size).Limit(size).Order("eventId DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return result.NewPage(list, total, page, size), nil
}
