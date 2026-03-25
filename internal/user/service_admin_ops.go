package user

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

const blacklistDocID = "global_blacklist"

func (s *Service) CreateUser(ctx context.Context, u *User) error {
	if u == nil {
		return result.ErrParam
	}
	if err := s.db.WithContext(ctx).Create(u).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (s *Service) AddAdmin(ctx context.Context, userID int64, username, password string) error {
	admin := Admin{
		UserID:   userID,
		Username: username,
		Password: md5Hex(password),
		Power:    2,
	}
	if err := s.db.WithContext(ctx).Create(&admin).Error; err != nil {
		return fmt.Errorf("add admin: %w", err)
	}
	return nil
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	if err := s.db.WithContext(ctx).Delete(&User{}, id).Error; err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (s *Service) ListUsers(ctx context.Context, page, size int, name string) (*result.PageResult[User], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	q := s.db.WithContext(ctx).Model(&User{})
	if name != "" {
		q = q.Where("nickname LIKE ?", "%"+name+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	var list []User
	if err := q.Offset((page - 1) * size).Limit(size).Order("id DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	for i := range list {
		list[i].StuPwd = ""
	}
	return result.NewPage(list, total, page, size), nil
}

func (s *Service) ClearAuthentication(ctx context.Context, userID int64) error {
	return s.DelAuthentication(ctx, userID)
}

func (s *Service) AddBlackList(ctx context.Context, identifiers []string) error {
	rootIDs, err := s.resolveBlacklistRootIDs(ctx, identifiers)
	if err != nil {
		return err
	}
	if len(rootIDs) == 0 {
		return result.ErrParam
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"updated_time": now,
		},
		"$setOnInsert": bson.M{
			"_id":          blacklistDocID,
			"created_time": now,
		},
		"$addToSet": bson.M{
			"blocked_user_ids": bson.M{"$each": rootIDs},
		},
	}
	if _, err := s.blacklistColl().UpdateByID(ctx, blacklistDocID, update, options.Update().SetUpsert(true)); err != nil {
		return fmt.Errorf("upsert blacklist doc: %w", err)
	}
	if s.redis != nil {
		members := make([]interface{}, 0, len(rootIDs))
		for _, rootID := range rootIDs {
			members = append(members, rootID)
		}
		if err := s.redis.SAdd(ctx, rediskey.GlobalBlacklist, members...).Err(); err != nil {
			return fmt.Errorf("add blacklist users: %w", err)
		}
	}
	return nil
}

func (s *Service) DelBlackList(ctx context.Context, identifiers []string) error {
	rootIDs, err := s.resolveBlacklistRootIDs(ctx, identifiers)
	if err != nil {
		return err
	}
	if len(rootIDs) == 0 {
		return result.ErrParam
	}

	if _, err := s.blacklistColl().UpdateByID(
		ctx,
		blacklistDocID,
		bson.M{
			"$set": bson.M{"updated_time": time.Now()},
			"$pull": bson.M{
				"blocked_user_ids": bson.M{"$in": rootIDs},
			},
		},
	); err != nil && err != mongo.ErrNoDocuments {
		return fmt.Errorf("remove blacklist from mongo: %w", err)
	}
	if s.redis != nil {
		members := make([]interface{}, 0, len(rootIDs))
		for _, rootID := range rootIDs {
			members = append(members, rootID)
		}
		if err := s.redis.SRem(ctx, rediskey.GlobalBlacklist, members...).Err(); err != nil {
			return fmt.Errorf("remove blacklist users: %w", err)
		}
	}
	return nil
}

func (s *Service) ListBlackList(ctx context.Context) ([]User, error) {
	rootIDs, err := s.loadBlacklistRootIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(rootIDs) == 0 {
		return []User{}, nil
	}
	ids := make([]int64, 0, len(rootIDs))
	for _, m := range rootIDs {
		if v, convErr := strconv.ParseInt(m, 10, 64); convErr == nil {
			ids = append(ids, v)
		}
	}
	var users []User
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("query blacklist users: %w", err)
	}
	byID := make(map[int64]User, len(users))
	for _, u := range users {
		u.StuPwd = ""
		byID[u.ID] = u
	}
	ordered := make([]User, 0, len(ids))
	for _, id := range ids {
		if u, ok := byID[id]; ok {
			ordered = append(ordered, u)
		}
	}
	return ordered, nil
}

func (s *Service) ListOfficialCertifications(ctx context.Context, page, size int) (*result.CusPage[OfficialCertification], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	coll := s.mongoDB.Collection("campus_official_certification")
	total, err := coll.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("count certifications: %w", err)
	}
	cur, err := coll.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"createdAt": -1}).SetSkip(int64((page-1)*size)).SetLimit(int64(size)))
	if err != nil {
		return nil, fmt.Errorf("find certifications: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close certification cursor failed", zap.Error(closeErr))
		}
	}()

	var list []OfficialCertification
	if err := cur.All(ctx, &list); err != nil {
		return nil, fmt.Errorf("decode certifications: %w", err)
	}
	return result.NewCusPage(list, total, page, size), nil
}

