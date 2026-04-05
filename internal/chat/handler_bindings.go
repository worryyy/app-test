package chat

type conversationIDURI struct {
	ID string `uri:"id" binding:"required"`
}

type lastMessageURI struct {
	LastMessageID string `uri:"last_message_id" binding:"required"`
}

type notificationTypeURI struct {
	Type string `uri:"type" binding:"required"`
}

type conversationQuery struct {
	TargetUserID string `form:"target_user_id" binding:"required"`
}

type conversationProfileQuery struct {
	ConversationID string `form:"conversation_id" binding:"required"`
}

type historyMessagesQuery struct {
	ConversationID  string `form:"conversation_id" binding:"required"`
	OldestMessageID string `form:"oldest_message_id"`
}

type notificationListQuery struct {
	Type string `form:"type"`
}
