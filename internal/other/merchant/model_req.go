package merchant

type MerchantThemeReq struct {
	ThemeID string `json:"themeId" binding:"required"`
}

type TaskNameReq struct {
	Name string `json:"name" binding:"required"`
}
