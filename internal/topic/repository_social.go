package topic

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *Repository) FindTopicStateDocs(
	ctx context.Context,
	collName string,
	userID string,
) ([]topicStateDoc, error) {
	coll, err := r.mongoCollection(collName)
	if err != nil {
		return nil, err
	}

	cur, err := coll.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return nil, fmt.Errorf("find %s docs: %w", collName, err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var docs []topicStateDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode %s docs: %w", collName, err)
	}
	return docs, nil
}

func (r *Repository) AddTopicState(
	ctx context.Context,
	collName string,
	userID, themeName, accountType, topicID string,
) (bool, error) {
	coll, err := r.mongoCollection(collName)
	if err != nil {
		return false, err
	}

	res, err := coll.UpdateOne(ctx, bson.M{
		"userId":    userID,
		"themeName": themeName,
	}, bson.M{
		"$setOnInsert": bson.M{
			"userId":      userID,
			"themeName":   themeName,
			"accountType": accountType,
		},
		"$addToSet": bson.M{"topicIds": topicID},
	}, options.Update().SetUpsert(true))
	if err != nil {
		return false, fmt.Errorf("update %s topic state: %w", collName, err)
	}
	return res.ModifiedCount > 0 || res.UpsertedCount > 0, nil
}

func (r *Repository) RemoveTopicState(
	ctx context.Context,
	collName string,
	userID, topicID string,
) (bool, error) {
	coll, err := r.mongoCollection(collName)
	if err != nil {
		return false, err
	}

	res, err := coll.UpdateMany(ctx, bson.M{"userId": userID}, bson.M{"$pull": bson.M{"topicIds": topicID}})
	if err != nil {
		return false, fmt.Errorf("remove topic %s from %s: %w", topicID, collName, err)
	}
	return res.ModifiedCount > 0, nil
}

func (r *Repository) CountTopicStateItems(ctx context.Context, collName, userID string) (int64, error) {
	docs, err := r.FindTopicStateDocs(ctx, collName, userID)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, doc := range docs {
		total += int64(len(doc.TopicIDs))
	}
	return total, nil
}

func (r *Repository) FindTopicsByIDs(
	ctx context.Context,
	topicIDs []primitive.ObjectID,
	onlyChecked bool,
) ([]Topic, error) {
	if len(topicIDs) == 0 {
		return []Topic{}, nil
	}

	coll, err := r.mongoCollection(mongoCollTopic)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": bson.M{"$in": topicIDs}}
	if onlyChecked {
		filter["hasCheck"] = true
	}

	cur, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find topics by ids: %w", err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var topics []Topic
	if err := cur.All(ctx, &topics); err != nil {
		return nil, fmt.Errorf("decode topics by ids: %w", err)
	}
	return topics, nil
}
