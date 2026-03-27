package monitor

import "time"

type CacheStats struct {
	CacheName string  `json:"cacheName"`
	Size      int64   `json:"size"`
	HitCount  int64   `json:"hitCount"`
	MissCount int64   `json:"missCount"`
	HitRate   float64 `json:"hitRate"`
}

type ControllerTime struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Controller string    `gorm:"column:controller" json:"controller"`
	TimeCost   int64     `gorm:"column:time_cost" json:"time_cost"`
	Success    int       `gorm:"column:success" json:"success"`
	AddTime    time.Time `gorm:"column:add_time" json:"add_time"`
}

func (ControllerTime) TableName() string {
	return "controller_time"
}
