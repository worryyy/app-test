package event

import "time"

type EventUpdateReq struct {
	EventType    string `json:"eventType"`
	EventInfo    string `json:"eventInfo"`
	EventContent string `json:"eventContent"`
}

type EventListReq struct {
	PrevID    int64
	Size      int
	StartTime time.Time
	UserID    string
	EventType string
	KeyWord   string
}

type EventListResp struct {
	Data  []Event `json:"data"`
	Total int64   `json:"total"`
}
