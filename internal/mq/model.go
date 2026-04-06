package mq

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MQLog struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CreatedTime time.Time          `bson:"createdTime" json:"createdTime"`
	Type        string             `bson:"type" json:"type"`
	Data        interface{}        `bson:"data" json:"data"`
}

type MQMessage struct {
	UniqueID int64       `json:"uniqueId"`
	Data     interface{} `json:"data"`
}

type TopicCheckMsg struct {
	TopicID string `json:"topicId"`
}

type AddCommentUser struct {
	UserID      string `json:"userId"`
	Avatar      string `json:"avatar"`
	NickName    string `json:"nickName"`
	AccountType string `json:"accountType"`
	Signature   string `json:"signature"`
}

type AddCommentPayload struct {
	ID          primitive.ObjectID `json:"id"`
	TopicID     string             `json:"topicId"`
	Comment     string             `json:"comment"`
	CreatedTime time.Time          `json:"createdTime"`
	User        AddCommentUser     `json:"user"`
	Parent      *AddCommentUser    `json:"parent,omitempty"`
	ParentCmtID string             `json:"parentCmtId,omitempty"`
	RootCmtID   string             `json:"rootCmtId,omitempty"`
	IsAuthor    bool               `json:"isAuthor"`
	LikeNum     int64              `json:"likeNum"`
	CommentNum  int64              `json:"commentNum"`
	HasCheck    bool               `json:"hasCheck"`
}

type AddCommentMsg struct {
	Comment AddCommentPayload `json:"comment"`
}

type CourseMsg struct {
	UserID int64  `json:"userId"`
	StuNum string `json:"stuNum"`
	StuPwd string `json:"stuPwd"`
	Term   string `json:"term"`
	Week   int    `json:"week"`
}

type NotifyMsg struct {
	TargetUserID string    `json:"targetUserId"`
	SenderUserID string    `json:"senderUserId"`
	Type         string    `json:"type"`
	Content      string    `json:"content"`
	TopicID      string    `json:"topicId"`
	CommentID    string    `json:"commentId"`
	CreatedTime  time.Time `json:"createdTime"`
}

type TopicUserUpdateMsg struct {
	UserID      string `json:"userId"`
	NickName    string `json:"nickName"`
	Avatar      string `json:"avatar"`
	Gender      string `json:"gender"`
	Signature   string `json:"signature"`
	AccountType int    `json:"accountType"`
}

type CommentUserUpdateMsg struct {
	UserID      string `json:"userId"`
	NickName    string `json:"nickName"`
	Avatar      string `json:"avatar"`
	Gender      string `json:"gender"`
	Signature   string `json:"signature"`
	AccountType int    `json:"accountType"`
}

type TopicDeleteMsg struct {
	TopicID string `json:"topicId"`
}

type CommentDeleteMsg struct {
	TopicID   string `json:"topicId"`
	CommentID string `json:"commentId"`
}

type DieMsg struct {
	Queue   string      `json:"queue"`
	Payload interface{} `json:"payload"`
	Reason  string      `json:"reason"`
}
