package support

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Service struct {
	mongoDB *mongo.Database
	logger  *zap.Logger
}

func NewService(mongoDB *mongo.Database, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		mongoDB: mongoDB,
		logger:  logger,
	}
}

func (s *Service) AddSupport(ctx context.Context, support *FrontendSupport) (string, error) {
	count, err := s.mongoDB.Collection("campus_frontend_support").CountDocuments(ctx, bson.M{"key": support.Key})
	if err != nil {
		return "", fmt.Errorf("count support by key: %w", err)
	}
	if count > 0 {
		return "", result.NewBizError(result.CodeFail, "key = "+support.Key+"has existed")
	}

	res, err := s.mongoDB.Collection("campus_frontend_support").InsertOne(ctx, support)
	if err != nil {
		return "", fmt.Errorf("add support: %w", err)
	}
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("support id invalid")
	}
	return oid.Hex(), nil
}

func (s *Service) UpdateSupport(ctx context.Context, support *FrontendSupport) error {
	if support == nil {
		return nil
	}
	if support.ID.IsZero() {
		update := bson.M{"val": support.Val}
		if support.KeyDesc != "" {
			update["keyDesc"] = support.KeyDesc
		}
		res, err := s.mongoDB.Collection("campus_frontend_support").UpdateOne(ctx, bson.M{"key": support.Key}, bson.M{"$set": update})
		if err != nil {
			return fmt.Errorf("update support by key: %w", err)
		}
		if res.MatchedCount == 0 {
			return result.NewBizError(result.CodeFail, support.Key+" not existed")
		}
		return nil
	}
	_, err := s.mongoDB.Collection("campus_frontend_support").UpdateByID(ctx, support.ID, bson.M{"$set": bson.M{
		"key":     support.Key,
		"val":     support.Val,
		"keyDesc": support.KeyDesc,
	}})
	if err != nil {
		return fmt.Errorf("update support: %w", err)
	}
	return nil
}

func (s *Service) DeleteSupport(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid support id: %w", err)
	}
	if _, err := s.mongoDB.Collection("campus_frontend_support").DeleteOne(ctx, bson.M{"_id": oid}); err != nil {
		return fmt.Errorf("delete support: %w", err)
	}
	return nil
}

func (s *Service) ListSupport(ctx context.Context) ([]FrontendSupport, error) {
	cur, err := s.mongoDB.Collection("campus_frontend_support").Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"_id": -1}))
	if err != nil {
		return nil, fmt.Errorf("list support: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close support cursor failed", zap.Error(closeErr))
		}
	}()

	var list []FrontendSupport
	if err := cur.All(ctx, &list); err != nil {
		return nil, fmt.Errorf("decode support: %w", err)
	}
	return list, nil
}

func (s *Service) GetSupportByKey(ctx context.Context, key string) (*FrontendSupport, error) {
	var support FrontendSupport
	err := s.mongoDB.Collection("campus_frontend_support").FindOne(ctx, bson.M{"key": key}).Decode(&support)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, result.ErrNotExisted
		}
		return nil, fmt.Errorf("get support by key: %w", err)
	}
	return &support, nil
}
