package comment

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const DefaultRootCommentID = "0"

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
	Avatar      string `bson:"avatar" json:"avatar"`
	NickName    string `bson:"nickName" json:"nickName"`
	Gender      string `bson:"gender" json:"gender"`
	AccountType string `bson:"accountType" json:"accountType"`
	Signature   string `bson:"signature" json:"signature"`
}

type CommentLike struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CommentID string             `bson:"commentId" json:"commentId"`
	UserIDs   []string           `bson:"userIds" json:"userIds"`
}

type CommentTopic struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ThemeID       string             `bson:"themeId" json:"themeId"`
	UserID        string             `bson:"userId" json:"userId"`
	Avatar        string             `bson:"avatar" json:"avatar"`
	NickName      string             `bson:"nickName" json:"nickName"`
	AccountType   string             `bson:"accountType" json:"accountType"`
	Title         string             `bson:"title" json:"title"`
	Content       string             `bson:"content" json:"content"`
	Imgs          []string           `bson:"imgs" json:"imgs"`
	HasCheck      bool               `bson:"hasCheck" json:"hasCheck"`
	Ext           interface{}        `bson:"ext,omitempty" json:"ext"`
	VisitedNum    int64              `bson:"visitedNum" json:"visitedNum"`
	LikeNum       int64              `bson:"likeNum" json:"likeNum"`
	CommentNum    int64              `bson:"commentNum" json:"commentNum"`
	CollectionNum int64              `bson:"collectionNum" json:"collectionNum"`
	CreatedTime   *time.Time         `bson:"-" json:"createdTime"`
	HasLike       bool               `bson:"-" json:"hasLike"`
	HasCollection bool               `bson:"-" json:"hasCollection"`
}

type MyCommentVO struct {
	Comment Comment      `json:"comment"`
	Topic   CommentTopic `json:"topic"`
}
