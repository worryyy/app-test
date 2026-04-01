package user

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) Follow(ctx context.Context, followerID, targetUserID int64) error {
	targetUser, err := s.GetByID(ctx, targetUserID)
	if err != nil {
		return err
	}
	if targetUser == nil {
		return result.ErrNotExisted
	}
	if followerID == targetUserID {
		return result.NewBizError(result.CodeFollowSelf, "用户不可关注自己")
	}

	filter := bson.M{
		"followerId":  followerID,
		"followingId": targetUserID,
	}
	exists, err := s.mongoDB.Collection("campus_follow").CountDocuments(ctx, filter)
	if err != nil {
		return fmt.Errorf("check follow exists: %w", err)
	}
	if exists > 0 {
		return result.NewBizError(result.CodeFollowRepeat, "不可重复关注")
	}

	doc := Follow{
		FollowerID:  followerID,
		FollowingID: targetUserID,
		FollowAt:    time.Now(),
	}
	if _, err := s.mongoDB.Collection("campus_follow").InsertOne(ctx, doc); err != nil {
		return fmt.Errorf("follow user: %w", err)
	}
	return nil
}

func (s *Service) Unfollow(ctx context.Context, followerID, targetUserID int64) error {
	filter := bson.M{
		"followerId":  followerID,
		"followingId": targetUserID,
	}
	exists, err := s.mongoDB.Collection("campus_follow").CountDocuments(ctx, filter)
	if err != nil {
		return fmt.Errorf("check follow before unfollow: %w", err)
	}
	if exists == 0 {
		return result.NewBizError(result.CodeFollowNotFollow, "不可对未关注用户取关")
	}

	res, err := s.mongoDB.Collection("campus_follow").DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("unfollow user: %w", err)
	}
	if res.DeletedCount != 1 {
		return result.NewBizError(result.CodeFail, "取关失败")
	}
	return nil
}

func (s *Service) IsFollowing(ctx context.Context, followerID, targetUserID int64) (bool, error) {
	targetUser, err := s.GetByID(ctx, targetUserID)
	if err != nil {
		return false, err
	}
	if targetUser == nil {
		return false, result.NewBizError(result.CodeNotExisted, "目标用户不存在")
	}

	count, err := s.mongoDB.Collection("campus_follow").CountDocuments(ctx, bson.M{
		"followerId":  followerID,
		"followingId": targetUserID,
	})
	if err != nil {
		return false, fmt.Errorf("check following: %w", err)
	}
	return count > 0, nil
}

func (s *Service) GetUserProfile(ctx context.Context, targetUserID int64) (*UserProfile, error) {
	user, err := s.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, result.NewBizError(result.CodeNotExisted, "用户不存在")
	}
	return &UserProfile{
		Avatar:    user.Avatar,
		Nickname:  user.Nickname,
		Gender:    user.Gender,
		StuCla:    user.StuCla,
		Signature: user.Signature,
	}, nil
}

func (s *Service) GetStats(ctx context.Context, targetUserID int64) (*UserStats, error) {
	user, err := s.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, result.NewBizError(result.CodeNotExisted, "用户不存在")
	}

	followerCount, err := s.mongoDB.Collection("campus_follow").
		CountDocuments(ctx, bson.M{"followingId": targetUserID})
	if err != nil {
		return nil, fmt.Errorf("count followers: %w", err)
	}
	followingCount, err := s.mongoDB.Collection("campus_follow").
		CountDocuments(ctx, bson.M{"followerId": targetUserID})
	if err != nil {
		return nil, fmt.Errorf("count followings: %w", err)
	}

	cur, err := s.mongoDB.Collection("campus_topic").
		Find(ctx, bson.M{"userId": strconv.FormatInt(targetUserID, 10)})
	if err != nil {
		return nil, fmt.Errorf("find user topics for likes: %w", err)
	}
	defer func() { _ = cur.Close(ctx) }()

	var topics []userTopicLikeDoc
	if err := cur.All(ctx, &topics); err != nil {
		return nil, fmt.Errorf("decode user topics for likes: %w", err)
	}

	var likeCount int64
	for _, item := range topics {
		likeCount += item.LikeNum
	}

	return &UserStats{
		FollowerCount:  followerCount,
		FollowingCount: followingCount,
		LikeCount:      likeCount,
	}, nil
}

func (s *Service) ListFollowers(ctx context.Context, targetUserID int64, page, size int) (*result.CusPage[FollowItem], error) {
	targetUser, err := s.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if targetUser == nil {
		return nil, result.NewBizError(result.CodeNotExisted, "目标用户不存在")
	}
	return s.listFollowVO(ctx, bson.M{"followingId": targetUserID}, true, targetUserID, page, size)
}

