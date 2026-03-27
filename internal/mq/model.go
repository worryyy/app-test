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
	UserID int64  `json:"user_id"`
	StuNum string `json:"stu_num"`
	StuPwd string `json:"stu_pwd"`
	Term   string `json:"term"`
	Week   int    `json:"week"`
}

type NotifyMsg struct {
	TargetUserID string    `json:"targetUserId"`
	SenderUserID string    `json:"senderUserId,omitempty"`
	Type         string    `json:"type"`
	Content      string    `json:"content"`
	TopicID      string    `json:"topicId,omitempty"`
	CommentID    string    `json:"commentId,omitempty"`
	CreatedTime  time.Time `json:"createdTime,omitempty"`
}

type TopicUserUpdateMsg struct {
	UserID      string `json:"user_id"`
	NickName    string `json:"nickName"`
	Avatar      string `json:"avatar"`
	Gender      string `json:"gender"`
	Signature   string `json:"signature"`
	AccountType int    `json:"account_type"`
}

type CommentUserUpdateMsg struct {
	UserID      string `json:"user_id"`
	NickName    string `json:"nickName"`
	Avatar      string `json:"avatar"`
	Gender      string `json:"gender"`
	Signature   string `json:"signature"`
	AccountType int    `json:"account_type"`
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
