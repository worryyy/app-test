package comment

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

func (s *Service) LikeComment(ctx context.Context, commentID string, userID int64) error {
	userIDStr := userIDString(userID)
	exists, err := s.repo.CommentLikeExists(ctx, commentID, userIDStr)
	if err != nil {
		return bizerr.InternalWrap("查询评论点赞状态失败", err)
	}
	if exists {
		return nil
	}

	if err := s.incCommentLikeStrict(ctx, commentID, 1); err != nil {
		return err
	}

	updated, err := s.repo.AddCommentLike(ctx, commentID, userIDStr)
	if err != nil {
		return bizerr.InternalWrap("点赞评论失败", err)
	}
	if !updated {
		return ErrCommentLikeFailed
	}
	return nil
}

func (s *Service) UnlikeComment(ctx context.Context, commentID string, userID int64) error {
	userIDStr := userIDString(userID)
	exists, err := s.repo.CommentLikeExists(ctx, commentID, userIDStr)
	if err != nil {
		return bizerr.InternalWrap("查询评论点赞状态失败", err)
	}
	if !exists {
		return ErrCommentLikeNotFound
	}

	if err := s.incCommentLikeStrict(ctx, commentID, -1); err != nil {
		return err
	}

	updated, err := s.repo.RemoveCommentLike(ctx, commentID, userIDStr)
	if err != nil {
		return bizerr.InternalWrap("取消点赞失败", err)
	}
	if !updated {
		return ErrCommentUnlikeFailed
	}
	return nil
}

func (s *Service) incCommentLikeStrict(ctx context.Context, commentID string, delta int64) error {
	oid, err := parseCommentObjectID(commentID)
	if err != nil {
		return err
	}

	ok, err := s.repo.IncrementCommentLikeNum(ctx, oid, delta)
	if err != nil {
		return bizerr.InternalWrap("更新评论点赞数失败", err)
	}
	if !ok {
		return ErrCommentLikeFailed
	}
	return nil
}
