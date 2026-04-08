package user

import (
	"context"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

var (
	ErrFollowRepeated = bizerr.Biz("不可重复关注")
	ErrFollowSelf     = bizerr.Biz("不可关注自己")
	ErrFollowNotFound = bizerr.Biz("未关注该用户")
)

func (s *Service) Follow(ctx context.Context, followerID, followingID int64) error {
	if followerID <= 0 || followingID <= 0 {
		return bizerr.Param(errMsgInvalidParam)
	}
	if followerID == followingID {
		return ErrFollowSelf
	}

	target, err := s.GetByID(ctx, followingID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrUserNotFound
	}

	following, err := s.repo.IsFollowing(ctx, followerID, followingID)
	if err != nil {
		return err
	}
	if following {
		return ErrFollowRepeated
	}

	return s.repo.CreateFollow(ctx, followerID, followingID, time.Now())
}

func (s *Service) Unfollow(ctx context.Context, followerID, followingID int64) error {
	if followerID <= 0 || followingID <= 0 {
		return bizerr.Param(errMsgInvalidParam)
	}
	if followerID == followingID {
		return ErrFollowSelf
	}

	following, err := s.repo.IsFollowing(ctx, followerID, followingID)
	if err != nil {
		return err
	}
	if !following {
		return ErrFollowNotFound
	}

	removed, err := s.repo.DeleteFollow(ctx, followerID, followingID)
	if err != nil {
		return err
	}
	if !removed {
		return ErrFollowNotFound
	}
	return nil
}

func (s *Service) GetUserStats(ctx context.Context, userID int64) (*UserStats, error) {
	if userID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	target, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, ErrUserNotFound
	}

	followerCount, err := s.repo.CountFollowers(ctx, userID)
	if err != nil {
		return nil, err
	}
	followingCount, err := s.repo.CountFollowings(ctx, userID)
	if err != nil {
		return nil, err
	}
	likeCount, err := s.repo.SumTopicLikesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserStats{
		FollowerCount:  followerCount,
		FollowingCount: followingCount,
		LikeCount:      likeCount,
	}, nil
}
