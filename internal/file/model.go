package file

import "go.mongodb.org/mongo-driver/bson/primitive"

type File struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MD5         string             `bson:"md5" json:"md5"`
	IsPublic    bool               `bson:"isPublic" json:"isPublic"`
	UserID      string             `bson:"userId" json:"userId"`
	RefCount    int                `bson:"refCount" json:"refCount"`
	CreatedTime int64              `bson:"-" json:"createdTime"`
}

type CompressFile struct {
	ID  primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MD5 string             `bson:"md5" json:"md5"`
	URL string             `bson:"url" json:"url"`
}
