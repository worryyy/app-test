package comment

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
)

type Service struct {
	db       *gorm.DB
	mongoDB  *mongo.Database
	redis    *redis.Client
	cfg      *config.Config
	logger   *zap.Logger
	producer CommentProducer
}

type CommentProducer interface {
	SendAddComment(ctx context.Context, cmt Comment) error
	SendDeleteComment(ctx context.Context, topicID, commentID string) error
}

func NewService(db *gorm.DB, mongoDB *mongo.Database, rds *redis.Client, cfg *config.Config, logger *zap.Logger, producer CommentProducer) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		db:       db,
		mongoDB:  mongoDB,
		redis:    rds,
		cfg:      cfg,
		logger:   logger,
		producer: producer,
	}
}

func (s *Service) commentColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_comment")
}

func (s *Service) defaultPageSize() int {
	if s.cfg != nil && s.cfg.Custom.PageSize > 0 {
		return s.cfg.Custom.PageSize
	}
	return 15
}
