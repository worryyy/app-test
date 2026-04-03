package theme

import (
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
)

func (s *Service) UpdateTheme(ctx context.Context, id string, req *ThemeUpdateReq) (*Theme, error) {
	if req == nil {
		return nil, bizerr.Param(errMsgInvalidParam)
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

	if err := s.repo.UpdateTheme(ctx, theme.ID, update); err != nil {
		return nil, bizerr.InternalWrap("更新主题失败", err)
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
		return bizerr.Param(errMsgInvalidParam)
	}

	oids, err := parseThemeObjectIDs(themeIDs)
	if err != nil {
		return err
	}

	if err := s.repo.UpdateThemesNeedSearch(ctx, oids, needSearch); err != nil {
		return bizerr.InternalWrap("更新主题检索状态失败", err)
	}
	return nil
}

func (s *Service) UpdateSuggestByList(ctx context.Context, req *ThemeSuggestReq) ([]Theme, error) {
	if req == nil || len(req.List) == 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	names := make([]string, 0, len(req.List))
	seen := make(map[string]struct{}, len(req.List))
	for _, item := range req.List {
		themeName := strings.TrimSpace(item.ThemeName)
		if themeName == "" {
			return nil, bizerr.Param(errMsgInvalidParam)
		}

		update := bson.M{
			"needSuggest":       item.NeedSuggest,
			"suggestBasicScore": item.SuggestBasicScore,
			"suggestNumber":     item.SuggestNumber,
			"suggestSetName":    item.SuggestSetName,
			"suggestType":       item.SuggestType,
		}
		if err := s.repo.UpdateThemeSuggestByName(ctx, themeName, update); err != nil {
			return nil, bizerr.InternalWrap("更新主题推荐配置失败", err)
		}

		if _, ok := seen[themeName]; !ok {
			names = append(names, themeName)
			seen[themeName] = struct{}{}
		}
	}

	themes, err := s.repo.FindThemesByNames(ctx, names)
	if err != nil {
		return nil, bizerr.InternalWrap("查询主题推荐配置失败", err)
	}
	return themes, nil
}
