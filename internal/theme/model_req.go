package theme

type ThemeUpdateReq struct {
	Name         string `json:"name" binding:"required,max=20"`
	CategoryName string `json:"category_name" binding:"required,max=20"`
	NeedSearch   *bool  `json:"needSearch"`
}

type ThemeSearchReq struct {
	ThemeIDs []string `json:"themeIds" binding:"required,min=1"`
}

type ThemeSuggestReq struct {
	List []ThemeSuggestItem `json:"list" binding:"required,min=1"`
}

type ThemeSuggestItem struct {
	ThemeName         string `json:"theme_name" binding:"required"`
	NeedSuggest       bool   `json:"needSuggest"`
	SuggestBasicScore int64  `json:"suggestBasicScore"`
	SuggestNumber     int    `json:"suggestNumber"`
	SuggestSetName    string `json:"suggestSetName" binding:"required"`
	SuggestType       int    `json:"suggestType"`
}
