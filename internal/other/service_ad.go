package other

import (
	"context"
	"fmt"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) CreateAd(ctx context.Context, ad *Ad) error {
	if err := s.db.WithContext(ctx).Create(ad).Error; err != nil {
		return fmt.Errorf("create ad: %w", err)
	}
	return nil
}

func (s *Service) DeleteAd(ctx context.Context, id int64) error {
	if err := s.db.WithContext(ctx).Delete(&Ad{}, id).Error; err != nil {
		return fmt.Errorf("delete ad: %w", err)
	}
	return nil
}

func (s *Service) UpdateAd(ctx context.Context, id int64, ad *Ad) error {
	if err := s.db.WithContext(ctx).Model(&Ad{}).Where("id = ?", id).Updates(ad).Error; err != nil {
		return fmt.Errorf("update ad: %w", err)
	}
	return nil
}

func (s *Service) GetAd(ctx context.Context, id int64) (*Ad, error) {
	var ad Ad
	if err := s.db.WithContext(ctx).First(&ad, id).Error; err != nil {
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
		Where("isOk = ?", true).
		Order("level DESC, id DESC").
		Limit(size).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list ads by level: %w", err)
	}
	return list, nil
}
