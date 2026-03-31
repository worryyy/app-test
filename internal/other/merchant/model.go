package merchant

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"gorm.io/gorm"
)

type MerchantTheme struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ThemeID string             `bson:"themeId" json:"themeId"`
}

type Task struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Status    int            `gorm:"column:status" json:"status"`
	Detail    string         `gorm:"column:detail" json:"detail"`
	Parent    int64          `gorm:"column:parent" json:"parent"`
	Func      string         `gorm:"column:func" json:"func"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
}

func (Task) TableName() string {
	return "campus_task"
}
