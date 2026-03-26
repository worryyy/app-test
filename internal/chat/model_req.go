package chat

type ConversationEnterReq struct {
	ConversationID string `form:"conversation_id" binding:"required"`
	LastMessageID  string `form:"last_message_id"`
}
