package event

import "time"

type Event struct {
	EventID      int64     `gorm:"column:event_id;primaryKey;autoIncrement" json:"eventId"`
	EventType    string    `gorm:"column:event_type" json:"eventType"`
	EventInfo    string    `gorm:"column:event_info" json:"eventInfo"`
	EventContent string    `gorm:"column:event_content" json:"eventContent"`
	UserID       string    `gorm:"column:user_id" json:"user_id"`
	TriggerTime  time.Time `gorm:"column:trigger_time" json:"triggerTime"`
}

func (Event) TableName() string {
	return "event_data"
}
