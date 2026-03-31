package notice

import (
	"time"

	"gorm.io/gorm"
)

type Notice struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Content   string         `gorm:"column:content" json:"content"`
	CreatedBy int64          `gorm:"column:created_by" json:"createdBy"`
	UpdatedBy int64          `gorm:"column:updated_by" json:"updatedBy"`
	DeletedBy int64          `gorm:"column:deleted_by" json:"deletedBy"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
}

func (Notice) TableName() string {
	return "campus_notice"
}
