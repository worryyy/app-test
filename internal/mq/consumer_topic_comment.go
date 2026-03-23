package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type topicDoc struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	ThemeID    string             `bson:"themeId"`
	UserID     string             `bson:"userId"`
	Title      string             `bson:"title"`
	Content    string             `bson:"content"`
	HasCheck   bool               `bson:"hasCheck"`
	CommentNum int64              `bson:"commentNum"`
}

type themeDoc struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Name       string             `bson:"name"`
	NeedSearch bool               `bson:"needSearch"`
}

type commentUserDoc struct {
	UserID string `json:"userId" bson:"userId"`
}

type commentDoc struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TopicID     string             `json:"topicId" bson:"topicId"`
	Comment     string             `json:"comment" bson:"comment"`
	CreatedTime time.Time          `json:"createdTime" bson:"createdTime"`
	User        commentUserDoc     `json:"user" bson:"user"`
	Parent      *commentUserDoc    `json:"parent" bson:"parent,omitempty"`
	ParentCmtID string             `json:"parentCmtId" bson:"parentCmtId,omitempty"`
	RootCmtID   string             `json:"rootCmtId" bson:"rootCmtId,omitempty"`
	HasCheck    bool               `json:"hasCheck" bson:"hasCheck"`
}

func (c *Consumers) handleTopicCheck(ctx context.Context, data json.RawMessage) error {
	var msg TopicCheckMsg
	if err := decodeData(data, &msg); err != nil {
		return err
	}

	topicOID, err := primitive.ObjectIDFromHex(msg.TopicID)
	if err != nil {
		return fmt.Errorf("invalid topic id: %w", err)
	}

	var topic topicDoc
	err = c.mongoDB.Collection("campus_topic").FindOne(ctx, bson.M{"_id": topicOID}).Decode(&topic)
	if err == mongo.ErrNoDocuments {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find topic for check: %w", err)
	}

	if c.wxClient == nil {
		return fmt.Errorf("wx client not initialized")
	}
	titleResult, err := c.wxClient.MsgSecCheck(ctx, topic.Title, topic.UserID)
	if err != nil {
		return fmt.Errorf("wx check topic title: %w", err)
	}
	contentResult, err := c.wxClient.MsgSecCheck(ctx, topic.Content, topic.UserID)
	if err != nil {
		return fmt.Errorf("wx check topic content: %w", err)
	}

	if isRisky(titleResult.Suggest) || isRisky(contentResult.Suggest) {
		_ = c.wxClient.SendSubscribeMsg(ctx, topic.UserID, "您的帖子未通过审核", topic.Title)
		return nil
	}

	filteredTitle := pickFiltered(topic.Title, titleResult.FilteredContent)
	filteredContent := pickFiltered(topic.Content, contentResult.FilteredContent)

	if _, err := c.mongoDB.Collection("campus_topic").UpdateByID(ctx, topicOID, bson.M{
		"$set": bson.M{
			"hasCheck": true,
			"title":    filteredTitle,
			"content":  filteredContent,
		},
	}); err != nil {
		return fmt.Errorf("update checked topic: %w", err)
	}

	if c.producer != nil {
		if err := c.producer.SendAddTopicSearch(ctx, AddTopicSearchMsg{TopicID: msg.TopicID}); err != nil {
			c.logger.Warn("send add topic search failed", zap.Error(err), zap.String("topicID", msg.TopicID))
		}
	}
	incPostPublish("success")
	_ = c.wxClient.SendSubscribeMsg(ctx, topic.UserID, "您的帖子已发布", filteredTitle)
	return nil
}

