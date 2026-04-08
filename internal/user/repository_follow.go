package user

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	mongoCollFollow = "campus_follow"
	mongoCollTopic  = "campus_topic"
)

func (r *Repository) IsFollowing(ctx context.Context, followerID, followingID int64) (bool, error) {
	coll, err := r.mongoCollection(mongoCollFollow)
	if err != nil {
		return false, err
	}

	count, err := coll.CountDocuments(ctx, bson.M{
		"followerId":  followerID,
		"followingId": followingID,
	})
	if err != nil {
		return false, fmt.Errorf("count follow relation: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) CreateFollow(ctx context.Context, followerID, followingID int64, followAt time.Time) error {
	coll, err := r.mongoCollection(mongoCollFollow)
	if err != nil {
		return err
	}

	if _, err := coll.InsertOne(ctx, Follow{
		FollowerID:  followerID,
		FollowingID: followingID,
		FollowAt:    followAt,
	}); err != nil {
		return fmt.Errorf("create follow relation: %w", err)
	}
	return nil
}

func (r *Repository) DeleteFollow(ctx context.Context, followerID, followingID int64) (bool, error) {
	coll, err := r.mongoCollection(mongoCollFollow)
	if err != nil {
		return false, err
	}

	res, err := coll.DeleteOne(ctx, bson.M{
		"followerId":  followerID,
		"followingId": followingID,
	})
	if err != nil {
		return false, fmt.Errorf("delete follow relation: %w", err)
	}
	return res.DeletedCount > 0, nil
}

func (r *Repository) CountFollowers(ctx context.Context, userID int64) (int64, error) {
	coll, err := r.mongoCollection(mongoCollFollow)
	if err != nil {
		return 0, err
	}

	count, err := coll.CountDocuments(ctx, bson.M{"followingId": userID})
	if err != nil {
		return 0, fmt.Errorf("count followers: %w", err)
	}
	return count, nil
}

func (r *Repository) CountFollowings(ctx context.Context, userID int64) (int64, error) {
	coll, err := r.mongoCollection(mongoCollFollow)
	if err != nil {
		return 0, err
	}

	count, err := coll.CountDocuments(ctx, bson.M{"followerId": userID})
	if err != nil {
		return 0, fmt.Errorf("count followings: %w", err)
	}
	return count, nil
}

func (r *Repository) SumTopicLikesByUser(ctx context.Context, userID int64) (int64, error) {
	coll, err := r.mongoCollection(mongoCollTopic)
	if err != nil {
		return 0, err
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"userId":   strconv.FormatInt(userID, 10),
			"hasCheck": true,
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$likeNum"},
		}}},
	}

	cur, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("aggregate topic likes: %w", err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var result []struct {
		Total int64 `bson:"total"`
	}
	if err := cur.All(ctx, &result); err != nil {
		return 0, fmt.Errorf("decode topic like aggregate: %w", err)
	}
	if len(result) == 0 {
		return 0, nil
	}
	return result[0].Total, nil
}
