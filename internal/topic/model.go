package topic

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Topic struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ThemeID       string             `bson:"themeId" json:"themeId"`
	UserID        string             `bson:"user_id" json:"user_id"`
	Title         string             `bson:"title" json:"title"`
	Content       string             `bson:"content" json:"content"`
	Imgs          []string           `bson:"imgs" json:"imgs"`
	HasCheck      bool               `bson:"hasCheck" json:"hasCheck"`
	VisitedNum    int64              `bson:"visitedNum" json:"visitedNum"`
	LikeNum       int64              `bson:"likeNum" json:"likeNum"`
	CommentNum    int64              `bson:"commentNum" json:"commentNum"`
	CollectionNum int64              `bson:"collectionNum" json:"collectionNum"`
	Ext           interface{}        `bson:"ext,omitempty" json:"ext"`
	AccountType   string             `bson:"account_type" json:"account_type"`
	NickName      string             `bson:"nickName" json:"nickName"`
	Avatar        string             `bson:"avatar" json:"avatar"`
	CreatedTime   *time.Time         `bson:"-" json:"createdTime"`
	HasLike       bool               `bson:"-" json:"hasLike"`
	HasCollection bool               `bson:"-" json:"hasCollection"`
}

type TopicSearch struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TopicID   string             `bson:"topicId" json:"topicId"`
	ThemeName string             `bson:"themeName" json:"themeName"`
	Title     string             `bson:"title" json:"title"`
	Content   string             `bson:"content" json:"content"`
}

type TopicLike struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      string             `bson:"user_id" json:"user_id"`
	ThemeName   string             `bson:"themeName" json:"themeName"`
	AccountType string             `bson:"account_type" json:"account_type"`
	TopicIDs    []string           `bson:"topicIds" json:"topicIds"`
}

type TopicCollection struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      string             `bson:"user_id" json:"user_id"`
	ThemeName   string             `bson:"themeName" json:"themeName"`
	AccountType string             `bson:"account_type" json:"account_type"`
	TopicIDs    []string           `bson:"topicIds" json:"topicIds"`
}

type SuggestList struct {
	Total   int64   `json:"total"`
	CurPage int     `json:"curPage"`
	Size    int     `json:"size"`
	Data    []Topic `json:"data"`
}
