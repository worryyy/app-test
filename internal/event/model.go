package event

import "time"

type Event struct {
	EventID      int64     `gorm:"column:eventId;primaryKey;autoIncrement" json:"eventId"`
	EventType    string    `gorm:"column:eventType" json:"eventType"`
	EventInfo    string    `gorm:"column:eventInfo" json:"eventInfo"`
	EventContent string    `gorm:"column:eventContent" json:"eventContent"`
	UserID       int64     `gorm:"column:userId" json:"userId"`
	TriggerTime  time.Time `gorm:"column:triggerTime" json:"triggerTime"`
}

func (Event) TableName() string {
	return "event_data"
}
