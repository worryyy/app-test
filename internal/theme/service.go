package theme

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (s *Service) InitCampusThemes(ctx context.Context, themes []Theme) error {
	if len(themes) == 0 {
		return nil
	}
	docs := make([]interface{}, 0, len(themes))
	for _, t := range themes {
		docs = append(docs, t)
	}
	_, err := s.themeColl().InsertMany(ctx, docs, options.InsertMany().SetOrdered(false))
	if err != nil {
		return fmt.Errorf("init campus themes: %w", err)
	}
	return nil
}

func (s *Service) ListCampusThemes(ctx context.Context) ([]Theme, error) {
	cur, err := s.themeColl().Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"name": 1}))
	if err != nil {
		return nil, fmt.Errorf("list campus themes: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close theme cursor failed", zap.Error(closeErr))
		}
	}()

	var themes []Theme
	if err := cur.All(ctx, &themes); err != nil {
		return nil, fmt.Errorf("decode campus themes: %w", err)
	}
	return themes, nil
}

func (s *Service) ListThemes(ctx context.Context, name string) ([]Theme, error) {
	filter := bson.M{}
	if name != "" {
		filter["name"] = bson.M{"$regex": name}
	}
	cur, err := s.themeColl().Find(ctx, filter, options.Find().SetSort(bson.M{"name": 1}))
	if err != nil {
		return nil, fmt.Errorf("list themes: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close theme cursor failed", zap.Error(closeErr))
		}
	}()

	var themes []Theme
	if err := cur.All(ctx, &themes); err != nil {
		return nil, fmt.Errorf("decode themes: %w", err)
	}
	return themes, nil
}

func (s *Service) AddTheme(ctx context.Context, theme *Theme) (string, error) {
	res, err := s.themeColl().InsertOne(ctx, theme)
	if err != nil {
		return "", fmt.Errorf("add theme: %w", err)
	}
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("theme id invalid")
	}
	return oid.Hex(), nil
}

func (s *Service) UpdateTheme(ctx context.Context, id string, theme *Theme) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid theme id: %w", err)
	}
	_, err = s.themeColl().UpdateByID(ctx, oid, bson.M{"$set": bson.M{
		"name":              theme.Name,
		"category_name":     theme.CategoryName,
		"needSearch":        theme.NeedSearch,
		"needSuggest":       theme.NeedSuggest,
		"suggestBasicScore": theme.SuggestBasicScore,
		"suggestNumber":     theme.SuggestNumber,
		"suggestSetName":    theme.SuggestSetName,
		"suggestType":       theme.SuggestType,
	}})
	if err != nil {
		return fmt.Errorf("update theme: %w", err)
	}
	return nil
}

func (s *Service) DeleteTheme(ctx context.Context, themeID string) error {
	oid, err := primitive.ObjectIDFromHex(themeID)
	if err != nil {
		return fmt.Errorf("invalid theme id: %w", err)
	}
	_, err = s.themeColl().DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return fmt.Errorf("delete theme: %w", err)
	}
	return nil
}

func (s *Service) UpdateNeedSearch(ctx context.Context, themeID string, needSearch bool) error {
	oid, err := primitive.ObjectIDFromHex(themeID)
	if err != nil {
		return fmt.Errorf("invalid theme id: %w", err)
	}
	_, err = s.themeColl().UpdateByID(ctx, oid, bson.M{"$set": bson.M{"needSearch": needSearch}})
	if err != nil {
		return fmt.Errorf("update need search: %w", err)
	}
	return nil
}

func (s *Service) UpdateSuggestConfig(ctx context.Context, themeID string, suggest Theme) error {
	oid, err := primitive.ObjectIDFromHex(themeID)
	if err != nil {
		return fmt.Errorf("invalid theme id: %w", err)
	}
	_, err = s.themeColl().UpdateByID(ctx, oid, bson.M{"$set": bson.M{
		"needSuggest":       suggest.NeedSuggest,
		"suggestBasicScore": suggest.SuggestBasicScore,
		"suggestNumber":     suggest.SuggestNumber,
		"suggestSetName":    suggest.SuggestSetName,
		"suggestType":       suggest.SuggestType,
	}})
	if err != nil {
		return fmt.Errorf("update suggest config: %w", err)
	}
	return nil
}

func (s *Service) themeColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_theme")
}
