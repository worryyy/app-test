package user

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OpenID       string         `gorm:"column:open_id" json:"-"`
	Nickname     string         `gorm:"column:nickname" json:"nickname"`
	Avatar       string         `gorm:"column:avatar" json:"avatar"`
	Power        int            `gorm:"column:power" json:"power"`
	AccountType  string         `gorm:"column:account_type;default:base" json:"accountType"`
	StuNum       string         `gorm:"column:stu_num" json:"stuNum"`
	StuName      string         `gorm:"column:stu_name" json:"stuName"`
	StuPwd       string         `gorm:"column:stu_pwd" json:"stuPwd"`
	StuCla       string         `gorm:"column:stu_cla" json:"stuCla"`
	StuIsCheck   bool           `gorm:"column:stu_is_check" json:"stuIsCheck"`
	School       string         `gorm:"column:school" json:"school"`
	Tag          string         `gorm:"column:tag;default:student" json:"tag"`
	Gender       string         `gorm:"column:gender;default:保密" json:"gender"`
	RootUserID   int64          `gorm:"column:root_user_id" json:"rootUserId"`
	LastSwitchID *int64         `gorm:"column:last_switch_id" json:"lastSwitchId"`
	Signature    string         `gorm:"column:signature" json:"signature"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime" json:"-"`
	CreatedBy    int64          `gorm:"column:created_by" json:"-"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"-"`
	UpdatedBy    int64          `gorm:"column:updated_by" json:"-"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
	DeletedBy    int64          `gorm:"column:deleted_by" json:"-"`
}

func (User) TableName() string {
	return "campus_user"
}

type Admin struct {
	ID       int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID   int64  `gorm:"column:user_id;uniqueIndex" json:"userId"`
	Username string `gorm:"column:username;uniqueIndex" json:"username"`
	Password string `gorm:"column:password" json:"-"`
	Power    int    `gorm:"column:power;default:2" json:"power"`
}

func (Admin) TableName() string {
	return "admin"
}

type AdminLoginReq struct {
	Username          string `json:"username" binding:"required"`
	Password          string `json:"password" binding:"required"`
	SecondaryPassword string `json:"secondaryPassword" binding:"required"`
}
