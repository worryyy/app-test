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

func (r *Repository) FindTopicStateDocs(
	ctx context.Context,
	collName string,
	userID string,
	accountType string,
) (*topicStateDoc, error) {
	coll, err := r.mongoCollection(collName)
	if err != nil {
		return nil, err
	}

	var docs topicStateDoc
	if err := coll.FindOne(ctx, bson.M{"userId": userID, "accountType": accountType}).Decode(&docs); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find %s docs: %w", collName, err)
	}

	return &docs, nil
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
		"userId":      userID,
		"accountType": accountType,
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
	userID, accountType, topicID string,
) (bool, error) {
	coll, err := r.mongoCollection(collName)
	if err != nil {
		return false, err
	}

	res, err := coll.UpdateOne(ctx, bson.M{
		"userId":      userID,
		"accountType": accountType,
	}, bson.M{"$pull": bson.M{"topicIds": topicID}})
	if err != nil {
		return false, fmt.Errorf("remove topic %s from %s: %w", topicID, collName, err)
	}
	return res.ModifiedCount > 0, nil
}

func (r *Repository) CountTopicStateItems(ctx context.Context, collName, userID, accountType string) (int64, error) {
	docs, err := r.FindTopicStateDocs(ctx, collName, userID, accountType)
	if err != nil {
		return 0, err
	}
	if docs == nil {
		return 0, nil
	}

	return int64(len(docs.TopicIDs)), nil
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
