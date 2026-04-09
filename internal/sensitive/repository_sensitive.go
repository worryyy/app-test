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

func (r *Repository) FindByWords(ctx context.Context, words []string) ([]SensitiveWord, error) {
	if len(words) == 0 {
		return []SensitiveWord{}, nil
	}

	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var list []SensitiveWord
	if err := db.Where("word IN ?", words).Find(&list).Error; err != nil {
		return nil, fmt.Errorf("find sensitive words by list: %w", err)
	}
	if list == nil {
		return []SensitiveWord{}, nil
	}
	return list, nil
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

func (r *Repository) CreateBatch(ctx context.Context, items []SensitiveWord) error {
	if len(items) == 0 {
		return nil
	}

	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	if err := db.Create(&items).Error; err != nil {
		return fmt.Errorf("create sensitive words batch: %w", err)
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

func (r *Repository) DeleteByWords(ctx context.Context, words []string) (int64, error) {
	if len(words) == 0 {
		return 0, nil
	}

	db, err := r.gormDB(ctx)
	if err != nil {
		return 0, err
	}

	res := db.Where("word IN ?", words).Delete(&SensitiveWord{})
	if res.Error != nil {
		return 0, fmt.Errorf("delete sensitive words batch: %w", res.Error)
	}
	return res.RowsAffected, nil
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

func (r *Repository) UpdateByWord(ctx context.Context, word, updateWord string) (bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}

	res := db.Model(&SensitiveWord{}).Where("word = ?", word).Update("word", updateWord)
	if res.Error != nil {
		return false, fmt.Errorf("update sensitive word %q to %q: %w", word, updateWord, res.Error)
	}
	return res.RowsAffected > 0, nil
}
