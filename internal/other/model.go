package other

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"gorm.io/gorm"
)

type Ad struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BannerURL string         `gorm:"column:bannerUrl" json:"bannerUrl"`
	TopicID   string         `gorm:"column:topicId" json:"topicId"`
	Level     int            `gorm:"column:level" json:"level"`
	IsOk      bool           `gorm:"column:isOk" json:"isOk"`
	CreatedAt time.Time      `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"column:deletedAt" json:"-"`
}

func (Ad) TableName() string {
	return "campus_ad"
}

type Notice struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Content   string         `gorm:"column:content" json:"content"`
	CreatedBy int64          `gorm:"column:createdBy" json:"createdBy"`
	UpdatedBy int64          `gorm:"column:updatedBy" json:"updatedBy"`
	DeletedBy int64          `gorm:"column:deletedBy" json:"deletedBy"`
	CreatedAt time.Time      `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"column:deletedAt" json:"-"`
}

func (Notice) TableName() string {
	return "campus_notice"
}

type SensitiveWord struct {
	ID   int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Word string `gorm:"column:word" json:"word"`
}

func (SensitiveWord) TableName() string {
	return "sensitive_words"
}

type VoteInfo struct {
	ID            int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Title         string         `gorm:"column:title" json:"title"`
	Content       string         `gorm:"column:content" json:"content"`
	AccessDraft   bool           `gorm:"column:accessDraft" json:"accessDraft"`
	AccessEndTime time.Time      `gorm:"column:accessEndTime" json:"accessEndTime"`
	VoteStart     time.Time      `gorm:"column:voteStart" json:"voteStart"`
	VoteEnd       time.Time      `gorm:"column:voteEnd" json:"voteEnd"`
	Mode          int            `gorm:"column:mode" json:"mode"`
	OptionType    int            `gorm:"column:optionType" json:"optionType"`
	UserID        int64          `gorm:"column:userId" json:"userId"`
	CreatedAt     time.Time      `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time      `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deletedAt" json:"-"`
}

func (VoteInfo) TableName() string {
	return "campus_vote_info"
}

type VoteOption struct {
	ID         int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	VoteInfoID int64          `gorm:"column:voteInfoId" json:"voteInfoId"`
	VoteText   string         `gorm:"column:voteText" json:"voteText"`
	VoteImg    string         `gorm:"column:voteImg" json:"voteImg"`
	IsOk       bool           `gorm:"column:isOk" json:"isOk"`
	CreatedAt  time.Time      `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt  time.Time      `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deletedAt" json:"-"`
}

func (VoteOption) TableName() string {
	return "campus_vote_option"
}

type VoteAns struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	VoteInfoID   int64     `gorm:"column:voteInfoId" json:"voteInfoId"`
	VoteDate     time.Time `gorm:"column:voteDate" json:"voteDate"`
	VoteUserID   int64     `gorm:"column:voteUserId" json:"voteUserId"`
	VoteOptionID int64     `gorm:"column:voteOptionId" json:"voteOptionId"`
}

func (VoteAns) TableName() string {
	return "campus_vote_ans"
}

type Task struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Status    int            `gorm:"column:status" json:"status"`
	Detail    string         `gorm:"column:detail" json:"detail"`
	Parent    int64          `gorm:"column:parent" json:"parent"`
	Func      string         `gorm:"column:func" json:"func"`
	CreatedAt time.Time      `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updatedAt;autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"column:deletedAt" json:"-"`
}

func (Task) TableName() string {
	return "campus_task"
}

type ReportComment struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CommentID      string             `bson:"commentId" json:"commentId"`
	ReportContent  string             `bson:"reportContent" json:"reportContent"`
	CreatedTime    time.Time          `bson:"createdTime" json:"createdTime"`
	ReportUserID   string             `bson:"reportUserId" json:"reportUserId"`
	HasHandle      bool               `bson:"hasHandle" json:"hasHandle"`
	HandlerContent string             `bson:"handlerContent" json:"handlerContent"`
	HandlerUserID  string             `bson:"handlerUserId" json:"handlerUserId"`
	HandlerTime    *time.Time         `bson:"handlerTime" json:"handlerTime"`
}

type MerchantTheme struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ThemeID string             `bson:"themeId" json:"themeId"`
}

type FrontendSupport struct {
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Key   string             `bson:"key" json:"key"`
	Value interface{}        `bson:"value" json:"value"`
}
