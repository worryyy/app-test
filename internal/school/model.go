package school

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

type campusUser struct {
	ID                   int64      `gorm:"column:id"`
	StuNum               string     `gorm:"column:stu_num"`
	StuName              string     `gorm:"column:stu_name"`
	StuPwd               string     `gorm:"column:stu_pwd"`
	StuCla               string     `gorm:"column:stu_cla"`
	StuIsCheck           bool       `gorm:"column:stu_is_check"`
	ProvisionalExpiresAt *time.Time `gorm:"column:provisional_expires_at"`
	School               string     `gorm:"column:school"`
}

func (campusUser) TableName() string {
	return "campus_user"
}
