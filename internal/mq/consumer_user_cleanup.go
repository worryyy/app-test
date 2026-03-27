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

func (c *Consumers) handleTopicUpdate(ctx context.Context, data json.RawMessage) error {
	var msg TopicUserUpdateMsg
	if err := decodeData(data, &msg); err != nil {
		return err
	}
	if msg.UserID == "" {
		return nil
	}

	update := bson.M{}
	if msg.NickName != "" {
		update["nickName"] = msg.NickName
	}
	if msg.Avatar != "" {
		update["avatar"] = msg.Avatar
	}
	if msg.Gender != "" {
		update["gender"] = msg.Gender
	}
	if msg.Signature != "" {
		update["signature"] = msg.Signature
	}
	if msg.AccountType > 0 {
		update["accountType"] = accountTypeName(msg.AccountType)
	}
	if len(update) == 0 {
		return nil
	}

	if _, err := c.mongoDB.Collection("campus_topic").UpdateMany(
		ctx,
		bson.M{"userId": msg.UserID},
		bson.M{"$set": update},
	); err != nil {
		return fmt.Errorf("update topic user profile: %w", err)
	}
	return nil
}

func (c *Consumers) handleCommentUpdate(ctx context.Context, data json.RawMessage) error {
	var msg CommentUserUpdateMsg
	if err := decodeData(data, &msg); err != nil {
		return err
	}
	if msg.UserID == "" {
		return nil
	}

	userSet := bson.M{}
	if msg.NickName != "" {
		userSet["user.nickName"] = msg.NickName
	}
	if msg.Avatar != "" {
		userSet["user.avatar"] = msg.Avatar
	}
	if msg.Gender != "" {
		userSet["user.gender"] = msg.Gender
	}
	if msg.Signature != "" {
		userSet["user.signature"] = msg.Signature
	}
	if msg.AccountType > 0 {
		userSet["user.accountType"] = accountTypeName(msg.AccountType)
	}
	if len(userSet) == 0 {
		return nil
	}

	commentColl := c.mongoDB.Collection("campus_comment")
	if _, err := commentColl.UpdateMany(
		ctx,
		bson.M{"user.userId": msg.UserID},
		bson.M{"$set": userSet},
	); err != nil {
		return fmt.Errorf("update comment user profile: %w", err)
	}

	parentSet := bson.M{}
	if msg.NickName != "" {
		parentSet["parent.nickName"] = msg.NickName
	}
	if msg.Avatar != "" {
		parentSet["parent.avatar"] = msg.Avatar
	}
	if msg.Gender != "" {
		parentSet["parent.gender"] = msg.Gender
	}
	if msg.Signature != "" {
		parentSet["parent.signature"] = msg.Signature
	}
	if len(parentSet) == 0 {
		return nil
	}
	if _, err := commentColl.UpdateMany(
		ctx,
		bson.M{"parent.userId": msg.UserID},
		bson.M{"$set": parentSet},
	); err != nil {
		return fmt.Errorf("update comment parent profile: %w", err)
	}
	return nil
}

func (c *Consumers) handleTopicDelete(ctx context.Context, data json.RawMessage) error {
	var msg TopicDeleteMsg
	if err := decodeData(data, &msg); err != nil {
		return err
	}
	if msg.TopicID == "" {
		return nil
	}

	_, _ = c.mongoDB.Collection("campus_topic_search").DeleteOne(ctx, bson.M{"topicId": msg.TopicID})
	_, _ = c.mongoDB.Collection("campus_topic_like").UpdateMany(ctx, bson.M{}, bson.M{"$pull": bson.M{"topicIds": msg.TopicID}})
	_, _ = c.mongoDB.Collection("campus_topic_collection").UpdateMany(ctx, bson.M{}, bson.M{"$pull": bson.M{"topicIds": msg.TopicID}})
	if _, err := c.mongoDB.Collection("campus_comment").UpdateMany(
		ctx,
		bson.M{"topicId": msg.TopicID},
		bson.M{"$set": bson.M{"hasCheck": false}},
	); err != nil {
		return fmt.Errorf("soft delete topic comments: %w", err)
	}
	return nil
}

func (c *Consumers) handleCommentDelete(ctx context.Context, data json.RawMessage) error {
	var msg CommentDeleteMsg
	if err := decodeData(data, &msg); err != nil {
		return err
	}
	if msg.CommentID == "" {
		return nil
	}

	commentOID, err := primitive.ObjectIDFromHex(msg.CommentID)
	if err != nil {
		return fmt.Errorf("invalid comment id: %w", err)
	}

	var cmt commentDoc
	err = c.mongoDB.Collection("campus_comment").FindOne(ctx, bson.M{"_id": commentOID}).Decode(&cmt)
	if err == nil {
		if cmt.HasCheck && cmt.TopicID != "" {
			if topicOID, convErr := primitive.ObjectIDFromHex(cmt.TopicID); convErr == nil {
				_, _ = c.mongoDB.Collection("campus_topic").UpdateByID(ctx, topicOID, bson.M{"$inc": bson.M{"commentNum": -1}})
			}
		}
		if cmt.HasCheck && cmt.RootCmtID != "" {
			if rootOID, convErr := primitive.ObjectIDFromHex(cmt.RootCmtID); convErr == nil {
				_, _ = c.mongoDB.Collection("campus_comment").UpdateByID(ctx, rootOID, bson.M{"$inc": bson.M{"commentNum": -1}})
			}
		}
	}

	_, _ = c.mongoDB.Collection("campus_comment_like").DeleteMany(ctx, bson.M{"commentId": msg.CommentID})

	if _, err := c.mongoDB.Collection("campus_comment").DeleteOne(ctx, bson.M{"_id": commentOID}); err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	return nil
}

func accountTypeName(accountType int) string {
	switch accountType {
	case 2:
		return "official"
	case 3:
		return "anonymous"
	default:
		return "base"
	}
}

func (c *Consumers) handleDie(ctx context.Context, data json.RawMessage) error {
	var msg DieMsg
	if err := decodeData(data, &msg); err != nil {
		msg = DieMsg{Payload: string(data)}
	}
	if c.mongoDB == nil {
		c.logger.Warn("die queue message received but mongo is nil", zap.String("queue", msg.Queue))
		return nil
	}

	_, err := c.mongoDB.Collection("campus_mq").InsertOne(ctx, MQLog{
		CreatedTime: time.Now(),
		Type:        "die",
		Data:        msg,
	})
	if err != nil {
		return fmt.Errorf("save die queue message: %w", err)
	}
	return nil
}
