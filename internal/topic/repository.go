package topic

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

const (
	mongoCollTopic           = "campus_topic"
	mongoCollThemeID         = "campus_theme_id"
	mongoCollFollow          = "campus_follow"
	mongoCollTopicLike       = "campus_topic_like"
	mongoCollTopicCollection = "campus_topic_collection"
)

type Repository struct {
	db      *gorm.DB
	mongoDB *mongo.Database
}

type topicAuthor struct {
	ID          int64  `gorm:"column:id"`
	Nickname    string `gorm:"column:nickname"`
	Avatar      string `gorm:"column:avatar"`
	AccountType string `gorm:"column:account_type"`
	RootUserID  int64  `gorm:"column:root_user_id"`
}

func (topicAuthor) TableName() string {
	return "campus_user"
}

type campusThemeID struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Name    string             `bson:"name"`
	ThemeID string             `bson:"themeId"`
}

type topicStateDoc struct {
	TopicIDs []string `bson:"topicIds"`
}

type followDoc struct {
	FollowingID int64 `bson:"followingId"`
}

func NewRepository(db *gorm.DB, mongoDB *mongo.Database) *Repository {
	return &Repository{
		db:      db,
		mongoDB: mongoDB,
	}
}

func (r *Repository) gormDB(ctx context.Context) (*gorm.DB, error) {
	if r.db == nil {
		return nil, errors.New("gorm db not initialized")
	}
	return r.db.WithContext(ctx), nil
}

func (r *Repository) mongoCollection(name string) (*mongo.Collection, error) {
	if r.mongoDB == nil {
		return nil, errors.New("mongo db not initialized")
	}
	return r.mongoDB.Collection(name), nil
}
