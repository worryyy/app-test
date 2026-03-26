package other

import (
	"context"
	"fmt"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) GetAllSensitiveWords(ctx context.Context) ([]SensitiveWord, error) {
	var list []SensitiveWord
	if err := s.db.WithContext(ctx).Order("id ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("get all sensitive words: %w", err)
	}
	return list, nil
}

func (s *Service) GetSensitiveWordByWord(ctx context.Context, word string) (*SensitiveWord, error) {
	var sw SensitiveWord
	if err := s.db.WithContext(ctx).Where("word = ?", word).First(&sw).Error; err != nil {
		return nil, fmt.Errorf("get sensitive word by word: %w", err)
	}
	return &sw, nil
}

func (s *Service) DeleteSensitiveWordByWord(ctx context.Context, word string) error {
	if err := s.db.WithContext(ctx).Where("word = ?", word).Delete(&SensitiveWord{}).Error; err != nil {
		return fmt.Errorf("delete sensitive word by word: %w", err)
	}
	return nil
}

func (s *Service) BatchDeleteSensitiveWords(ctx context.Context, words []string) error {
	if len(words) == 0 {
		return nil
	}
	if err := s.db.WithContext(ctx).Where("word IN ?", words).Delete(&SensitiveWord{}).Error; err != nil {
		return fmt.Errorf("batch delete sensitive words: %w", err)
	}
	return nil
}

func (s *Service) AddSensitiveWord(ctx context.Context, sw *SensitiveWord) error {
	if err := s.db.WithContext(ctx).Create(sw).Error; err != nil {
		return fmt.Errorf("add sensitive word: %w", err)
	}
	return nil
}

func (s *Service) BatchAddSensitiveWords(ctx context.Context, words []string) error {
	if len(words) == 0 {
		return nil
	}
	items := make([]SensitiveWord, 0, len(words))
	for _, word := range words {
		items = append(items, SensitiveWord{Word: word})
	}
	if err := s.db.WithContext(ctx).Create(&items).Error; err != nil {
		return fmt.Errorf("batch add sensitive words: %w", err)
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
	return list, nil
}

func (s *Service) UpdateSensitiveWord(ctx context.Context, sw *SensitiveWord) error {
	if err := s.db.WithContext(ctx).Model(&SensitiveWord{}).Where("id = ?", sw.ID).Update("word", sw.Word).Error; err != nil {
		return fmt.Errorf("update sensitive word: %w", err)
	}
	return nil
}

func (s *Service) UpdateSensitiveWordByWord(ctx context.Context, word, updateWord string) error {
	if err := s.db.WithContext(ctx).Model(&SensitiveWord{}).Where("word = ?", word).Update("word", updateWord).Error; err != nil {
		return fmt.Errorf("update sensitive word by word: %w", err)
	}
	return nil
}