func (s *Service) ListFollowings(ctx context.Context, targetUserID int64, page, size int) (*result.CusPage[FollowItem], error) {
	targetUser, err := s.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if targetUser == nil {
		return nil, result.NewBizError(result.CodeNotExisted, "目标用户不存在")
	}
	return s.listFollowVO(ctx, bson.M{"followerId": targetUserID}, false, targetUserID, page, size)
}

func (s *Service) listFollowVO(
	ctx context.Context,
	filter bson.M,
	isFollowers bool,
	targetUserID int64,
	page, size int,
) (*result.CusPage[FollowItem], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = s.defaultPageSize()
	}

	coll := s.mongoDB.Collection("campus_follow")
	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count follow users: %w", err)
	}

	opts := options.Find().
		SetSkip(int64((page - 1) * size)).
		SetLimit(int64(size)).
		SetSort(bson.M{"_id": -1})
	cur, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find follow users: %w", err)
	}
	defer func() { _ = cur.Close(ctx) }()

	var docs []Follow
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode follow users: %w", err)
	}
	if len(docs) == 0 {
		return result.NewCusPage([]FollowItem{}, total, page, size), nil
	}

	ids := make([]int64, 0, len(docs))
	for _, doc := range docs {
		if isFollowers {
			ids = append(ids, doc.FollowerID)
		} else {
			ids = append(ids, doc.FollowingID)
		}
	}

	usersByID, err := s.loadUsersMapByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	mutualSet, err := s.loadMutualSet(ctx, ids, targetUserID, isFollowers)
	if err != nil {
		return nil, err
	}

	items := make([]FollowItem, 0, len(docs))
	for _, doc := range docs {
		var counterpartID int64
		if isFollowers {
			counterpartID = doc.FollowerID
		} else {
			counterpartID = doc.FollowingID
		}
		user, ok := usersByID[counterpartID]
		if !ok {
			continue
		}
		item := FollowItem{
			Avatar:      user.Avatar,
			Nickname:    user.Nickname,
			FollowerID:  counterpartFollowerID(isFollowers, counterpartID, targetUserID),
			FollowingID: counterpartFollowingID(isFollowers, counterpartID, targetUserID),
			FollowAt:    doc.FollowAt,
			CoFollow:    false,
			BothFollow:  mutualSet[counterpartID],
		}
		items = append(items, item)
	}
	return result.NewCusPage(items, total, page, size), nil
}

func (s *Service) loadUsersMapByIDs(ctx context.Context, ids []int64) (map[int64]User, error) {
	if len(ids) == 0 {
		return map[int64]User{}, nil
	}
	var users []User
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("load users by ids: %w", err)
	}
	users = s.sanitizeUsers(users)
	out := make(map[int64]User, len(users))
	for _, user := range users {
		out[user.ID] = user
	}
	return out, nil
}

func (s *Service) loadMutualSet(ctx context.Context, ids []int64, targetUserID int64, isFollowers bool) (map[int64]bool, error) {
	if len(ids) == 0 {
		return map[int64]bool{}, nil
	}
	filter := bson.M{}
	if isFollowers {
		filter = bson.M{"followerId": targetUserID, "followingId": bson.M{"$in": ids}}
	} else {
		filter = bson.M{"followerId": bson.M{"$in": ids}, "followingId": targetUserID}
	}

	cur, err := s.mongoDB.Collection("campus_follow").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find mutual follows: %w", err)
	}
	defer func() { _ = cur.Close(ctx) }()

	var docs []Follow
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode mutual follows: %w", err)
	}

	out := make(map[int64]bool, len(docs))
	for _, doc := range docs {
		if !isFollowers {
			out[doc.FollowerID] = true
		} else {
			out[doc.FollowingID] = true
		}
	}
	return out, nil
}

func counterpartFollowerID(isFollowers bool, counterpartID, targetUserID int64) int64 {
	if isFollowers {
		return counterpartID
	}
	return targetUserID
}

func counterpartFollowingID(isFollowers bool, counterpartID, targetUserID int64) int64 {
	if isFollowers {
		return targetUserID
	}
	return counterpartID
}

func (s *Service) defaultPageSize() int {
	if s.cfg != nil && s.cfg.Custom.PageSize > 0 {
		return s.cfg.Custom.PageSize
	}
	return 15
}

type userTopicLikeDoc struct {
	LikeNum int64 `bson:"likeNum"`
}
