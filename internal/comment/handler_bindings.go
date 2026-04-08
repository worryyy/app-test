package comment

type topicURI struct {
	TopicID string `uri:"topic_id" binding:"required"`
}

type topicCommentURI struct {
	TopicID   string `uri:"topic_id" binding:"required"`
	CommentID string `uri:"comment_id" binding:"required"`
}

type commentURI struct {
	CommentID string `uri:"comment_id" binding:"required"`
}

type commentListQuery struct {
	RootID string `form:"root_id"`
}

type targetUserCommentsQuery struct {
	TargetUserID int64 `form:"target_user_id" binding:"required,gt=0"`
}
