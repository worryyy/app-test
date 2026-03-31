package event

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
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

func (s *Service) AddEvent(ctx context.Context, e *Event, userID int64) error {
	if e == nil {
		return result.ErrParam
	}
	if e.TriggerTime.IsZero() {
		e.TriggerTime = time.Now()
	}
	e.UserID = fmt.Sprintf("%d", userID)
	if err := s.db.WithContext(ctx).Create(e).Error; err != nil {
		return fmt.Errorf("create event: %w", err)
	}
	return nil
}

func (s *Service) DeleteEvent(ctx context.Context, id int64) error {
	res := s.db.WithContext(ctx).Where("event_id = ?", id).Delete(&Event{})
	if res.Error != nil {
		return fmt.Errorf("delete event: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return result.NewBizError(result.CodeFail, "失败")
	}
	return nil
}

func (s *Service) UpdateEvent(ctx context.Context, id int64, e *EventUpdateReq) (bool, error) {
	res := s.db.WithContext(ctx).Model(&Event{}).Where("event_id = ?", id).Updates(e)
	if res.Error != nil {
		return false, fmt.Errorf("update event: %w", res.Error)
	}
	return res.RowsAffected > 0, nil
}

func (s *Service) GetEvent(ctx context.Context, id int64) (*Event, error) {
	var e Event
	if err := s.db.WithContext(ctx).Where("event_id = ?", id).First(&e).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get event: %w", err)
	}
	return &e, nil
}

func (s *Service) ListEvents(ctx context.Context, req EventListReq) (*EventListResp, error) {
	size := req.Size
	if size <= 0 || size > s.defaultPageSize() {
		size = s.defaultPageSize()
	}
	q := s.db.WithContext(ctx).Model(&Event{})
	q = q.Where("event_id > ?", req.PrevID).Where("trigger_time >= ?", req.StartTime)
	if req.UserID != "" {
		q = q.Where("user_id = ?", req.UserID)
	}
	if req.EventType != "" {
		q = q.Where("event_type = ?", req.EventType)
	}
	if strings.TrimSpace(req.KeyWord) != "" {
		if req.UserID == "" && req.EventType == "" {
			return nil, result.NewBizError(result.CodeFail, "使用key_word必须传入 user_id 或者 event_type")
		}
		q = q.Where("event_content LIKE ?", buildKeywordLike(req.KeyWord))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count events: %w", err)
	}

	var list []Event
	if err := q.Order("event_id ASC").Limit(size).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return &EventListResp{Data: result.EnsureSlice(list), Total: total}, nil
}

func (s *Service) defaultPageSize() int {
	if s != nil && s.cfg != nil && s.cfg.Custom.PageSize > 0 {
		return s.cfg.Custom.PageSize
	}
	return 15
}

func buildKeywordLike(keyWord string) string {
	parts := strings.Fields(strings.TrimSpace(keyWord))
	if len(parts) == 0 {
		return "%"
	}
	return "%" + strings.Join(parts, "%") + "%"
}
