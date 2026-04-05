package theme

type themeIDURI struct {
	ID string `uri:"id" binding:"required"`
}

type campusThemeURI struct {
	ThemeID string `uri:"themeId" binding:"required"`
}

type themeListQuery struct {
	Name string `form:"name"`
}
