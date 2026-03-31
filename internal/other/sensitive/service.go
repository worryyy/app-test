package sensitive

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

func (s *Service) GetAllSensitiveWords(ctx context.Context) ([]SensitiveWord, error) {
	var list []SensitiveWord
	if err := s.db.WithContext(ctx).Order("id ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("get all sensitive words: %w", err)
	}
	return list, nil
}

func (s *Service) GetSensitiveWordByWord(ctx context.Context, word string) (*SensitiveWord, error) {
	if strings.TrimSpace(word) == "" {
		return nil, result.NewBizError(result.CodeFail, "参数为NULL，请重试")
	}
	var sw SensitiveWord
	if err := s.db.WithContext(ctx).Where("word = ?", word).First(&sw).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, result.NewBizError(result.CodeFail, "不存在这个敏感词")
		}
		return nil, fmt.Errorf("get sensitive word by word: %w", err)
	}
	return &sw, nil
}

func (s *Service) DeleteSensitiveWordByWord(ctx context.Context, word string) error {
	if strings.TrimSpace(word) == "" {
		return result.NewBizError(result.CodeFail, "参数为NULL，请重试")
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&SensitiveWord{}).Where("word = ?", word).Count(&count).Error; err != nil {
		return fmt.Errorf("count sensitive word by word: %w", err)
	}
	if count == 0 {
		return result.NewBizError(result.CodeFail, "关键词："+word+"不存在")
	}

	res := s.db.WithContext(ctx).Where("word = ?", word).Delete(&SensitiveWord{})
	if res.Error != nil {
		return fmt.Errorf("delete sensitive word by word: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return result.NewBizError(result.CodeFail, "删除关键词："+word+"失败")
	}
	return nil
}

func (s *Service) BatchDeleteSensitiveWords(ctx context.Context, words []string) error {
	if len(words) == 0 {
		return result.NewBizError(result.CodeFail, "待删除关键词列表为空")
	}

	res := s.db.WithContext(ctx).Where("word IN ?", words).Delete(&SensitiveWord{})
	if res.Error != nil {
		return fmt.Errorf("batch delete sensitive words: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return result.NewBizError(result.CodeFail, "批量删除关键词失败")
	}
	return nil
}

func (s *Service) AddSensitiveWord(ctx context.Context, sw *SensitiveWord) error {
	if sw == nil || strings.TrimSpace(sw.Word) == "" {
		return result.NewBizError(result.CodeFail, "参数为NULL，请重试")
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&SensitiveWord{}).Where("word = ?", sw.Word).Count(&count).Error; err != nil {
		return fmt.Errorf("count sensitive word: %w", err)
	}
	if count > 0 {
		return result.NewBizError(result.CodeFail, "关键词："+sw.Word+"已经存在")
	}

	if err := s.db.WithContext(ctx).Create(sw).Error; err != nil {
		return result.NewBizError(result.CodeFail, "关键词："+sw.Word+"添加失败，请重试")
	}
	return nil
}

func (s *Service) BatchAddSensitiveWords(ctx context.Context, words []string) error {
	if len(words) == 0 {
		return result.NewBizError(result.CodeFail, "参数为NULL，请重试")
	}

	for _, word := range words {
		if strings.TrimSpace(word) == "" {
			return result.NewBizError(result.CodeFail, "参数为NULL，请重试")
		}
		var count int64
		if err := s.db.WithContext(ctx).Model(&SensitiveWord{}).Where("word = ?", word).Count(&count).Error; err != nil {
			return fmt.Errorf("count sensitive word in batch: %w", err)
		}
		if count > 0 {
			return result.NewBizError(result.CodeFail, "关键词："+word+"已经存在")
		}
	}

	items := make([]SensitiveWord, 0, len(words))
	for _, word := range words {
		items = append(items, SensitiveWord{Word: word})
	}
	if err := s.db.WithContext(ctx).Create(&items).Error; err != nil {
		return result.NewBizError(result.CodeFail, "关键词批量添加失败，请重试")
	}
	return nil
}

func (s *Service) SensitiveWordPage(ctx context.Context, page, size int) (*result.PageResult[SensitiveWord], error) {
	var total int64
	if err := s.db.WithContext(ctx).Model(&SensitiveWord{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count sensitive words: %w", err)
	}
	var list []SensitiveWord
	if err := s.db.WithContext(ctx).Offset((page - 1) * size).Limit(size).Order("id DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("page sensitive words: %w", err)
	}
	return result.NewPage(list, total, page, size), nil
}

func (s *Service) SearchSensitiveWordLike(ctx context.Context, word string) ([]SensitiveWord, error) {
	var list []SensitiveWord
	if err := s.db.WithContext(ctx).Where("word LIKE ?", "%"+word+"%").Order("id DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("search sensitive words like: %w", err)
	}
	if len(list) == 0 {
		return nil, result.NewBizError(result.CodeFail, "无匹配的敏感词")
	}
	return list, nil
}

func (s *Service) UpdateSensitiveWord(ctx context.Context, sw *SensitiveWord) error {
	if err := s.db.WithContext(ctx).Model(&SensitiveWord{}).Where("id = ?", sw.ID).Update("word", sw.Word).Error; err != nil {
		return fmt.Errorf("update sensitive word: %w", err)
	}
	return nil
}

func (s *Service) UpdateSensitiveWordByWord(ctx context.Context, word, updateWord string) error {
	if strings.TrimSpace(word) == "" || strings.TrimSpace(updateWord) == "" {
		return result.NewBizError(result.CodeFail, "参数为NULL，请重试")
	}

	res := s.db.WithContext(ctx).Model(&SensitiveWord{}).Where("word = ?", word).Update("word", updateWord)
	if res.Error != nil {
		return fmt.Errorf("update sensitive word by word: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return result.NewBizError(result.CodeFail, "更新失败，请重试")
	}
	return nil
}
