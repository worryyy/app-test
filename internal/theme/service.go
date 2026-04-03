package theme

import (
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
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
	repo   *Repository
	logger *zap.Logger
}

func NewService(db *gorm.DB, mongoDB *mongo.Database, _ *redis.Client, _ *config.Config, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		repo:   NewRepository(db, mongoDB),
		logger: logger,
	}
}

func (s *Service) InitCampusThemes(ctx context.Context) ([]CampusTheme, error) {
	for _, theme := range defaultCampusThemes {
		item := theme
		if err := s.repo.UpsertCampusTheme(ctx, &item); err != nil {
			return nil, bizerr.InternalWrap("初始化校园主题失败", err)
		}
	}
	return s.ListCampusThemes(ctx)
}

func (s *Service) ListCampusThemes(ctx context.Context) ([]CampusTheme, error) {
	themes, err := s.repo.FindCampusThemes(ctx)
	if err != nil {
		return nil, bizerr.InternalWrap("查询校园主题失败", err)
	}
	return themes, nil
}

func (s *Service) ListThemes(ctx context.Context, name string) ([]Theme, error) {
	themes, err := s.repo.FindThemes(ctx, strings.TrimSpace(name))
	if err != nil {
		return nil, bizerr.InternalWrap("查询主题列表失败", err)
	}
	return themes, nil
}

func (s *Service) AddTheme(ctx context.Context, theme *Theme) (string, error) {
	if theme == nil {
		return "", bizerr.Param(errMsgInvalidParam)
	}

	oid, err := s.repo.CreateTheme(ctx, theme)
	if err != nil {
		return "", bizerr.InternalWrap("新增主题失败", err)
	}
	return oid.Hex(), nil
}

func (s *Service) AddCampusTheme(ctx context.Context, theme *CampusTheme) (*CampusTheme, error) {
	if theme == nil || strings.TrimSpace(theme.ThemeID) == "" || strings.TrimSpace(theme.Name) == "" {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	if err := s.repo.CreateCampusTheme(ctx, theme); err != nil {
		return nil, bizerr.InternalWrap("新增校园主题失败", err)
	}
	return theme, nil
}

func (s *Service) DeleteCampusTheme(ctx context.Context, themeID string) error {
	if strings.TrimSpace(themeID) == "" {
		return bizerr.Param(errMsgInvalidParam)
	}

	deleted, err := s.repo.DeleteCampusThemeByThemeID(ctx, themeID)
	if err != nil {
		return bizerr.InternalWrap("删除校园主题失败", err)
	}
	if !deleted {
		return ErrCampusThemeNotFound
	}
	return nil
}

func (s *Service) GetThemeByID(ctx context.Context, id string) (*Theme, error) {
	oid, err := parseThemeObjectID(id)
	if err != nil {
		return nil, err
	}

	theme, err := s.repo.FindThemeByID(ctx, oid)
	if err != nil {
		return nil, bizerr.InternalWrap("查询主题失败", err)
	}
	if theme == nil {
		return nil, ErrThemeNotFound
	}
	return theme, nil
}
