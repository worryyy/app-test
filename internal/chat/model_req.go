package chat

type ConversationEnterReq struct {
	ConversationID int64 `json:"conversationId" binding:"required"`
}
