package agentchat

import "time"

const (
	conversationStatusPending = "pending"
	conversationStatusReady   = "ready"
	conversationStatusError   = "error"
)

type Conversation struct {
	SessionID            string    `gorm:"column:session_id;primaryKey;size:64" json:"conversationId"`
	RootUserID           int64     `gorm:"column:root_user_id;index:idx_agent_conversations_root_updated,priority:1" json:"rootUserId"`
	CreatorUserID        int64     `gorm:"column:creator_user_id" json:"creatorUserId"`
	LastActorUserID      int64     `gorm:"column:last_actor_user_id" json:"lastActorUserId"`
	Title                string    `gorm:"column:title;size:128" json:"title"`
	LastUserPreview      string    `gorm:"column:last_user_preview;size:255" json:"lastUserPreview"`
	LastAssistantPreview string    `gorm:"column:last_assistant_preview;size:255" json:"lastAssistantPreview"`
	LastRequestID        string    `gorm:"column:last_request_id;size:64" json:"lastRequestId"`
	LastTraceID          string    `gorm:"column:last_trace_id;size:64" json:"lastTraceId"`
	Status               string    `gorm:"column:status;size:32;index:idx_agent_conversations_root_updated,priority:2" json:"status"`
	CreatedAt            time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt            time.Time `gorm:"column:updated_at;autoUpdateTime;index:idx_agent_conversations_root_updated,priority:3,sort:desc" json:"updatedAt"`
}

func (Conversation) TableName() string {
	return "agent_conversations"
}

type HistoryTurn struct {
	SequenceNo int64  `json:"sequenceNo"`
	Role       string `json:"role"`
	Content    string `json:"content"`
	CreatedAt  string `json:"createdAt"`
	Domain     string `json:"domain"`
}

type HistoryResponse struct {
	ConversationID       string        `json:"conversationId"`
	Turns                []HistoryTurn `json:"turns"`
	HasMore              bool          `json:"hasMore"`
	NextBeforeSequenceNo int64         `json:"nextBeforeSequenceNo"`
}

type TurnReference struct {
	Source string  `json:"source"`
	Ref    string  `json:"ref"`
	Score  float64 `json:"score"`
}

type TurnResult struct {
	Domain        string          `json:"domain"`
	Intent        string          `json:"intent"`
	Mode          string          `json:"mode"`
	AnswerText    string          `json:"answerText"`
	TraceID       string          `json:"traceId"`
	Degraded      bool            `json:"degraded"`
	DegradeReason string          `json:"degradeReason,omitempty"`
	ErrorCode     string          `json:"errorCode"`
	References    []TurnReference `json:"references,omitempty"`
}

type TurnResponse struct {
	RequestID      string     `json:"requestId"`
	ConversationID string     `json:"conversationId"`
	Result         TurnResult `json:"result"`
}

type WSEvent struct {
	Type           string           `json:"type"`
	RequestID      string           `json:"requestId,omitempty"`
	ConversationID string           `json:"conversationId,omitempty"`
	RootUserID     string           `json:"rootUserId,omitempty"`
	Stage          string           `json:"stage,omitempty"`
	Message        string           `json:"message,omitempty"`
	Delta          string           `json:"delta,omitempty"`
	ErrorCode      string           `json:"errorCode,omitempty"`
	Result         *TurnResult      `json:"result,omitempty"`
	History        *HistoryResponse `json:"history,omitempty"`
}
