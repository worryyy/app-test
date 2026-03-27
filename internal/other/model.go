package other

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"gorm.io/gorm"
)

type Ad struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	BannerURL string         `gorm:"column:banner_url" json:"bannerUrl"`
	TopicID   string         `gorm:"column:topic_id" json:"topicId"`
	Level     int            `gorm:"column:level" json:"level"`
	IsOk      bool           `gorm:"column:is_ok" json:"is_ok"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
}

func (Ad) TableName() string {
	return "campus_ad"
}

type Notice struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Content   string         `gorm:"column:content" json:"content"`
	CreatedBy int64          `gorm:"column:created_by" json:"createdBy"`
	UpdatedBy int64          `gorm:"column:updated_by" json:"updatedBy"`
	DeletedBy int64          `gorm:"column:deleted_by" json:"deletedBy"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
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
	AccessDraft   int            `gorm:"column:access_draft" json:"accessDraft"`
	AccessEndTime time.Time      `gorm:"column:access_end_time" json:"accessEndTime"`
	VoteStart     time.Time      `gorm:"column:vote_start" json:"voteStart"`
	VoteEnd       time.Time      `gorm:"column:vote_end" json:"voteEnd"`
	Mode          int            `gorm:"column:mode" json:"mode"`
	OptionType    int            `gorm:"column:option_type" json:"optionType"`
	CreatedBy     int64          `gorm:"column:created_by" json:"createdBy"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
}

func (VoteInfo) TableName() string {
	return "campus_vote_info"
}

type VoteOption struct {
	ID         int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	VoteInfoID int64          `gorm:"column:vote_info_id" json:"voteInfoId"`
	VoteText   string         `gorm:"column:vote_text" json:"voteText"`
	VoteImg    string         `gorm:"column:vote_img" json:"voteImg"`
	CreatedBy  int64          `gorm:"column:created_by" json:"createdBy"`
	IsOk       int            `gorm:"column:is_ok" json:"is_ok"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
}

func (VoteOption) TableName() string {
	return "campus_vote_option"
}

type VoteAns struct {
	VoteInfoID   int64     `gorm:"column:vote_info_id" json:"voteInfoId"`
	VoteDate     time.Time `gorm:"column:vote_date" json:"voteDate"`
	VoteUserID   int64     `gorm:"column:vote_user_id" json:"voteUserId"`
	VoteOptionID int64     `gorm:"column:vote_option_id" json:"voteOptionId"`
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
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
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
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Key     string             `bson:"key" json:"key"`
	Val     string             `bson:"val" json:"val"`
	KeyDesc string             `bson:"keyDesc,omitempty" json:"keyDesc"`
}
