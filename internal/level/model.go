package level

import "time"

type ExpDetail struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"column:user_id" json:"userId"`
	GetExpDate time.Time `gorm:"column:get_exp_date" json:"getExpDate"`
	GetExp     int       `gorm:"column:get_exp" json:"getExp"`
}

func (ExpDetail) TableName() string {
	return "exp_detail"
}

type SignDetail struct {
	Exp         int  `json:"exp"`
	SignDays    int  `json:"signDays"`
	TodaySigned bool `json:"todaySigned"`
}
