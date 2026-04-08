package user

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

type Repository struct {
	db      *gorm.DB
	mongoDB *mongo.Database
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