func (s *Service) ReviewCertification(ctx context.Context, certID string, approved bool) error {
	oid, err := primitive.ObjectIDFromHex(certID)
	if err != nil {
		return fmt.Errorf("invalid cert id: %w", err)
	}
	status := 2
	if approved {
		status = 1
	}
	var cert OfficialCertification
	if err := s.mongoDB.Collection("campus_official_certification").FindOneAndUpdate(
		ctx,
		bson.M{"_id": oid},
		bson.M{"$set": bson.M{"status": status}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&cert); err != nil {
		return fmt.Errorf("review certification: %w", err)
	}
	if approved {
		userID, convErr := strconv.ParseInt(cert.UserID, 10, 64)
		if convErr == nil {
			if err := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
				"accountType": "official",
			}).Error; err != nil {
				return fmt.Errorf("update official account type: %w", err)
			}
		}
	}
	return nil
}

func (s *Service) RequestCourseByKey(ctx context.Context, key string) error {
	if strings.TrimSpace(key) == "" {
		return result.ErrParam
	}
	if s.producer == nil {
		return nil
	}

	prefix := rediskey.UserCoursePrefix
	if !strings.HasPrefix(key, prefix) {
		return result.ErrParam
	}
	raw := strings.TrimPrefix(key, prefix)
	parts := strings.Split(raw, ":")
	if len(parts) != 3 {
		return result.ErrParam
	}

	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return result.ErrParam
	}
	week, err := strconv.Atoi(parts[2])
	if err != nil {
		return result.ErrParam
	}
	u, err := s.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrUserNotFound
	}

	msg := mq.CourseMsg{
		UserID: userID,
		StuNum: u.StuNum,
		StuPwd: u.StuPwd,
		Term:   parts[1],
		Week:   week,
	}
	if err := s.producer.SendGetCourse(ctx, msg); err != nil {
		return fmt.Errorf("send get course mq: %w", err)
	}
	return nil
}

func (s *Service) blacklistColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_user_blacklist")
}

func (s *Service) loadBlacklistRootIDs(ctx context.Context) ([]string, error) {
	if s.redis != nil {
		members, err := s.redis.SMembers(ctx, rediskey.GlobalBlacklist).Result()
		if err == nil && len(members) > 0 {
			return members, nil
		}
		if err != nil && err != redis.Nil {
			return nil, fmt.Errorf("list blacklist members: %w", err)
		}
	}

	var doc UserBlacklist
	err := s.blacklistColl().FindOne(ctx, bson.M{"_id": blacklistDocID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []string{}, nil
		}
		return nil, fmt.Errorf("find blacklist doc: %w", err)
	}

	rootIDs, err := s.resolveBlacklistRootIDs(ctx, doc.BlockedUserIDs)
	if err != nil {
		return nil, err
	}
	if len(rootIDs) == 0 {
		return []string{}, nil
	}

	if s.redis != nil {
		members := make([]interface{}, 0, len(rootIDs))
		for _, rootID := range rootIDs {
			members = append(members, rootID)
		}
		if err := s.redis.SAdd(ctx, rediskey.GlobalBlacklist, members...).Err(); err != nil {
			return nil, fmt.Errorf("restore blacklist cache: %w", err)
		}
	}

	if !sameStringSlice(doc.BlockedUserIDs, rootIDs) {
		_, updateErr := s.blacklistColl().UpdateByID(ctx, blacklistDocID, bson.M{
			"$set": bson.M{
				"blocked_user_ids": rootIDs,
				"updated_time":     time.Now(),
			},
		})
		if updateErr != nil {
			return nil, fmt.Errorf("migrate blacklist doc: %w", updateErr)
		}
	}
	return rootIDs, nil
}

func (s *Service) resolveBlacklistRootIDs(ctx context.Context, identifiers []string) ([]string, error) {
	seen := make(map[string]struct{})
	rootIDs := make([]string, 0, len(identifiers))
	for _, raw := range identifiers {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		var target *User
		if id, convErr := strconv.ParseInt(raw, 10, 64); convErr == nil && id > 0 {
			target, _ = s.GetByID(ctx, id)
		}
		if target == nil {
			var err error
			target, err = s.GetByOpenID(ctx, raw)
			if err != nil {
				return nil, err
			}
		}
		if target == nil {
			continue
		}

		rootID := strconv.FormatInt(rootUserID(target), 10)
		if _, ok := seen[rootID]; ok {
			continue
		}
		seen[rootID] = struct{}{}
		rootIDs = append(rootIDs, rootID)
	}
	return rootIDs, nil
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
