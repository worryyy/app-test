package theme

import (
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

const (
	mongoCollTheme       = "campus_theme"
	mongoCollCampusTheme = "campus_theme_id"
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

func (r *Repository) themeCollection() (*mongo.Collection, error) {
	if r.mongoDB == nil {
		return nil, errors.New("mongo db not initialized")
	}
	return r.mongoDB.Collection(mongoCollTheme), nil
}

func (r *Repository) campusThemeCollection() (*mongo.Collection, error) {
	if r.mongoDB == nil {
		return nil, errors.New("mongo db not initialized")
	}
	return r.mongoDB.Collection(mongoCollCampusTheme), nil
}
