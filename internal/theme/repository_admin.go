package theme

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *Repository) UpdateThemesNeedSearch(ctx context.Context, themeIDs []primitive.ObjectID, needSearch bool) error {
	if len(themeIDs) == 0 {
		return nil
	}

	coll, err := r.themeCollection()
	if err != nil {
		return err
	}

	if _, err := coll.UpdateMany(ctx, bson.M{"_id": bson.M{"$in": themeIDs}}, bson.M{"$set": bson.M{"needSearch": needSearch}}); err != nil {
		return fmt.Errorf("update themes needSearch: %w", err)
	}
	return nil
}

func (r *Repository) UpdateThemeSuggestByName(ctx context.Context, themeName string, update bson.M) error {
	coll, err := r.themeCollection()
	if err != nil {
		return err
	}

	if _, err := coll.UpdateMany(ctx, bson.M{"name": themeName}, bson.M{"$set": update}); err != nil {
		return fmt.Errorf("update theme suggest config %s: %w", themeName, err)
	}
	return nil
}

func (r *Repository) FindThemesByNames(ctx context.Context, names []string) ([]Theme, error) {
	if len(names) == 0 {
		return []Theme{}, nil
	}

	coll, err := r.themeCollection()
	if err != nil {
		return nil, err
	}

	cur, err := coll.Find(ctx, bson.M{"name": bson.M{"$in": names}}, options.Find().SetSort(bson.M{"name": 1}))
	if err != nil {
		return nil, fmt.Errorf("find themes by names: %w", err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var themes []Theme
	if err := cur.All(ctx, &themes); err != nil {
		return nil, fmt.Errorf("decode themes by names: %w", err)
	}
	if themes == nil {
		return []Theme{}, nil
	}
	return themes, nil
}
