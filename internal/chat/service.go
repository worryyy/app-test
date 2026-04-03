package chat

import (
	"strconv"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
)

const maxPageSize = 100

type Service struct {
	repo   *Repository
	redis  *redis.Client
	cfg    *config.Config
	logger *zap.Logger
}

func NewService(db *gorm.DB, mongoDB *mongo.Database, rds *redis.Client, cfg *config.Config, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		repo:   NewRepository(db, mongoDB),
		redis:  rds,
		cfg:    cfg,
		logger: logger,
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
