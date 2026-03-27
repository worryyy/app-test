package vote

import (
	"time"

	"gorm.io/gorm"
)

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
