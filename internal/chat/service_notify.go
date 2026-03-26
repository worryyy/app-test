package chat

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) ListNotifications(
	ctx context.Context,
	userID int64,
	typ string,
	page, size int,
) (*result.CusPage[Notification], error) {
	if strings.TrimSpace(typ) == "" {
		return nil, newFail("type 不能为空")
	}
	page, size = normalizePage(page, size, s.defaultPageSize())

	filter := bson.M{
		"receiver_id": userIDString(userID),
		"type":        typ,
	}
	findOpts := options.Find().
		SetSort(bson.D{{Key: "_id", Value: -1}}).
		SetSkip(int64((page - 1) * size)).
		SetLimit(int64(size))
	cur, err := s.notifyColl().Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("find notifications: %w", err)
	}
	defer closeCursor(ctx, s.logger, cur, "close notification cursor failed")

	var list []Notification
	if err := cur.All(ctx, &list); err != nil {
		return nil, fmt.Errorf("decode notifications: %w", err)
	}
	if list == nil {
		list = []Notification{}
	}
	if len(list) == 0 {
		return result.NewCusPage(list, 0, page, size), nil
	}

	var updated Notification
	err = s.notifyColl().FindOneAndUpdate(
		ctx,
		filter,
		bson.M{"$set": bson.M{"is_read": true}},
		options.FindOneAndUpdate().SetSort(bson.D{{Key: "created_time", Value: -1}}),
	).Decode(&updated)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("mark latest notification read: %w", err)
	}
	return result.NewCusPage(list, int64(len(list)), page, size), nil
}

func (s *Service) HaveUnreadNotification(ctx context.Context, userID int64, typ string) (bool, error) {
	notification, err := s.LatestNotification(ctx, userID, typ)
	if err != nil {
		return false, err
	}
	if notification == nil {
		return false, nil
	}
	return !notification.IsRead, nil
}

func (s *Service) LatestNotification(ctx context.Context, userID int64, typ string) (*Notification, error) {
	if strings.TrimSpace(typ) == "" {
		return nil, newFail("type 不能为空")
	}

	filter := bson.M{
		"receiver_id": userIDString(userID),
		"type":        typ,
	}

	var notification Notification
	err := s.notifyColl().
		FindOne(ctx, filter, options.FindOne().SetSort(bson.D{{Key: "created_time", Value: -1}})).
		Decode(&notification)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find latest notification: %w", err)
	}
	return &notification, nil
}
