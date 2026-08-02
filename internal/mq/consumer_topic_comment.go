package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const rootCommentSentinel = "0"

type topicDoc struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	ThemeID    string             `bson:"themeId"`
	UserID     string             `bson:"userId"`
	Title      string             `bson:"title"`
	Content    string             `bson:"content"`
	Imgs       []string           `bson:"imgs"`
	HasCheck   bool               `bson:"hasCheck"`
	CommentNum int64              `bson:"commentNum"`
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
	openID, err := c.resolveWXOpenID(ctx, topic.UserID)
	if err != nil {
		return fmt.Errorf("resolve topic wx openid: %w", err)
	}

	filteredTitle, err := c.filterSensitiveText(ctx, topic.Title)
	if err != nil {
		return fmt.Errorf("filter topic title: %w", err)
	}
	filteredContent, err := c.filterSensitiveText(ctx, topic.Content)
	if err != nil {
		return fmt.Errorf("filter topic content: %w", err)
	}

	titleResult, err := c.wxClient.MsgSecCheck(ctx, filteredTitle, openID)
	if err != nil {
		return fmt.Errorf("wx check topic title: %w", err)
	}
	contentResult, err := c.wxClient.MsgSecCheck(ctx, filteredContent, openID)
	if err != nil {
		return fmt.Errorf("wx check topic content: %w", err)
	}

	if isRisky(titleResult.Suggest) || isRisky(contentResult.Suggest) {
		return nil
	}

	checkedTitle := pickFiltered(filteredTitle, titleResult.FilteredContent)
	checkedContent := pickFiltered(filteredContent, contentResult.FilteredContent)
	filteredImgs, err := c.filterImagesWithQRCode(ctx, topic.Imgs)
	if err != nil {
		c.logger.Warn("filter topic qrcode images failed", zap.Error(err), zap.String("topicID", msg.TopicID))
		filteredImgs = topic.Imgs
	}

	if _, err := c.mongoDB.Collection("campus_topic").UpdateByID(ctx, topicOID, bson.M{
		"$set": bson.M{
			"hasCheck": true,
			"title":    checkedTitle,
			"content":  checkedContent,
			"imgs":     filteredImgs,
		},
	}); err != nil {
		return fmt.Errorf("update checked topic: %w", err)
	}

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
	openID, err := c.resolveWXOpenID(ctx, cmt.User.UserID)
	if err != nil {
		return fmt.Errorf("resolve comment wx openid: %w", err)
	}
	filteredComment, err := c.filterSensitiveText(ctx, cmt.Comment)
	if err != nil {
		return fmt.Errorf("filter comment: %w", err)
	}
	checkResult, err := c.wxClient.MsgSecCheck(ctx, filteredComment, openID)
	if err != nil {
		return fmt.Errorf("wx check comment: %w", err)
	}
	if isRisky(checkResult.Suggest) {
		if err := c.deleteRejectedComment(ctx, cmt.ID); err != nil {
			return err
		}
		return nil
	}

	now := time.Now()
	filteredComment = pickFiltered(filteredComment, checkResult.FilteredContent)
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
		c.notifyCommentUsers(ctx, cmt, filteredComment, topic.UserID, now)
	}

	return nil
}

func (c *Consumers) deleteRejectedComment(ctx context.Context, commentID primitive.ObjectID) error {
	if commentID.IsZero() {
		return nil
	}
	if _, err := c.mongoDB.Collection("campus_comment").DeleteOne(ctx, bson.M{"_id": commentID}); err != nil {
		return fmt.Errorf("delete rejected comment: %w", err)
	}
	return nil
}

func (c *Consumers) notifyCommentUsers(ctx context.Context, cmt commentDoc, filteredComment, topicAuthorID string, createdTime time.Time) {
	if cmt.ParentCmtID == rootCommentSentinel {
		if topicAuthorID != "" && topicAuthorID != cmt.User.UserID {
			c.sendNotify(ctx, NotifyMsg{
				TargetUserID: topicAuthorID,
				SenderUserID: cmt.User.UserID,
				Type:         "COMMENT_ADD",
				Content:      filteredComment,
				TopicID:      cmt.TopicID,
				CommentID:    cmt.ID.Hex(),
				CreatedTime:  createdTime,
			})
		}
		return
	}

	parentUserID := ""
	if cmt.Parent != nil {
		parentUserID = cmt.Parent.UserID
	}
	if parentUserID == "" && cmt.ParentCmtID != "" && cmt.ParentCmtID != rootCommentSentinel {
		parentOID, convErr := primitive.ObjectIDFromHex(cmt.ParentCmtID)
		if convErr == nil {
			var parent commentDoc
			if err := c.mongoDB.Collection("campus_comment").FindOne(ctx, bson.M{"_id": parentOID}).Decode(&parent); err == nil {
				parentUserID = parent.User.UserID
			}
		}
	}
	if parentUserID == cmt.User.UserID {
		return
	}

	receivers := make(map[string]struct{}, 2)
	if topicAuthorID != "" {
		receivers[topicAuthorID] = struct{}{}
	}
	if parentUserID != "" {
		receivers[parentUserID] = struct{}{}
	}
	delete(receivers, cmt.User.UserID)

	for receiverID := range receivers {
		notifyType := "COMMENT_ADD"
		if receiverID == parentUserID {
			notifyType = "COMMENT_REPLY"
		}
		c.sendNotify(ctx, NotifyMsg{
			TargetUserID: receiverID,
			SenderUserID: cmt.User.UserID,
			Type:         notifyType,
			Content:      filteredComment,
			TopicID:      cmt.TopicID,
			CommentID:    cmt.ID.Hex(),
			CreatedTime:  createdTime,
		})
	}
}
