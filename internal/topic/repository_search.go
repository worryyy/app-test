package topic

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (r *Repository) SearchHotTopics(
	ctx context.Context,
	filter bson.M,
	page, size int,
) ([]Topic, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}

	coll, err := r.mongoCollection(mongoCollTopic)
	if err != nil {
		return nil, 0, err
	}

	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count hot topics: %w", err)
	}

	sevenDaysAgo := primitive.NewObjectIDFromTimestamp(time.Now().AddDate(0, 0, -7))
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$addFields", Value: bson.M{
			"hotScore": bson.M{"$add": []any{
				bson.M{"$multiply": []any{"$commentNum", 9}},
				bson.M{"$multiply": []any{"$likeNum", 6}},
				bson.M{"$multiply": []any{"$visitedNum", 1}},
			}},
		}}},
		{{Key: "$addFields", Value: bson.M{
			"recentFlag": bson.M{"$cond": bson.M{
				"if":   bson.M{"$gte": []any{"$_id", sevenDaysAgo}},
				"then": 1,
				"else": 0,
			}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "recentFlag", Value: -1}, {Key: "hotScore", Value: -1}, {Key: "_id", Value: -1}}}},
		{{Key: "$skip", Value: int64((page - 1) * size)}},
		{{Key: "$limit", Value: int64(size)}},
	}

	cur, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, fmt.Errorf("aggregate hot topics: %w", err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var topics []Topic
	if err := cur.All(ctx, &topics); err != nil {
		return nil, 0, fmt.Errorf("decode hot topics: %w", err)
	}
	return topics, total, nil
}
