package comment

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) ListByTopic(ctx context.Context, topicID, rootID string, viewerUserID int64, page, size int) (*result.CusPage[Comment], error) {
	page, size = s.normalizePage(page, size)
	if err := s.ensureTopicExists(ctx, topicID); err != nil {
		return nil, err
	}
	if rootID == "" {
		rootID = DefaultRootCommentID
	}
	if rootID != DefaultRootCommentID {
		if _, err := primitive.ObjectIDFromHex(rootID); err != nil {
			return nil, result.NewBizError(result.CodeFail, "请传入有效的root_id")
		}
	}

	filter := bson.M{
		"topicId":   topicID,
		"rootCmtId": rootID,
		"hasCheck":  bson.M{"$ne": false},
	}
	total, err := s.commentColl().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count comments: %w", err)
	}

	sort := bson.D{{Key: "_id", Value: 1}}
	if rootID == DefaultRootCommentID {
		sort = bson.D{{Key: "commentNum", Value: -1}, {Key: "_id", Value: -1}}
	}
	cur, err := s.commentColl().Find(ctx, filter, options.Find().
		SetSort(sort).
		SetSkip(int64((page-1)*size)).
		SetLimit(int64(size)))
	if err != nil {
		return nil, fmt.Errorf("find comments: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close comment cursor failed", zap.Error(closeErr))
		}
	}()

	var comments []Comment
	if err := cur.All(ctx, &comments); err != nil {
		return nil, fmt.Errorf("decode comments: %w", err)
	}

	liked, err := s.getHasLikeBatch(ctx, userIDString(viewerUserID), comments)
	if err != nil {
		return nil, err
	}
	for i := range comments {
		_, ok := liked[comments[i].ID.Hex()]
		comments[i].HasLike = ok
	}
	return result.NewCusPage(comments, total, page, size), nil
}

func (s *Service) ListMine(ctx context.Context, userID int64, page, size int) (*result.CusPage[MyCommentItem], error) {
	page, size = s.normalizePage(page, size)
	filter := bson.M{"user.userId": userIDString(userID), "hasCheck": bson.M{"$ne": false}}
	total, err := s.commentColl().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count my comments: %w", err)
	}

	cur, err := s.commentColl().Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "commentNum", Value: -1}, {Key: "likeNum", Value: -1}, {Key: "_id", Value: -1}}).
		SetSkip(int64((page-1)*size)).
		SetLimit(int64(size)))
	if err != nil {
		return nil, fmt.Errorf("find my comments: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close my comment cursor failed", zap.Error(closeErr))
		}
	}()

	var comments []Comment
	if err := cur.All(ctx, &comments); err != nil {
		return nil, fmt.Errorf("decode my comments: %w", err)
	}

	liked, err := s.getHasLikeBatch(ctx, userIDString(userID), comments)
	if err != nil {
		return nil, err
	}

	items := make([]MyCommentItem, 0, len(comments))
	for _, comment := range comments {
		_, ok := liked[comment.ID.Hex()]
		comment.HasLike = ok

		topic, err := s.getTopic(ctx, comment.TopicID, true)
		if err != nil {
			return nil, err
		}
		if topic == nil {
			return nil, result.NewBizError(result.CodeFail, fmt.Sprintf("%s 帖子不存在", comment.TopicID))
		}

		items = append(items, MyCommentItem{
			Comment: comment,
			Topic:   *topic,
		})
	}
	return result.NewCusPage(items, total, page, size), nil
}

func (s *Service) ListTargetUserComments(ctx context.Context, targetUserID string, page, size int) (interface{}, error) {
	page, size = s.normalizePage(page, size)
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" {
		return nil, result.NewBizError(result.CodeFail, "目标用户id不能为空")
	}

	filter := bson.M{"user.userId": targetUserID, "hasCheck": bson.M{"$ne": false}}
	cur, err := s.commentColl().Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "_id", Value: -1}}).
		SetSkip(int64((page-1)*size)).
		SetLimit(int64(size)))
	if err != nil {
		return nil, fmt.Errorf("find target user comments: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close target comment cursor failed", zap.Error(closeErr))
		}
	}()

	var comments []Comment
	if err := cur.All(ctx, &comments); err != nil {
		return nil, fmt.Errorf("decode target user comments: %w", err)
	}
	if len(comments) == 0 {
		return []Comment{}, nil
	}
	return result.NewCusPage(comments, int64(len(comments)), page, size), nil
}

func userIDString(userID int64) string {
	if userID <= 0 {
		return ""
	}
	return strconv.FormatInt(userID, 10)
}
