package ad

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Service struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{
		db:  db,
		cfg: cfg,
	}
}

func (s *Service) defaultPageSize() int {
	if s != nil && s.cfg != nil && s.cfg.Custom.PageSize > 0 {
		return s.cfg.Custom.PageSize
	}
	return 15
}

func (s *Service) CreateAd(ctx context.Context, ad *Ad) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&Ad{}).Where("topic_id = ?", ad.TopicID).Count(&count).Error; err != nil {
		return fmt.Errorf("count ad by topic id: %w", err)
	}
	if count > 0 {
		return result.NewBizError(result.CodeFail, ad.TopicID+"的广告位已存在")
	}

	if err := s.db.WithContext(ctx).Create(ad).Error; err != nil {
		return result.NewBizError(result.CodeFail, "添加失败")
	}
	return nil
}

func (s *Service) DeleteAd(ctx context.Context, id int64) error {
	var ad Ad
	if err := s.db.WithContext(ctx).First(&ad, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result.NewBizError(result.CodeFail, fmt.Sprintf("id:%d不存在", id))
		}
		return fmt.Errorf("query ad before delete: %w", err)
	}

	res := s.db.WithContext(ctx).Delete(&Ad{}, id)
	if res.Error != nil {
		return fmt.Errorf("delete ad: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return result.NewBizError(result.CodeFail, "删除失败")
	}
	return nil
}

func (s *Service) UpdateAd(ctx context.Context, id int64, ad *Ad) error {
	var existing Ad
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result.NewBizError(result.CodeFail, "不存在")
		}
		return fmt.Errorf("query ad before update: %w", err)
	}

	if ad != nil && ad.TopicID != "" {
		var conflict Ad
		err := s.db.WithContext(ctx).Where("topic_id = ?", ad.TopicID).First(&conflict).Error
		if err == nil && conflict.ID != id {
			return result.NewBizError(result.CodeFail, "当前帖子已被其他广告关联")
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("query ad conflict by topic id: %w", err)
		}
	}

	res := s.db.WithContext(ctx).Model(&Ad{}).Where("id = ?", id).Updates(ad)
	if res.Error != nil {
		return fmt.Errorf("update ad: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return result.NewBizError(result.CodeFail, "更新失败")
	}
	return nil
}

func (s *Service) GetAd(ctx context.Context, id int64) (*Ad, error) {
	var ad Ad
	if err := s.db.WithContext(ctx).First(&ad, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get ad: %w", err)
	}
	return &ad, nil
}

func (s *Service) ListAds(ctx context.Context, page, size int) (*result.PageResult[Ad], error) {
	var total int64
	if err := s.db.WithContext(ctx).Model(&Ad{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count ads: %w", err)
	}
	var list []Ad
	if err := s.db.WithContext(ctx).Offset((page - 1) * size).Limit(size).Order("id DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list ads: %w", err)
	}
	return result.NewPage(list, total, page, size), nil
}

func (s *Service) ListAdByLevel(ctx context.Context, size int) ([]Ad, error) {
	if size <= 0 {
		size = s.defaultPageSize()
	}
	var list []Ad
	if err := s.db.WithContext(ctx).
		Where("is_ok = ?", true).
		Order("level DESC, id DESC").
		Limit(size).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list ads by level: %w", err)
	}
	return list, nil
}
