package topic

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *Repository) CreateTopic(ctx context.Context, topic *Topic) (primitive.ObjectID, error) {
	coll, err := r.mongoCollection(mongoCollTopic)
	if err != nil {
		return primitive.NilObjectID, err
	}

	res, err := coll.InsertOne(ctx, topic)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("insert topic: %w", err)
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("inserted topic id type invalid")
	}
	topic.ID = oid
	return oid, nil
}

func (r *Repository) HideTopic(
	ctx context.Context,
	topicID primitive.ObjectID,
	userID string,
	isAdmin bool,
) (bool, error) {
	coll, err := r.mongoCollection(mongoCollTopic)
	if err != nil {
		return false, err
	}

	filter := bson.M{"_id": topicID}
	if !isAdmin {
		filter["userId"] = userID
	}

	res, err := coll.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"hasCheck": false}})
	if err != nil {
		return false, fmt.Errorf("hide topic %s: %w", topicID.Hex(), err)
	}
	return res.MatchedCount > 0, nil
}

func (r *Repository) FindTopicByID(
	ctx context.Context,
	topicID primitive.ObjectID,
	onlyChecked bool,
) (*Topic, error) {
	coll, err := r.mongoCollection(mongoCollTopic)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": topicID}
	if onlyChecked {
		filter["hasCheck"] = true
	}

	var topic Topic
	if err := coll.FindOne(ctx, filter).Decode(&topic); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find topic by id %s: %w", topicID.Hex(), err)
	}
	return &topic, nil
}

func (r *Repository) IncrementTopicField(
	ctx context.Context,
	topicID primitive.ObjectID,
	field string,
	delta int64,
) error {
	if field == "" || delta == 0 {
		return nil
	}

	coll, err := r.mongoCollection(mongoCollTopic)
	if err != nil {
		return err
	}

	if _, err := coll.UpdateByID(ctx, topicID, bson.M{"$inc": bson.M{field: delta}}); err != nil {
		return fmt.Errorf("increment topic %s field %s by %d: %w", topicID.Hex(), field, delta, err)
	}
	return nil
}

func (r *Repository) UpdateTopic(
	ctx context.Context,
	topicID primitive.ObjectID,
	userID string,
	update bson.M,
) (bool, error) {
	coll, err := r.mongoCollection(mongoCollTopic)
	if err != nil {
		return false, err
	}

	res, err := coll.UpdateOne(ctx, bson.M{
		"_id":    topicID,
		"userId": userID,
	}, bson.M{"$set": update})
	if err != nil {
		return false, fmt.Errorf("update topic %s: %w", topicID.Hex(), err)
	}
	return res.MatchedCount > 0, nil
}

func (r *Repository) FindTopicsPage(
	ctx context.Context,
	filter bson.M,
	sort bson.D,
	page, size int,
) ([]Topic, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	if len(sort) == 0 {
		sort = bson.D{{Key: "_id", Value: -1}}
	}

	coll, err := r.mongoCollection(mongoCollTopic)
	if err != nil {
		return nil, 0, err
	}

	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count topics: %w", err)
	}

	opts := options.Find().
		SetSort(sort).
		SetSkip(int64((page - 1) * size)).
		SetLimit(int64(size))
	cur, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("find topics: %w", err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var topics []Topic
	if err := cur.All(ctx, &topics); err != nil {
		return nil, 0, fmt.Errorf("decode topics: %w", err)
	}
	return topics, total, nil
}
