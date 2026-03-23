package theme

type ThemeSearchReq struct {
	ThemeID    string `json:"themeId" binding:"required"`
	NeedSearch bool   `json:"needSearch"`
}

type ThemeSuggestReq struct {
	ThemeID           string `json:"themeId" binding:"required"`
	NeedSuggest       bool   `json:"needSuggest"`
	SuggestBasicScore int64  `json:"suggestBasicScore"`
	SuggestNumber     int    `json:"suggestNumber"`
	SuggestSetName    string `json:"suggestSetName"`
	SuggestType       string `json:"suggestType"`
}
