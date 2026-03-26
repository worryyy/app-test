package theme

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) UpdateTheme(ctx context.Context, id string, req *ThemeUpdateReq) (*Theme, error) {
	if req == nil {
		return nil, result.ErrParam
	}

	theme, err := s.GetThemeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	update := bson.M{
		"name":          req.Name,
		"category_name": req.CategoryName,
	}
	if req.NeedSearch != nil {
		update["needSearch"] = *req.NeedSearch
	}

	if _, err := s.themeColl().UpdateByID(ctx, theme.ID, bson.M{"$set": update}); err != nil {
		return nil, fmt.Errorf("update theme: %w", err)
	}

	theme.Name = req.Name
	theme.CategoryName = req.CategoryName
	if req.NeedSearch != nil {
		theme.NeedSearch = *req.NeedSearch
	}
	return theme, nil
}

func (s *Service) UpdateNeedSearch(ctx context.Context, themeIDs []string, needSearch bool) error {
	if len(themeIDs) == 0 {
		return nil
	}

	oids := make([]primitive.ObjectID, 0, len(themeIDs))
	for _, id := range themeIDs {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return fmt.Errorf("invalid theme id: %w", err)
		}
		oids = append(oids, oid)
	}

	if _, err := s.themeColl().UpdateMany(ctx, bson.M{"_id": bson.M{"$in": oids}}, bson.M{"$set": bson.M{"needSearch": needSearch}}); err != nil {
		return fmt.Errorf("update need search: %w", err)
	}
	return nil
}

func (s *Service) UpdateSuggestByList(ctx context.Context, req *ThemeSuggestReq) ([]Theme, error) {
	if req == nil {
		return nil, result.ErrParam
	}
	if len(req.List) == 0 {
		return []Theme{}, nil
	}

	names := make([]string, 0, len(req.List))
	seen := make(map[string]struct{}, len(req.List))
	for _, item := range req.List {
		update := bson.M{
			"needSuggest":       item.NeedSuggest,
			"suggestBasicScore": item.SuggestBasicScore,
			"suggestNumber":     item.SuggestNumber,
			"suggestSetName":    item.SuggestSetName,
			"suggestType":       item.SuggestType,
		}
		if _, err := s.themeColl().UpdateMany(ctx, bson.M{"name": item.ThemeName}, bson.M{"$set": update}); err != nil {
			return nil, fmt.Errorf("update suggest config: %w", err)
		}
		if _, ok := seen[item.ThemeName]; !ok {
			names = append(names, item.ThemeName)
			seen[item.ThemeName] = struct{}{}
		}
	}

	cur, err := s.themeColl().Find(ctx, bson.M{"name": bson.M{"$in": names}}, options.Find().SetSort(bson.M{"name": 1}))
	if err != nil {
		return nil, fmt.Errorf("list updated suggest themes: %w", err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var themes []Theme
	if err := cur.All(ctx, &themes); err != nil {
		return nil, fmt.Errorf("decode updated suggest themes: %w", err)
	}
	return themes, nil
}
