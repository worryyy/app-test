package topic

type resourceIDURI struct {
	ID string `uri:"id" binding:"required"`
}

type topicURI struct {
	TopicID string `uri:"topic_id" binding:"required"`
}

type topicSearchQuery struct {
	ThemeIDs   string `form:"theme_ids"`
	ThemeID    string `form:"theme_id"`
	Content    string `form:"content"`
	Keyword    string `form:"keyword"`
	OrdCreated string `form:"ord_created"`
	OrderBy    string `form:"order_by"`
}

func (q topicSearchQuery) ResolvedThemeInput() string {
	return firstNonEmpty(q.ThemeIDs, q.ThemeID)
}

func (q topicSearchQuery) ResolvedContent() string {
	return firstNonEmpty(q.Content, q.Keyword)
}

func (q topicSearchQuery) ResolvedOrdCreated() string {
	return firstNonEmpty(q.OrdCreated, q.OrderBy)
}

type targetUserTopicsQuery struct {
	TargetUserID string `form:"target_user_id"`
}

func (q targetUserTopicsQuery) ResolvedTargetUserID() string {
	return q.TargetUserID
}
