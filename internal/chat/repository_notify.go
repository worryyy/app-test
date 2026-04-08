package chat

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *Repository) FindNotificationsPage(
	ctx context.Context,
	receiverID, typ string,
	page, size int,
) ([]Notification, int64, error) {
	coll, err := r.mongoCollection(mongoCollNotify)
	if err != nil {
		return nil, 0, err
	}

	filter := bson.M{
		"receiver_id": receiverID,
		"type":        typ,
	}
	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "_id", Value: -1}}).
		SetSkip(int64((page - 1) * size)).
		SetLimit(int64(size))
	cur, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("find notifications: %w", err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var notifications []Notification
	if err := cur.All(ctx, &notifications); err != nil {
		return nil, 0, fmt.Errorf("decode notifications: %w", err)
	}
	if notifications == nil {
		return []Notification{}, total, nil
	}
	return notifications, total, nil
}

func (r *Repository) MarkLatestNotificationRead(ctx context.Context, receiverID, typ string) error {
	coll, err := r.mongoCollection(mongoCollNotify)
	if err != nil {
		return err
	}

	var notification Notification
	err = coll.FindOneAndUpdate(
		ctx,
		bson.M{
			"receiver_id": receiverID,
			"type":        typ,
		},
		bson.M{"$set": bson.M{"is_read": true}},
		options.FindOneAndUpdate().SetSort(bson.D{{Key: "created_time", Value: -1}}),
	).Decode(&notification)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("mark latest notification read: %w", err)
	}
	return nil
}

func (r *Repository) FindLatestNotification(ctx context.Context, receiverID, typ string) (*Notification, error) {
	coll, err := r.mongoCollection(mongoCollNotify)
	if err != nil {
		return nil, err
	}

	var notification Notification
	if err := coll.FindOne(
		ctx,
		bson.M{
			"receiver_id": receiverID,
			"type":        typ,
		},
		options.FindOne().SetSort(bson.D{{Key: "created_time", Value: -1}}),
	).Decode(&notification); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find latest notification: %w", err)
	}
	return &notification, nil
}
