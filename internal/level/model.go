package level

import "time"

type ExpDetail struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"column:user_id" json:"user_id"`
	GetExpDate time.Time `gorm:"column:get_exp_date" json:"getExpDate"`
	GetExp     int       `gorm:"column:get_exp" json:"getExp"`
}

func (ExpDetail) TableName() string {
	return "exp_detail"
}

type UserSignDetail struct {
	UserID  int64 `json:"userId"`
	UserExp int   `json:"userExp"`
	Signed  bool  `json:"signed"`
}

type UserExpItem struct {
	UserID  int64 `json:"userId"`
	UserExp int   `json:"userExp"`
}
