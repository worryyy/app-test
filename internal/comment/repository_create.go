package comment

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Repository) CreateComment(ctx context.Context, comment *Comment) (primitive.ObjectID, error) {
	coll, err := r.mongoCollection(mongoCollComment)
	if err != nil {
		return primitive.NilObjectID, err
	}

	res, err := coll.InsertOne(ctx, comment)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("insert comment: %w", err)
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, fmt.Errorf("comment inserted id type invalid")
	}
	comment.ID = oid
	return oid, nil
}

func (r *Repository) HideComment(
	ctx context.Context,
	topicID string,
	commentID primitive.ObjectID,
	userID string,
	isAdmin bool,
) (bool, bool, error) {
	coll, err := r.mongoCollection(mongoCollComment)
	if err != nil {
		return false, false, err
	}

	filter := bson.M{"_id": commentID, "topicId": topicID}
	if !isAdmin {
		filter["user.userId"] = userID
	}

	res, err := coll.UpdateOne(ctx, filter, bson.M{
		"$set": bson.M{
			"hasCheck":    false,
			"deletedTime": time.Now(),
		},
	})
	if err != nil {
		return false, false, fmt.Errorf("hide comment %s: %w", commentID.Hex(), err)
	}
	return res.MatchedCount > 0, res.ModifiedCount > 0, nil
}

func (r *Repository) IncrementTopicCommentNum(ctx context.Context, topicID primitive.ObjectID, delta int64) (bool, error) {
	coll, err := r.mongoCollection(mongoCollTopic)
	if err != nil {
		return false, err
	}

	res, err := coll.UpdateByID(ctx, topicID, bson.M{"$inc": bson.M{"commentNum": delta}})
	if err != nil {
		return false, fmt.Errorf("update topic %s commentNum: %w", topicID.Hex(), err)
	}
	return res.MatchedCount > 0, nil
}

func (r *Repository) IncrementRootCommentNum(ctx context.Context, rootCommentID primitive.ObjectID, delta int64) (bool, error) {
	coll, err := r.mongoCollection(mongoCollComment)
	if err != nil {
		return false, err
	}

	res, err := coll.UpdateByID(ctx, rootCommentID, bson.M{"$inc": bson.M{"commentNum": delta}})
	if err != nil {
		return false, fmt.Errorf("update root comment %s commentNum: %w", rootCommentID.Hex(), err)
	}
	return res.MatchedCount > 0, nil
}
