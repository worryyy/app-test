package comment

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
)

const maxPageSize = 100

type Service struct {
	repo     *Repository
	cfg      *config.Config
	logger   *zap.Logger
	producer CommentProducer
}

type CommentProducer interface {
	SendAddComment(ctx context.Context, cmt Comment) error
	SendDeleteComment(ctx context.Context, topicID, commentID string) error
}

func NewService(
	db *gorm.DB,
	mongoDB *mongo.Database,
	_ *redis.Client,
	cfg *config.Config,
	logger *zap.Logger,
	producer CommentProducer,
) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		repo:     NewRepository(db, mongoDB),
		cfg:      cfg,
		logger:   logger,
		producer: producer,
	}
}

func (s *Service) defaultPageSize() int {
	if s.cfg != nil && s.cfg.Custom.PageSize > 0 {
		return s.cfg.Custom.PageSize
	}
	return 15
}
