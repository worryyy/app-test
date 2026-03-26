package other

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func listMongoPage[T any](ctx context.Context, coll *mongo.Collection, filter bson.M, sort interface{}, page, size int) (*result.CusPage[T], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count mongo docs: %w", err)
	}
	opts := options.Find().
		SetSkip(int64((page - 1) * size)).
		SetLimit(int64(size)).
		SetSort(sort)
	cur, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find mongo docs: %w", err)
	}

	var list []T
	if err := cur.All(ctx, &list); err != nil {
		if closeErr := cur.Close(ctx); closeErr != nil {
			return nil, fmt.Errorf("close mongo cursor after decode failure: %w", closeErr)
		}
		return nil, fmt.Errorf("decode mongo docs: %w", err)
	}
	if closeErr := cur.Close(ctx); closeErr != nil {
		return nil, fmt.Errorf("close mongo cursor: %w", closeErr)
	}
	return result.NewCusPage(list, total, page, size), nil
}
