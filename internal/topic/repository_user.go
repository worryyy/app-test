package topic

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

func (r *Repository) FindUserByID(ctx context.Context, id int64) (*topicAuthor, error) {
	if id <= 0 {
		return nil, nil
	}

	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var author topicAuthor
	if err := db.Where(colTopicUserID+" = ?", id).First(&author).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find topic author by id %d: %w", id, err)
	}
	return &author, nil
}

func (r *Repository) FindUserByRootAndAccountType(
	ctx context.Context,
	rootUserID int64,
	accountType string,
) (*topicAuthor, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var author topicAuthor
	if err := db.
		Where(colTopicRootUserID+" = ? AND "+colTopicAccountType+" = ?", rootUserID, accountType).
		First(&author).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find topic author by root_user_id/account_type: %w", err)
	}
	return &author, nil
}
