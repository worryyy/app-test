package file

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (r *Repository) FindFileByMD5(ctx context.Context, md5Value string) (*File, error) {
	if md5Value == "" {
		return nil, nil
	}

	coll, err := r.fileCollection()
	if err != nil {
		return nil, err
	}

	var file File
	if err := coll.FindOne(ctx, bson.M{"md5": md5Value}).Decode(&file); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find file by md5 %s: %w", md5Value, err)
	}
	return &file, nil
}

func (r *Repository) FindFileByUserAndMD5(ctx context.Context, userID, md5Value string) (*File, error) {
	if userID == "" || md5Value == "" {
		return nil, nil
	}

	coll, err := r.fileCollection()
	if err != nil {
		return nil, err
	}

	var file File
	if err := coll.FindOne(ctx, bson.M{"userId": userID, "md5": md5Value}).Decode(&file); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find file by user %s and md5 %s: %w", userID, md5Value, err)
	}
	return &file, nil
}
