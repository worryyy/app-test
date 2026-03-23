package comment

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Comment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TopicID     string             `bson:"topicId" json:"topicId"`
	Comment     string             `bson:"comment" json:"comment"`
	CreatedTime time.Time          `bson:"createdTime" json:"createdTime"`
	User        CommentUser        `bson:"user" json:"user"`
	Parent      *CommentUser       `bson:"parent,omitempty" json:"parent"`
	ParentCmtID string             `bson:"parentCmtId,omitempty" json:"parentCmtId"`
	RootCmtID   string             `bson:"rootCmtId,omitempty" json:"rootCmtId"`
	IsAuthor    bool               `bson:"isAuthor" json:"isAuthor"`
	LikeNum     int64              `bson:"likeNum" json:"likeNum"`
	CommentNum  int64              `bson:"commentNum" json:"commentNum"`
	HasCheck    bool               `bson:"hasCheck" json:"hasCheck"`
	HasLike     bool               `bson:"-" json:"hasLike"`
}

type CommentUser struct {
	UserID      string `bson:"userId" json:"userId"`
	NickName    string `bson:"nickName" json:"nickName"`
	Avatar      string `bson:"avatar" json:"avatar"`
	AccountType int    `bson:"accountType" json:"accountType"`
}

type CommentLike struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CommentID string             `bson:"commentId" json:"commentId"`
	UserIDs   []string           `bson:"userIds" json:"userIds"`
}
