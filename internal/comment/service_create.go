package comment

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) AddComment(ctx context.Context, topicID string, currentUserID int64, content, parentCmtID string) (string, error) {
	topic, err := s.getTopic(ctx, topicID, false)
	if err != nil {
		return "", err
	}
	if topic == nil {
		return "", result.NewBizError(result.CodeFail, "帖子不存在")
	}

	user, err := s.loadUser(ctx, currentUserID)
	if err != nil {
		return "", err
	}
	if err := s.validateCommentPermission(ctx, user, topic); err != nil {
		return "", err
	}

	comment := Comment{
		TopicID:     topicID,
		Comment:     content,
		CreatedTime: time.Now(),
		User:        buildCommentUser(user),
		ParentCmtID: parentCmtID,
		RootCmtID:   DefaultRootCommentID,
		IsAuthor:    topic.UserID == strconv.FormatInt(currentUserID, 10),
		LikeNum:     0,
		CommentNum:  0,
		HasCheck:    false,
	}

	if parentCmtID != DefaultRootCommentID {
		parent, err := s.getCommentByID(ctx, parentCmtID)
		if err != nil {
			return "", err
		}
		parentUser := parent.User
		comment.Parent = &parentUser
		if parent.RootCmtID != "" && parent.RootCmtID != DefaultRootCommentID {
			comment.RootCmtID = parent.RootCmtID
		} else {
			comment.RootCmtID = parent.ID.Hex()
		}
	}

	res, err := s.commentColl().InsertOne(ctx, comment)
	if err != nil {
		return "", fmt.Errorf("insert comment: %w", err)
	}
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("comment inserted id type invalid")
	}

	comment.ID = oid
	if s.producer != nil {
		if err := s.producer.SendAddComment(ctx, comment); err != nil {
			s.logger.Warn("send add comment mq failed", zap.Error(err), zap.String("commentID", oid.Hex()))
		}
	}
	return oid.Hex(), nil
}

func (s *Service) DeleteComment(ctx context.Context, topicID, commentID string, userID int64, isAdmin bool) error {
	comment, err := s.getCommentByID(ctx, commentID)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": comment.ID, "topicId": topicID}
	if !isAdmin {
		filter["user.userId"] = strconv.FormatInt(userID, 10)
	}

	update := bson.M{"$set": bson.M{"hasCheck": false, "deletedTime": time.Now()}}
	res, err := s.commentColl().UpdateOne(ctx, filter, update)
	if err != nil {
		return s.deleteCommentFallback(ctx, topicID, commentID, comment.RootCmtID, fmt.Errorf("delete comment: %w", err))
	}
	if res.MatchedCount == 0 {
		return result.ErrNotExisted
	}
	if res.ModifiedCount == 0 {
		return result.NewBizError(result.CodeFail, "删除评论失败")
	}

	if err := s.decrementCommentCounters(ctx, topicID, comment.RootCmtID); err != nil {
		return s.deleteCommentFallback(ctx, topicID, commentID, comment.RootCmtID, err)
	}
	return nil
}

func (s *Service) decrementCommentCounters(ctx context.Context, topicID, rootCommentID string) error {
	topicOID, err := primitive.ObjectIDFromHex(topicID)
	if err != nil {
		return fmt.Errorf("invalid topic id: %w", err)
	}
	topicRes, err := s.mongoDB.Collection("campus_topic").UpdateByID(ctx, topicOID, bson.M{"$inc": bson.M{"commentNum": -1}})
	if err != nil {
		return fmt.Errorf("decrease topic comment num: %w", err)
	}
	if topicRes.MatchedCount == 0 {
		return result.NewBizError(result.CodeFail, "删除评论失败")
	}

	if rootCommentID == "" || rootCommentID == DefaultRootCommentID {
		return nil
	}
	rootOID, err := primitive.ObjectIDFromHex(rootCommentID)
	if err != nil {
		return fmt.Errorf("invalid root comment id: %w", err)
	}
	rootRes, err := s.commentColl().UpdateByID(ctx, rootOID, bson.M{"$inc": bson.M{"commentNum": -1}})
	if err != nil {
		return fmt.Errorf("decrease root comment num: %w", err)
	}
	if rootRes.MatchedCount == 0 {
		return result.NewBizError(result.CodeFail, "删除评论失败")
	}
	return nil
}

func (s *Service) deleteCommentFallback(ctx context.Context, topicID, commentID, _ string, cause error) error {
	if s.producer != nil {
		if err := s.producer.SendDeleteComment(ctx, topicID, commentID); err != nil {
			s.logger.Warn("send delete comment mq failed", zap.Error(err), zap.String("commentID", commentID))
		}
	}
	return cause
}
