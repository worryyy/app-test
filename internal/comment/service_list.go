package comment

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

func (s *Service) ListByTopic(
	ctx context.Context,
	topicID, rootID string,
	viewerUserID int64,
	page, size int,
) (*PageResult[Comment], error) {
	page, size = s.normalizePage(page, size)
	if err := s.ensureTopicExists(ctx, topicID); err != nil {
		return nil, err
	}
	if rootID == "" {
		rootID = DefaultRootCommentID
	}
	if rootID != DefaultRootCommentID {
		if _, err := parseCommentObjectID(rootID); err != nil {
			return nil, ErrInvalidRootID
		}
	}

	filter := bson.M{
		"topicId":   topicID,
		"rootCmtId": rootID,
		"hasCheck":  bson.M{"$ne": false},
	}
	sort := bson.D{{Key: "_id", Value: 1}}
	if rootID == DefaultRootCommentID {
		sort = bson.D{{Key: "commentNum", Value: -1}, {Key: "_id", Value: -1}}
	}

	comments, total, err := s.repo.FindCommentsPage(ctx, filter, sort, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询评论列表失败", err)
	}

	liked, err := s.getHasLikeBatch(ctx, userIDString(viewerUserID), comments)
	if err != nil {
		return nil, err
	}
	for i := range comments {
		_, ok := liked[comments[i].ID.Hex()]
		comments[i].HasLike = ok
	}
	return NewPageResult(comments, total, page, size), nil
}

func (s *Service) ListMine(ctx context.Context, userID int64, page, size int) (*PageResult[MyCommentItem], error) {
	page, size = s.normalizePage(page, size)
	filter := bson.M{"user.userId": userIDString(userID), "hasCheck": bson.M{"$ne": false}}

	comments, total, err := s.repo.FindCommentsPage(
		ctx,
		filter,
		bson.D{{Key: "commentNum", Value: -1}, {Key: "likeNum", Value: -1}, {Key: "_id", Value: -1}},
		page,
		size,
	)
	if err != nil {
		return nil, bizerr.InternalWrap("查询我的评论失败", err)
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
			return nil, ErrTopicNotFound
		}

		items = append(items, MyCommentItem{
			Comment: comment,
			Topic:   *topic,
		})
	}
	return NewPageResult(items, total, page, size), nil
}

func (s *Service) ListTargetUserComments(ctx context.Context, targetUserID int64, page, size int) (*PageResult[Comment], error) {
	page, size = s.normalizePage(page, size)
	if targetUserID <= 0 {
		return nil, ErrTargetUserIDRequired
	}

	comments, _, err := s.repo.FindCommentsPage(
		ctx,
		bson.M{"user.userId": userIDString(targetUserID), "hasCheck": bson.M{"$ne": false}},
		bson.D{{Key: "_id", Value: -1}},
		page,
		size,
	)
	if err != nil {
		return nil, bizerr.InternalWrap("查询目标用户评论失败", err)
	}
	if len(comments) == 0 {
		return NewPageResult([]Comment{}, 0, page, size), nil
	}
	return NewPageResult(comments, int64(len(comments)), page, size), nil
}
