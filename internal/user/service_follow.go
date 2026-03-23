package user

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) Follow(ctx context.Context, followerID, targetUserID int64) error {
	if followerID == targetUserID {
		return ErrFollowSelf
	}
	coll := s.mongoDB.Collection("campus_follow")
	doc := Follow{
		FollowerID:  strconv.FormatInt(followerID, 10),
		FollowingID: strconv.FormatInt(targetUserID, 10),
		FollowAt:    time.Now(),
	}
	_, err := coll.UpdateOne(
		ctx,
		bson.M{"followerId": doc.FollowerID, "followingId": doc.FollowingID},
		bson.M{"$setOnInsert": doc},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("follow user: %w", err)
	}
	return nil
}

func (s *Service) Unfollow(ctx context.Context, followerID, targetUserID int64) error {
	coll := s.mongoDB.Collection("campus_follow")
	_, err := coll.DeleteOne(ctx, bson.M{
		"followerId":  strconv.FormatInt(followerID, 10),
		"followingId": strconv.FormatInt(targetUserID, 10),
	})
	if err != nil {
		return fmt.Errorf("unfollow user: %w", err)
	}
	return nil
}

func (s *Service) IsFollowing(ctx context.Context, followerID, targetUserID int64) (bool, error) {
	coll := s.mongoDB.Collection("campus_follow")
	count, err := coll.CountDocuments(ctx, bson.M{
		"followerId":  strconv.FormatInt(followerID, 10),
		"followingId": strconv.FormatInt(targetUserID, 10),
	})
	if err != nil {
		return false, fmt.Errorf("check following: %w", err)
	}
	return count > 0, nil
}

func (s *Service) GetStats(ctx context.Context, currentUserID, targetUserID int64) (*UserProfile, error) {
	u, err := s.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}

	followerCount, err := s.mongoDB.Collection("campus_follow").
		CountDocuments(ctx, bson.M{"followingId": strconv.FormatInt(targetUserID, 10)})
	if err != nil {
		return nil, fmt.Errorf("count followers: %w", err)
	}

	followingCount, err := s.mongoDB.Collection("campus_follow").
		CountDocuments(ctx, bson.M{"followerId": strconv.FormatInt(targetUserID, 10)})
	if err != nil {
		return nil, fmt.Errorf("count followings: %w", err)
	}

	likeCount, err := s.mongoDB.Collection("campus_topic_like").
		CountDocuments(ctx, bson.M{"userId": strconv.FormatInt(targetUserID, 10)})
	if err != nil {
		return nil, fmt.Errorf("count likes: %w", err)
	}

	topicCount, err := s.mongoDB.Collection("campus_topic").
		CountDocuments(ctx, bson.M{"userId": strconv.FormatInt(targetUserID, 10), "hasCheck": true})
	if err != nil {
		return nil, fmt.Errorf("count topics: %w", err)
	}

	isFollowing, err := s.IsFollowing(ctx, currentUserID, targetUserID)
	if err != nil {
		return nil, err
	}

	profile := &UserProfile{
		User:           *u,
		FollowerCount:  followerCount,
		FollowingCount: followingCount,
		LikeCount:      likeCount,
		TopicCount:     topicCount,
		IsFollowing:    isFollowing,
	}
	profile.StuPwd = ""
	return profile, nil
}

func (s *Service) ListFollowers(ctx context.Context, targetUserID int64, page, size int) (*result.CusPage[User], error) {
	return s.listFollowUsers(ctx, bson.M{"followingId": strconv.FormatInt(targetUserID, 10)}, "followerId", page, size)
}

func (s *Service) ListFollowings(ctx context.Context, targetUserID int64, page, size int) (*result.CusPage[User], error) {
	return s.listFollowUsers(ctx, bson.M{"followerId": strconv.FormatInt(targetUserID, 10)}, "followingId", page, size)
}

func (s *Service) listFollowUsers(ctx context.Context, filter bson.M, field string, page, size int) (*result.CusPage[User], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}

	coll := s.mongoDB.Collection("campus_follow")
	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count follow users: %w", err)
	}

	opts := options.Find().SetSkip(int64((page - 1) * size)).SetLimit(int64(size)).SetSort(bson.M{"followAt": -1})
	cur, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find follow users: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close follow cursor failed", zap.Error(closeErr))
		}
	}()

	var docs []Follow
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode follow users: %w", err)
	}

	ids := make([]int64, 0, len(docs))
	for _, d := range docs {
		raw := d.FollowerID
		if field == "followingId" {
			raw = d.FollowingID
		}
		id, convErr := strconv.ParseInt(raw, 10, 64)
		if convErr == nil {
			ids = append(ids, id)
		}
	}
	users, err := s.loadUsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return result.NewCusPage(users, total, page, size), nil
}

func (s *Service) loadUsersByIDs(ctx context.Context, ids []int64) ([]User, error) {
	if len(ids) == 0 {
		return []User{}, nil
	}

	var users []User
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("load users by ids: %w", err)
	}

	byID := make(map[int64]User, len(users))
	for _, u := range users {
		u.StuPwd = ""
		byID[u.ID] = u
	}

	out := make([]User, 0, len(ids))
	for _, id := range ids {
		if u, ok := byID[id]; ok {
			out = append(out, u)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