func (c *Consumers) handleCommentAdd(ctx context.Context, data json.RawMessage) error {
	var msg AddCommentMsg
	if err := decodeData(data, &msg); err != nil {
		return err
	}

	rawComment, err := json.Marshal(msg.Comment)
	if err != nil {
		return fmt.Errorf("marshal add comment payload: %w", err)
	}
	var cmt commentDoc
	if err := json.Unmarshal(rawComment, &cmt); err != nil {
		return fmt.Errorf("unmarshal add comment payload: %w", err)
	}
	if cmt.ID.IsZero() {
		return fmt.Errorf("comment id is empty")
	}

	if c.wxClient == nil {
		return fmt.Errorf("wx client not initialized")
	}
	checkResult, err := c.wxClient.MsgSecCheck(ctx, cmt.Comment, cmt.User.UserID)
	if err != nil {
		return fmt.Errorf("wx check comment: %w", err)
	}
	if isRisky(checkResult.Suggest) {
		_ = c.wxClient.SendSubscribeMsg(ctx, cmt.User.UserID, "您的评论未通过审核", cmt.Comment)
		return nil
	}

	now := time.Now()
	filteredComment := pickFiltered(cmt.Comment, checkResult.FilteredContent)
	update := bson.M{
		"$set": bson.M{
			"comment":     filteredComment,
			"hasCheck":    true,
			"createdTime": now,
		},
	}
	res, err := c.mongoDB.Collection("campus_comment").UpdateByID(ctx, cmt.ID, update)
	if err != nil {
		return fmt.Errorf("update comment checked status: %w", err)
	}
	if res.MatchedCount == 0 {
		cmt.Comment = filteredComment
		cmt.HasCheck = true
		cmt.CreatedTime = now
		if _, err := c.mongoDB.Collection("campus_comment").InsertOne(ctx, cmt); err != nil {
			return fmt.Errorf("insert checked comment: %w", err)
		}
	}

	topicOID, err := primitive.ObjectIDFromHex(cmt.TopicID)
	if err == nil {
		_, _ = c.mongoDB.Collection("campus_topic").UpdateByID(ctx, topicOID, bson.M{"$inc": bson.M{"commentNum": 1}})
	}
	if cmt.RootCmtID != "" {
		if rootOID, convErr := primitive.ObjectIDFromHex(cmt.RootCmtID); convErr == nil {
			_, _ = c.mongoDB.Collection("campus_comment").UpdateByID(ctx, rootOID, bson.M{"$inc": bson.M{"commentNum": 1}})
		}
	}

	var topic topicDoc
	if err := c.mongoDB.Collection("campus_topic").FindOne(ctx, bson.M{"_id": topicOID}).Decode(&topic); err == nil {
		if topic.UserID != "" && topic.UserID != cmt.User.UserID {
			c.sendNotify(ctx, NotifyMsg{
				TargetUserID: topic.UserID,
				Type:         "comment",
				Content: map[string]string{
					"topicId":  cmt.TopicID,
					"comment":  filteredComment,
					"commentId": cmt.ID.Hex(),
				},
			})
		}
	}

	parentUserID := ""
	if cmt.Parent != nil {
		parentUserID = cmt.Parent.UserID
	}
	if parentUserID == "" && cmt.ParentCmtID != "" {
		parentOID, convErr := primitive.ObjectIDFromHex(cmt.ParentCmtID)
		if convErr == nil {
			var parent commentDoc
			if err := c.mongoDB.Collection("campus_comment").FindOne(ctx, bson.M{"_id": parentOID}).Decode(&parent); err == nil {
				parentUserID = parent.User.UserID
			}
		}
	}
	if parentUserID != "" && parentUserID != cmt.User.UserID {
		c.sendNotify(ctx, NotifyMsg{
			TargetUserID: parentUserID,
			Type:         "comment",
			Content: map[string]string{
				"topicId":  cmt.TopicID,
				"comment":  filteredComment,
				"commentId": cmt.ID.Hex(),
			},
		})
	}

	incCommentPublish("success")
	return nil
}

func (c *Consumers) handleTopicSearchAdd(ctx context.Context, data json.RawMessage) error {
	return c.upsertTopicSearch(ctx, data)
}

func (c *Consumers) handleTopicSearchUpdate(ctx context.Context, data json.RawMessage) error {
	return c.upsertTopicSearch(ctx, data)
}

func (c *Consumers) handleTopicSearchDel(ctx context.Context, data json.RawMessage) error {
	var msg AddTopicSearchMsg
	if err := decodeData(data, &msg); err != nil {
		return err
	}
	if _, err := c.mongoDB.Collection("campus_topic_search").DeleteOne(ctx, bson.M{"topicId": msg.TopicID}); err != nil {
		return fmt.Errorf("delete topic search index: %w", err)
	}
	return nil
}

func (c *Consumers) upsertTopicSearch(ctx context.Context, data json.RawMessage) error {
	var msg AddTopicSearchMsg
	if err := decodeData(data, &msg); err != nil {
		return err
	}

	oid, err := primitive.ObjectIDFromHex(msg.TopicID)
	if err != nil {
		return fmt.Errorf("invalid topic id for search: %w", err)
	}
	var topic topicDoc
	if err := c.mongoDB.Collection("campus_topic").FindOne(ctx, bson.M{"_id": oid}).Decode(&topic); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil
		}
		return fmt.Errorf("find topic for search: %w", err)
	}

	themeOID, err := primitive.ObjectIDFromHex(topic.ThemeID)
	if err != nil {
		return fmt.Errorf("invalid theme id for search: %w", err)
	}
	var theme themeDoc
	if err := c.mongoDB.Collection("campus_theme").FindOne(ctx, bson.M{"_id": themeOID}).Decode(&theme); err != nil {
		return fmt.Errorf("find theme for search: %w", err)
	}
	if !theme.NeedSearch {
		return nil
	}

	update := bson.M{
		"$set": bson.M{
			"topicId":   msg.TopicID,
			"themeName": theme.Name,
			"title":     tokenizeText(topic.Title),
			"content":   tokenizeText(topic.Content),
		},
	}
	if _, err := c.mongoDB.Collection("campus_topic_search").UpdateOne(
		ctx,
		bson.M{"topicId": msg.TopicID},
		update,
		options.Update().SetUpsert(true),
	); err != nil {
		return fmt.Errorf("upsert topic search index: %w", err)
	}
	return nil
}
