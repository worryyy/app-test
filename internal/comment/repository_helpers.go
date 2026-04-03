package comment

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

func (r *Repository) FindUserByID(ctx context.Context, userID int64) (*userRecord, error) {
	if userID <= 0 {
		return nil, nil
	}

	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var user userRecord
	if err := db.Table(user.TableName()).Where("id = ?", userID).Take(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user %d: %w", userID, err)
	}
	return &user, nil
}

func (r *Repository) FindTopicByID(
	ctx context.Context,
	topicID primitive.ObjectID,
	onlyChecked bool,
) (*CommentTopic, error) {
	coll, err := r.mongoCollection(mongoCollTopic)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": topicID}
	if onlyChecked {
		filter["hasCheck"] = true
	}

	var topic CommentTopic
	if err := coll.FindOne(ctx, filter).Decode(&topic); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find topic %s: %w", topicID.Hex(), err)
	}
	return &topic, nil
}

func (r *Repository) FindCommentByID(ctx context.Context, commentID primitive.ObjectID) (*Comment, error) {
	coll, err := r.mongoCollection(mongoCollComment)
	if err != nil {
		return nil, err
	}

	var cmt Comment
	if err := coll.FindOne(ctx, bson.M{"_id": commentID}).Decode(&cmt); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find comment %s: %w", commentID.Hex(), err)
	}
	return &cmt, nil
}
