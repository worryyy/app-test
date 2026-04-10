package theme

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *Repository) FindThemes(ctx context.Context, name string) ([]Theme, error) {
	coll, err := r.themeCollection()
	if err != nil {
		return nil, err
	}

	filter := bson.M{}
	if strings.TrimSpace(name) != "" {
		filter["name"] = strings.TrimSpace(name)
	}

	cur, err := coll.Find(ctx, filter, options.Find().SetSort(bson.M{"name": 1}))
	if err != nil {
		return nil, fmt.Errorf("find themes: %w", err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var themes []Theme
	if err := cur.All(ctx, &themes); err != nil {
		return nil, fmt.Errorf("decode themes: %w", err)
	}
	if themes == nil {
		return []Theme{}, nil
	}
	return themes, nil
}

func (r *Repository) CreateTheme(ctx context.Context, theme *Theme) (primitive.ObjectID, error) {
	if theme == nil {
		return primitive.NilObjectID, nil
	}

	coll, err := r.themeCollection()
	if err != nil {
		return primitive.NilObjectID, err
	}

	res, err := coll.InsertOne(ctx, theme)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("insert theme: %w", err)
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("inserted theme id type invalid")
	}
	theme.ID = oid
	return oid, nil
}

func (r *Repository) FindThemeByID(ctx context.Context, id primitive.ObjectID) (*Theme, error) {
	coll, err := r.themeCollection()
	if err != nil {
		return nil, err
	}

	var theme Theme
	if err := coll.FindOne(ctx, bson.M{"_id": id}).Decode(&theme); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find theme %s: %w", id.Hex(), err)
	}
	return &theme, nil
}
