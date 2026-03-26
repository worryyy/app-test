package chat

import (
	"context"
	"strconv"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

const maxPageSize = 100

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

func (s *Service) defaultPageSize() int {
	if s.cfg != nil && s.cfg.Custom.PageSize > 0 {
		return s.cfg.Custom.PageSize
	}
	return 15
}

func normalizePage(page, size, defaultSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = defaultSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	return page, size
}

func userIDString(userID int64) string {
	return strconv.FormatInt(userID, 10)
}

func newFail(msg string) error {
	return result.NewBizError(result.CodeFail, msg)
}

func newNotExisted(msg string) error {
	if msg == "" {
		msg = result.ErrNotExisted.Error()
	}
	return result.NewBizError(result.CodeNotExisted, msg)
}

func (s *Service) messageColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_messages")
}

func (s *Service) notifyColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_notifications")
}

func closeCursor(ctx context.Context, logger *zap.Logger, cur *mongo.Cursor, msg string) {
	if cur == nil {
		return
	}
	if err := cur.Close(ctx); err != nil && logger != nil {
		logger.Warn(msg, zap.Error(err))
	}
}
