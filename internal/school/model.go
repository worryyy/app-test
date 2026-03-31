package school

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserCourse struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"column:user_id" json:"userId"`
	Status    int       `gorm:"column:status" json:"status"`
	Term      string    `gorm:"column:term" json:"term"`
	Week      int       `gorm:"column:week" json:"week"`
	Course    string    `gorm:"column:course;type:text" json:"course"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"-"`
}

func (UserCourse) TableName() string {
	return "campus_user_course"
}

type CourseColor struct {
	UserID     int64     `gorm:"column:user_id;primaryKey" json:"userId"`
	CourseName string    `gorm:"column:course_name;primaryKey" json:"courseName"`
	Color      string    `gorm:"column:color" json:"color"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"-"`
}

func (CourseColor) TableName() string {
	return "campus_course_color"
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

type CurDateAndTerm struct {
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

type weekCourse struct {
	CourseList []innerCourse `json:"courseList"`
}

type innerCourse struct {
	Name string `json:"name"`
}
