package school

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserCourse struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"column:userId" json:"userId"`
	Status    int       `gorm:"column:status" json:"status"`
	Term      string    `gorm:"column:term" json:"term"`
	Week      int       `gorm:"column:week" json:"week"`
	Course    string    `gorm:"column:course;type:text" json:"course"`
	UpdatedAt time.Time `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
}

func (UserCourse) TableName() string {
	return "campus_user_course"
}

type Term struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Term       string             `bson:"term" json:"term"`
	StartDate  string             `bson:"startDate" json:"startDate"`
	TotalWeeks int                `bson:"totalWeeks" json:"totalWeeks"`
}

type CurTerm struct {
	ID   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Term string             `bson:"term" json:"term"`
}

type CurDateAndTermVO struct {
	CurDate    string `json:"curDate"`
	CurTerm    string `json:"curTerm"`
	StartDate  string `json:"startDate"`
	TotalWeeks int    `json:"totalWeeks"`
}

type Course struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Key      string             `bson:"key" json:"key"`
	FilePath string             `bson:"filePath" json:"filePath"`
	Val      []byte             `bson:"val" json:"val"`
}
