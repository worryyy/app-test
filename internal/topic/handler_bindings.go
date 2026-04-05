package topic

type resourceIDURI struct {
	ID string `uri:"id" binding:"required"`
}

type topicURI struct {
	TopicID string `uri:"topic_id" binding:"required"`
}

type topicSearchQuery struct {
	ThemeIDs      string `form:"themeIds"`
	LegacyThemeID string `form:"themeId"`
	Content       string `form:"content"`
	LegacyKeyword string `form:"keyword"`
	OrdCreated    string `form:"ord_created"`
	OrderBy       string `form:"orderBy"`
}

func (q topicSearchQuery) ResolvedThemeInput() string {
	return firstNonEmpty(q.ThemeIDs, q.LegacyThemeID)
}

func (q topicSearchQuery) ResolvedContent() string {
	return firstNonEmpty(q.Content, q.LegacyKeyword)
}

func (q topicSearchQuery) ResolvedOrdCreated() string {
	return firstNonEmpty(q.OrdCreated, q.OrderBy)
}

type themeMineQuery struct {
	ThemeID       string `form:"theme_id"`
	LegacyThemeID string `form:"themeId"`
}

func (q themeMineQuery) ResolvedThemeID() string {
	return firstNonEmpty(q.ThemeID, q.LegacyThemeID)
}

type targetUserTopicsQuery struct {
	TargetUserID       string `form:"target_user_id"`
	LegacyTargetUserID string `form:"targetUserId"`
}

func (q targetUserTopicsQuery) ResolvedTargetUserID() string {
	return firstNonEmpty(q.TargetUserID, q.LegacyTargetUserID)
}
