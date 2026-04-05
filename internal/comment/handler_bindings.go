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
	RootID       string `form:"root_id"`
	LegacyRootID string `form:"rootId"`
}

func (q commentListQuery) ResolvedRootID() string {
	return firstNonEmpty(q.RootID, q.LegacyRootID)
}

type targetUserCommentsQuery struct {
	TargetUserID       string `form:"target_user_id"`
	LegacyTargetUserID string `form:"targetUserId"`
}

func (q targetUserCommentsQuery) ResolvedTargetUserID() string {
	return firstNonEmpty(q.TargetUserID, q.LegacyTargetUserID)
}
