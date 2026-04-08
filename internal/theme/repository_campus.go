package theme

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *Repository) UpsertCampusTheme(ctx context.Context, theme *CampusTheme) error {
	if theme == nil {
		return nil
	}

	coll, err := r.campusThemeCollection()
	if err != nil {
		return err
	}

	if _, err := coll.UpdateOne(
		ctx,
		bson.M{"themeId": theme.ThemeID},
		bson.M{"$setOnInsert": bson.M{
			"name":              theme.Name,
			"themeId":           theme.ThemeID,
			"category_name":     theme.CategoryName,
			"needSearch":        theme.NeedSearch,
			"needSuggest":       theme.NeedSuggest,
			"suggestBasicScore": theme.SuggestBasicScore,
			"suggestNumber":     theme.SuggestNumber,
			"suggestSetName":    theme.SuggestSetName,
			"suggestType":       theme.SuggestType,
		}},
		options.Update().SetUpsert(true),
	); err != nil {
		return fmt.Errorf("upsert campus theme %s: %w", theme.ThemeID, err)
	}
	return nil
}

func (r *Repository) FindCampusThemes(ctx context.Context) ([]CampusTheme, error) {
	coll, err := r.campusThemeCollection()
	if err != nil {
		return nil, err
	}

	cur, err := coll.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"_id": 1}))
	if err != nil {
		return nil, fmt.Errorf("find campus themes: %w", err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var themes []CampusTheme
	if err := cur.All(ctx, &themes); err != nil {
		return nil, fmt.Errorf("decode campus themes: %w", err)
	}
	if themes == nil {
		return []CampusTheme{}, nil
	}
	return themes, nil
}

func (r *Repository) CreateCampusTheme(ctx context.Context, theme *CampusTheme) error {
	if theme == nil {
		return nil
	}

	coll, err := r.campusThemeCollection()
	if err != nil {
		return err
	}

	res, err := coll.InsertOne(ctx, theme)
	if err != nil {
		return fmt.Errorf("insert campus theme: %w", err)
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		theme.ID = oid
	}
	return nil
}

func (r *Repository) DeleteCampusThemeByThemeID(ctx context.Context, themeID string) (bool, error) {
	coll, err := r.campusThemeCollection()
	if err != nil {
		return false, err
	}

	res, err := coll.DeleteOne(ctx, bson.M{"themeId": themeID})
	if err != nil {
		return false, fmt.Errorf("delete campus theme %s: %w", themeID, err)
	}
	return res.DeletedCount > 0, nil
}
