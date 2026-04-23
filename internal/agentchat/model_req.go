package agentchat

type TurnRequest struct {
	RequestID      string `json:"requestId"`
	ConversationID string `json:"conversationId"`
	Content        string `json:"content" binding:"required"`
	ClientPlatform string `json:"clientPlatform"`
	SchoolID       string `json:"schoolId"`
	SchoolName     string `json:"schoolName"`
	Locale         string `json:"locale"`
}
