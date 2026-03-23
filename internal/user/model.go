package user

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"gorm.io/gorm"
)

type User struct {
	ID           int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OpenID       string         `gorm:"column:openId" json:"openId"`
	Nickname     string         `gorm:"column:nickname" json:"nickname"`
	Avatar       string         `gorm:"column:avatar" json:"avatar"`
	Power        int            `gorm:"column:power" json:"power"`
	AccountType  string         `gorm:"column:accountType;default:base" json:"accountType"`
	StuNum       string         `gorm:"column:stuNum" json:"stuNum"`
	StuName      string         `gorm:"column:stuName" json:"stuName"`
	StuPwd       string         `gorm:"column:stuPwd" json:"-"`
	StuCla       string         `gorm:"column:stuCla" json:"stuCla"`
	StuIsCheck   bool           `gorm:"column:stuIsCheck" json:"stuIsCheck"`
	School       string         `gorm:"column:school" json:"school"`
	Tag          string         `gorm:"column:tag;default:student" json:"tag"`
	Gender       string         `gorm:"column:gender;default:保密" json:"gender"`
	RootUserID   int64          `gorm:"column:rootUserId" json:"rootUserId"`
	LastSwitchID *int64         `gorm:"column:lastSwitchId" json:"lastSwitchId"`
	Signature    string         `gorm:"column:signature" json:"signature"`
	CreatedAt    time.Time      `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	CreatedBy    int64          `gorm:"column:createdBy" json:"createdBy"`
	UpdatedAt    time.Time      `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
	UpdatedBy    int64          `gorm:"column:updatedBy" json:"updatedBy"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deletedAt" json:"-"`
	DeletedBy    int64          `gorm:"column:deletedBy" json:"-"`
}

func (User) TableName() string {
	return "campus_user"
}

type Admin struct {
	ID       int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID   int64  `gorm:"column:userId;uniqueIndex" json:"userId"`
	Username string `gorm:"column:username;uniqueIndex" json:"username"`
	Password string `gorm:"column:password" json:"-"`
	Power    int    `gorm:"column:power;default:2" json:"power"`
}

func (Admin) TableName() string {
	return "admin"
}

type Follow struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FollowerID  string             `bson:"followerId" json:"followerId"`
	FollowingID string             `bson:"followingId" json:"followingId"`
	FollowAt    time.Time          `bson:"followAt" json:"followAt"`
}

type UserBlacklist struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BlockedUserIDs []string           `bson:"blocked_user_ids" json:"blockedUserIds"`
	CreatedTime    time.Time          `bson:"created_time" json:"createdTime"`
	UpdatedTime    time.Time          `bson:"updated_time" json:"updatedTime"`
}

type OfficialCertification struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string             `bson:"userId" json:"userId"`
	Name      string             `bson:"name" json:"name"`
	Reason    string             `bson:"reason" json:"reason"`
	Status    int                `bson:"status" json:"status"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

type AdminLoginReq struct {
	Username          string `json:"username" binding:"required"`
	Password          string `json:"password" binding:"required"`
	SecondaryPassword string `json:"secondaryPassword" binding:"required"`
}

type UserProfile struct {
	User
	FollowerCount  int64 `json:"followerCount"`
	FollowingCount int64 `json:"followingCount"`
	LikeCount      int64 `json:"likeCount"`
	TopicCount     int64 `json:"topicCount"`
	IsFollowing    bool  `json:"isFollowing"`
}
