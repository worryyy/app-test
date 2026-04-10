package topic

type resourceIDURI struct {
	ID string `uri:"topic_id" binding:"required"`
}

type topicURI struct {
	TopicID string `uri:"topic_id" binding:"required"`
}

type topicSearchQuery struct {
	ThemeIDs   string `form:"theme_ids"`
	Content    string `form:"content"`
	OrdCreated int    `form:"ord_created" binding:"omitempty,oneof=0 1"`
}

type targetUserTopicsQuery struct {
	TargetUserID int64 `form:"target_user_id" binding:"required,gt=0"`
}
