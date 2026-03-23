package user

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

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

func (s *Service) AddBlackList(ctx context.Context, userIDs []int64) error {
	if len(userIDs) == 0 {
		return nil
	}
	members := make([]interface{}, 0, len(userIDs))
	for _, id := range userIDs {
		rootID := id
		if u, err := s.GetByID(ctx, id); err == nil && u != nil && u.RootUserID != 0 {
			rootID = u.RootUserID
		}
		members = append(members, strconv.FormatInt(rootID, 10))
	}
	if err := s.redis.SAdd(ctx, rediskey.GlobalBlacklist, members...).Err(); err != nil {
		return fmt.Errorf("add blacklist users: %w", err)
	}
	return nil
}

func (s *Service) DelBlackList(ctx context.Context, userIDs []int64) error {
	if len(userIDs) == 0 {
		return nil
	}
	members := make([]interface{}, 0, len(userIDs))
	for _, id := range userIDs {
		rootID := id
		if u, err := s.GetByID(ctx, id); err == nil && u != nil && u.RootUserID != 0 {
			rootID = u.RootUserID
		}
		members = append(members, strconv.FormatInt(rootID, 10))
	}
	if err := s.redis.SRem(ctx, rediskey.GlobalBlacklist, members...).Err(); err != nil {
		return fmt.Errorf("remove blacklist users: %w", err)
	}
	return nil
}

func (s *Service) ListBlackList(ctx context.Context) ([]User, error) {
	members, err := s.redis.SMembers(ctx, rediskey.GlobalBlacklist).Result()
	if err != nil {
		return nil, fmt.Errorf("list blacklist members: %w", err)
	}
	if len(members) == 0 {
		return []User{}, nil
	}
	ids := make([]int64, 0, len(members))
	for _, m := range members {
		if v, convErr := strconv.ParseInt(m, 10, 64); convErr == nil {
			ids = append(ids, v)
		}
	}
	var users []User
	if err := s.db.WithContext(ctx).Where("id IN ? OR rootUserId IN ?", ids, ids).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("query blacklist users: %w", err)
	}
	for i := range users {
		users[i].StuPwd = ""
	}
	return users, nil
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
	defer cur.Close(ctx)

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
