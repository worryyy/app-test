package ad

import (
	"time"

	"gorm.io/gorm"
)

type Ad struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BannerURL string         `gorm:"column:banner_url" json:"bannerUrl"`
	TopicID   string         `gorm:"column:topic_id" json:"topicId"`
	Level     int            `gorm:"column:level" json:"level"`
	IsOk      int            `gorm:"column:is_ok" json:"isOk"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
}

func (Ad) TableName() string {
	return "campus_ad"
}
