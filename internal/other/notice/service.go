package notice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateNotice(ctx context.Context, n *Notice) error {
	if err := s.db.WithContext(ctx).Create(n).Error; err != nil {
		return result.NewBizError(result.CodeFail, "添加失败")
	}
	return nil
}

func (s *Service) DeleteNotice(ctx context.Context, id int64) error {
	var n Notice
	if err := s.db.WithContext(ctx).First(&n, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result.NewBizError(result.CodeFail, fmt.Sprintf("id:%d不存在", id))
		}
		return fmt.Errorf("query notice before delete: %w", err)
	}
	res := s.db.WithContext(ctx).Delete(&Notice{}, id)
	if res.Error != nil {
		return fmt.Errorf("delete notice: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return result.NewBizError(result.CodeFail, "删除失败")
	}
	return nil
}

func (s *Service) UpdateNotice(ctx context.Context, id int64, n *Notice) error {
	var existed Notice
	if err := s.db.WithContext(ctx).First(&existed, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result.NewBizError(result.CodeFail, "不存在")
		}
		return fmt.Errorf("query notice before update: %w", err)
	}
	if n == nil || strings.TrimSpace(n.Content) == "" {
		return result.NewBizError(result.CodeFail, "content 不能为空")
	}

	res := s.db.WithContext(ctx).Model(&Notice{}).Where("id = ?", id).Updates(n)
	if res.Error != nil {
		return fmt.Errorf("update notice: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return result.NewBizError(result.CodeFail, "更新失败")
	}
	return nil
}

func (s *Service) GetNotice(ctx context.Context, id int64) (*Notice, error) {
	var n Notice
	if err := s.db.WithContext(ctx).First(&n, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
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
	if err := s.db.WithContext(ctx).Offset((page - 1) * size).Limit(size).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list notices: %w", err)
	}
	return result.NewPage(list, total, page, size), nil
}

func (s *Service) ListFrontendNotices(ctx context.Context, page, size int) ([]NoticeItem, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}

	var notices []Notice
	if err := s.db.WithContext(ctx).
		Select("content", "updated_at").
		Offset((page - 1) * size).
		Limit(size).
		Order("updated_at DESC").
		Find(&notices).Error; err != nil {
		return nil, fmt.Errorf("list frontend notices: %w", err)
	}

	vos := make([]NoticeItem, 0, len(notices))
	for _, item := range notices {
		vos = append(vos, NoticeItem{
			Content:   item.Content,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return vos, nil
}
