package topic

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

func (r *Repository) ThemeExists(ctx context.Context, themeID string) (bool, error) {
	coll, err := r.mongoCollection(mongoCollThemeID)
	if err != nil {
		return false, err
	}

	err = coll.FindOne(ctx, bson.M{"themeId": themeID}).Err()
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return false, fmt.Errorf("check theme %s exists: %w", themeID, err)
}

func (r *Repository) FindThemeName(ctx context.Context, themeID string) (string, error) {
	coll, err := r.mongoCollection(mongoCollThemeID)
	if err != nil {
		return "", err
	}

	var doc campusThemeID
	if err := coll.FindOne(ctx, bson.M{"themeId": themeID}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", nil
		}
		return "", fmt.Errorf("find theme %s: %w", themeID, err)
	}
	return doc.Name, nil
}

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
