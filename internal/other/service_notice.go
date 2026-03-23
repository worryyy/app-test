package other

import (
	"context"
	"fmt"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) CreateNotice(ctx context.Context, n *Notice) error {
	if err := s.db.WithContext(ctx).Create(n).Error; err != nil {
		return fmt.Errorf("create notice: %w", err)
	}
	return nil
}

func (s *Service) DeleteNotice(ctx context.Context, id int64) error {
	if err := s.db.WithContext(ctx).Delete(&Notice{}, id).Error; err != nil {
		return fmt.Errorf("delete notice: %w", err)
	}
	return nil
}

func (s *Service) UpdateNotice(ctx context.Context, id int64, n *Notice) error {
	if err := s.db.WithContext(ctx).Model(&Notice{}).Where("id = ?", id).Updates(n).Error; err != nil {
		return fmt.Errorf("update notice: %w", err)
	}
	return nil
}

func (s *Service) GetNotice(ctx context.Context, id int64) (*Notice, error) {
	var n Notice
	if err := s.db.WithContext(ctx).First(&n, id).Error; err != nil {
		return nil, fmt.Errorf("get notice: %w", err)
	}
	return &n, nil
}

func (s *Service) ListNotices(ctx context.Context, page, size int) (*result.PageResult[Notice], error) {
	var total int64
	if err := s.db.WithContext(ctx).Model(&Notice{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count notices: %w", err)
	}

	var list []Notice
	if err := s.db.WithContext(ctx).Offset((page - 1) * size).Limit(size).Order("createdAt DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list notices: %w", err)
	}
	return result.NewPage(list, total, page, size), nil
}
