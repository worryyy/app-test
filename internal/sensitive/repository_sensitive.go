package sensitive

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

func (r *Repository) FindAll(ctx context.Context) ([]SensitiveWord, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var list []SensitiveWord
	if err := db.Order("id DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("find all sensitive words: %w", err)
	}
	if list == nil {
		return []SensitiveWord{}, nil
	}
	return list, nil
}

func (r *Repository) FindByWord(ctx context.Context, word string) (*SensitiveWord, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var item SensitiveWord
	if err := db.Where("word = ?", word).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find sensitive word %q: %w", word, err)
	}
	return &item, nil
}

func (r *Repository) Create(ctx context.Context, item *SensitiveWord) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	if err := db.Create(item).Error; err != nil {
		return fmt.Errorf("create sensitive word %q: %w", item.Word, err)
	}
	return nil
}

func (r *Repository) DeleteByWord(ctx context.Context, word string) (bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}

	res := db.Where("word = ?", word).Delete(&SensitiveWord{})
	if res.Error != nil {
		return false, fmt.Errorf("delete sensitive word %q: %w", word, res.Error)
	}
	return res.RowsAffected > 0, nil
}

func (r *Repository) FindPage(ctx context.Context, page, size int) ([]SensitiveWord, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}

	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, 0, err
	}

	query := db.Model(&SensitiveWord{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count sensitive words: %w", err)
	}

	var list []SensitiveWord
	if err := query.
		Order("id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("find sensitive words page: %w", err)
	}
	if list == nil {
		return []SensitiveWord{}, total, nil
	}
	return list, total, nil
}

func (r *Repository) FindByLike(ctx context.Context, word string) ([]SensitiveWord, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var list []SensitiveWord
	if err := db.Where("word LIKE ?", "%"+word+"%").Order("id DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("search sensitive words by like %q: %w", word, err)
	}
	if list == nil {
		return []SensitiveWord{}, nil
	}
	return list, nil
}
