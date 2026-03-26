package comment

import (
	"context"
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) LikeComment(ctx context.Context, commentID string, userID int64) error {
	userIDStr := strconv.FormatInt(userID, 10)
	exists, err := s.commentLikeExists(ctx, commentID, userIDStr)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if err := s.incCommentLikeStrict(ctx, commentID, 1); err != nil {
		return err
	}
	res, err := s.mongoDB.Collection("campus_comment_like").UpdateOne(
		ctx,
		bson.M{"commentId": commentID},
		bson.M{
			"$setOnInsert": bson.M{"commentId": commentID},
			"$addToSet":    bson.M{"userIds": userIDStr},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("like comment: %w", err)
	}
	if res.ModifiedCount == 0 && res.UpsertedCount == 0 {
		return result.NewBizError(result.CodeFail, "点赞失败")
	}
	return nil
}

func (s *Service) UnlikeComment(ctx context.Context, commentID string, userID int64) error {
	userIDStr := strconv.FormatInt(userID, 10)
	exists, err := s.commentLikeExists(ctx, commentID, userIDStr)
	if err != nil {
		return err
	}
	if !exists {
		return result.NewBizError(result.CodeFail, "还没有对该评论进行点赞")
	}

	if err := s.incCommentLikeStrict(ctx, commentID, -1); err != nil {
		return err
	}
	res, err := s.mongoDB.Collection("campus_comment_like").UpdateOne(
		ctx,
		bson.M{"commentId": commentID},
		bson.M{"$pull": bson.M{"userIds": userIDStr}},
	)
	if err != nil {
		return fmt.Errorf("unlike comment: %w", err)
	}
	if res.ModifiedCount == 0 {
		return result.NewBizError(result.CodeFail, "取消点赞失败")
	}
	return nil
}

func (s *Service) commentLikeExists(ctx context.Context, commentID, userID string) (bool, error) {
	count, err := s.mongoDB.Collection("campus_comment_like").CountDocuments(ctx, bson.M{
		"commentId": commentID,
		"userIds":   userID,
	})
	if err != nil {
		return false, fmt.Errorf("count comment like: %w", err)
	}
	return count > 0, nil
}

func (s *Service) incCommentLikeStrict(ctx context.Context, commentID string, delta int64) error {
	oid, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return result.NewBizError(result.CodeFail, "点赞失败")
	}
	res, err := s.commentColl().UpdateByID(ctx, oid, bson.M{"$inc": bson.M{"likeNum": delta}})
	if err != nil {
		return fmt.Errorf("update comment like num: %w", err)
	}
	if res.MatchedCount == 0 {
		return result.NewBizError(result.CodeFail, "点赞失败")
	}
	return nil
}
