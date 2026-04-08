package topic

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
