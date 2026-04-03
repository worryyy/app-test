package comment

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

const (
	mongoCollComment     = "campus_comment"
	mongoCollCommentLike = "campus_comment_like"
	mongoCollTopic       = "campus_topic"
)

type Repository struct {
	db      *gorm.DB
	mongoDB *mongo.Database
}

type userRecord struct {
	ID          int64  `gorm:"column:id"`
	Nickname    string `gorm:"column:nickname"`
	Avatar      string `gorm:"column:avatar"`
	AccountType string `gorm:"column:account_type"`
	Power       int    `gorm:"column:power"`
	Gender      string `gorm:"column:gender"`
	RootUserID  int64  `gorm:"column:root_user_id"`
	Signature   string `gorm:"column:signature"`
}

func (userRecord) TableName() string {
	return "campus_user"
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

func commentIDString(id primitive.ObjectID) string {
	if id.IsZero() {
		return ""
	}
	return id.Hex()
}
