package theme

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"

)

var defaultCampusThemes = []CampusTheme{
	{Name: "日常", ThemeID: "10001"},
	{Name: "美食", ThemeID: "20001"},
	{Name: "树洞", ThemeID: "30001"},
	{Name: "二手", ThemeID: "40001"},
	{Name: "学习", ThemeID: "50001"},
	{Name: "搭子", ThemeID: "60001"},
	{Name: "求助", ThemeID: "70001"},
}

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

func (s *Service) InitCampusThemes(ctx context.Context) ([]CampusTheme, error) {
	for _, theme := range defaultCampusThemes {
		if _, err := s.campusThemeColl().UpdateOne(
			ctx,
			bson.M{"themeId": theme.ThemeID},
			bson.M{"$setOnInsert": bson.M{
				"name":    theme.Name,
				"themeId": theme.ThemeID,
			}},
			options.Update().SetUpsert(true),
		); err != nil {
			return nil, fmt.Errorf("init campus themes: %w", err)
		}
	}
	return s.ListCampusThemes(ctx)
}

func (s *Service) ListCampusThemes(ctx context.Context) ([]CampusTheme, error) {
	cur, err := s.campusThemeColl().Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"_id": 1}))
	if err != nil {
		return nil, fmt.Errorf("list campus themes: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close theme cursor failed", zap.Error(closeErr))
		}
	}()

	var themes []CampusTheme
	if err := cur.All(ctx, &themes); err != nil {
		return nil, fmt.Errorf("decode campus themes: %w", err)
	}
	return themes, nil
}

func (s *Service) ListThemes(ctx context.Context, name string) ([]Theme, error) {
	filter := bson.M{}
	if strings.TrimSpace(name) != "" {
		filter["name"] = name
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

func (s *Service) AddCampusTheme(ctx context.Context, theme *CampusTheme) (*CampusTheme, error) {
	if theme == nil {
		return nil, result.ErrParam
	}

	res, err := s.campusThemeColl().InsertOne(ctx, theme)
	if err != nil {
		return nil, fmt.Errorf("add campus theme: %w", err)
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		theme.ID = oid
	}
	return theme, nil
}

func (s *Service) DeleteCampusTheme(ctx context.Context, themeID string) (bool, error) {
	res, err := s.campusThemeColl().DeleteOne(ctx, bson.M{"themeId": themeID})
	if err != nil {
		return false, fmt.Errorf("delete campus theme: %w", err)
	}
	return res.DeletedCount > 0, nil
}

func (s *Service) GetThemeByID(ctx context.Context, id string) (*Theme, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid theme id: %w", err)
	}

	var theme Theme
	if err := s.themeColl().FindOne(ctx, bson.M{"_id": oid}).Decode(&theme); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, result.NewBizError(result.CodeNotExisted, "资源不存在")
		}
		return nil, fmt.Errorf("get theme: %w", err)
	}
	return &theme, nil
}

func (s *Service) themeColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_theme")
}

func (s *Service) campusThemeColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_theme_id")
}
