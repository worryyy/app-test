package agentchat

type conversationIDURI struct {
	ID string `uri:"id" binding:"required"`
}

type historyQuery struct {
	BeforeSequenceNo string `form:"before_sequence_no"`
	Size             int    `form:"size"`
}

type wsAuthEnvelope struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type wsTurnStartRequest struct {
	Type           string `json:"type"`
	RequestID      string `json:"requestId"`
	ConversationID string `json:"conversationId"`
	Content        string `json:"content"`
	ClientPlatform string `json:"clientPlatform"`
	SchoolID       string `json:"schoolId"`
	SchoolName     string `json:"schoolName"`
	Locale         string `json:"locale"`
}
