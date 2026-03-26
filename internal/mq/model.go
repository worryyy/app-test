package mq

import (
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/comment"
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

type AddCommentMsg struct {
	Comment comment.Comment `json:"comment"`
}

type AddTopicSearchMsg struct {
	TopicID string `json:"topicId"`
}

type CourseMsg struct {
	UserID int64  `json:"userId"`
	StuNum string `json:"stuNum"`
	StuPwd string `json:"stuPwd"`
	Term   string `json:"term"`
	Week   int    `json:"week"`
}

type NotifyMsg struct {
	TargetUserID string      `json:"targetUserId"`
	Type         string      `json:"type"`
	Content      interface{} `json:"content"`
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
	Reason  string      `json:"reason,omitempty"`
}
