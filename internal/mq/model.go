package mq

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Milchstrassse/Ecampus-go/internal/comment"
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
