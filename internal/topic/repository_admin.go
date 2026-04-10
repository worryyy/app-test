package topic

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Repository) UpdateTopicAdmin(
	ctx context.Context,
	topicID primitive.ObjectID,
	update bson.M,
) (bool, error) {
	coll, err := r.mongoCollection(mongoCollTopic)
	if err != nil {
		return false, err
	}

	res, err := coll.UpdateOne(ctx, bson.M{"_id": topicID}, bson.M{"$set": update})
	if err != nil {
		return false, fmt.Errorf("admin update topic %s: %w", topicID.Hex(), err)
	}
	return res.MatchedCount > 0, nil
}

func (r *Repository) CleanupDeletedTopic(ctx context.Context, topicID string) error {
	if _, err := r.removeTopicFromStateCollection(ctx, mongoCollTopicLike, topicID); err != nil {
		return err
	}
	if _, err := r.removeTopicFromStateCollection(ctx, mongoCollTopicCollection, topicID); err != nil {
		return err
	}

	coll, err := r.mongoCollection(mongoCollComment)
	if err != nil {
		return err
	}
	if _, err := coll.UpdateMany(
		ctx,
		bson.M{"topicId": topicID},
		bson.M{"$set": bson.M{"hasCheck": false}},
	); err != nil {
		return fmt.Errorf("hide topic comments %s: %w", topicID, err)
	}
	return nil
}

func (r *Repository) removeTopicFromStateCollection(
	ctx context.Context,
	collName string,
	topicID string,
) (int64, error) {
	coll, err := r.mongoCollection(collName)
	if err != nil {
		return 0, err
	}

	res, err := coll.UpdateMany(ctx, bson.M{}, bson.M{"$pull": bson.M{"topicIds": topicID}})
	if err != nil {
		return 0, fmt.Errorf("remove topic %s from %s: %w", topicID, collName, err)
	}
	return res.ModifiedCount, nil
}
