package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

func (c *Consumers) handleNotifyUser(ctx context.Context, data json.RawMessage) error {
	var msg NotifyMsg
	if err := decodeData(data, &msg); err != nil {
		return err
	}
	if msg.TargetUserID == "" {
		return nil
	}

	createdTime := msg.CreatedTime
	if createdTime.IsZero() {
		createdTime = time.Now()
	}

	id := primitive.NewObjectID()
	notification := bson.M{
		"_id":          id,
		"receiver_id":  msg.TargetUserID,
		"sender_id":    msg.SenderUserID,
		"type":         msg.Type,
		"content":      msg.Content,
		"topic_id":     msg.TopicID,
		"comment_id":   msg.CommentID,
		"created_time": createdTime,
		"is_read":      false,
	}
	if _, err := c.mongoDB.Collection("campus_notifications").InsertOne(ctx, notification); err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}

	if c.notifyPusher != nil {
		if err := c.notifyPusher(ctx, msg.TargetUserID, notification); err != nil {
			c.logger.Warn("push realtime notification failed", zap.Error(err), zap.String("targetUserID", msg.TargetUserID))
		}
	}
	return nil
}
