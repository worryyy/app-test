package user

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)


func (s *Service) CreateUser(ctx context.Context, u *User) error {
	if u == nil {
		return result.ErrParam
	}
	if u.OpenID != "" {
		var count int64
		if err := s.db.WithContext(ctx).Model(&User{}).Where("open_id = ?", u.OpenID).Count(&count).Error; err != nil {
			return fmt.Errorf("count user by open_id: %w", err)
		}
		if count > 0 {
			return result.NewBizError(result.CodeFail, "openId:"+u.OpenID+"已存在")
		}
	}
	if err := s.db.WithContext(ctx).Create(u).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (s *Service) AddAdmin(ctx context.Context, userID int64, username, password string, power *int) error {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return result.NewBizError(result.CodeFail, "关联用户不存在")
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&Admin{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return fmt.Errorf("count admin by username: %w", err)
	}
	if count > 0 {
		return result.NewBizError(result.CodeFail, "用户名重复")
	}
	if err := s.db.WithContext(ctx).Model(&Admin{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return fmt.Errorf("count admin by user id: %w", err)
	}
	if count > 0 {
		return result.NewBizError(result.CodeFail, "管理员已存在")
	}

	adminPower := 0
	if power != nil {
		adminPower = *power
	}
	admin := Admin{
		UserID:   userID,
		Username: username,
		Password: md5Hex(password),
		Power:    resolveAdminPower(adminPower),
	}
	if err := s.db.WithContext(ctx).Create(&admin).Error; err != nil {
		return result.NewBizError(result.CodeFail, "添加失败，请重试")
	}
	return nil
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	if err := s.db.WithContext(ctx).Delete(&User{}, id).Error; err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (s *Service) EditAdminUser(ctx context.Context, userID, operatorID int64, req AdminEditUserReq) error {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return result.NewBizError(result.CodeFail, "不存在")
	}

	updates := map[string]interface{}{}
	if req.StuNum != "" {
		updates["stu_num"] = req.StuNum
	}
	if req.StuCla != "" {
		updates["stu_cla"] = req.StuCla
	}
	if req.StuName != "" {
		updates["stu_name"] = req.StuName
	}
	if req.StuIsCheck != nil {
		updates["stu_is_check"] = *req.StuIsCheck
	}
	if req.Power != nil {
		updates["power"] = *req.Power
	}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if operatorID > 0 {
		updates["updated_by"] = operatorID
	}
	if len(updates) == 0 {
		return result.NewBizError(result.CodeFail, "更新失败")
	}

	if err := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return fmt.Errorf("edit admin user: %w", err)
	}

	if s.producer != nil {
		msg := mq.TopicUserUpdateMsg{
			UserID:      strconv.FormatInt(userID, 10),
			NickName:    req.Nickname,
			Avatar:      req.Avatar,
			AccountType: mapAccountType(user.AccountType),
		}
		if err := s.producer.SendUpdateTopicUser(ctx, msg); err != nil {
			s.logger.Warn("send admin topic user update mq failed", zap.Error(err), zap.Int64("userID", userID))
		}
		if err := s.producer.SendUpdateCommentUser(ctx, mq.CommentUserUpdateMsg(msg)); err != nil {
			s.logger.Warn("send admin comment user update mq failed", zap.Error(err), zap.Int64("userID", userID))
		}
	}
	return nil
}

func (s *Service) ListUsers(ctx context.Context, page, size int, name string) (*result.PageResult[User], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = s.defaultPageSize()
	}
	q := s.db.WithContext(ctx).Model(&User{})
	if name != "" {
		q = q.Where("nickname = ?", name)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	var list []User
	if err := q.Offset((page - 1) * size).Limit(size).Order("id DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return result.NewPage(s.sanitizeUsers(list), total, page, size), nil
}

func (s *Service) ClearAuthentication(ctx context.Context, userID int64) error {
	if err := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"stu_is_check": false,
		"stu_name":     "",
		"stu_cla":      "",
		"stu_num":      "",
	}).Error; err != nil {
		return fmt.Errorf("clear authentication: %w", err)
	}
	return nil
}


func (s *Service) GetCourseFileByKey(ctx context.Context, key string) (*CourseFile, error) {
	if strings.TrimSpace(key) == "" {
		return nil, result.ErrParam
	}

	var course CourseFile
	if err := s.mongoDB.Collection("campus_course").FindOne(ctx, bson.M{"key": key}).Decode(&course); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, result.NewBizStatusError(404, result.CodeFail, "获取课表失败,请联系管理员")
		}
		return nil, fmt.Errorf("find course by key: %w", err)
	}
	return &course, nil
}

type CourseFile struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Key      string             `bson:"key"`
	FilePath string             `bson:"filePath"`
	Val      []byte             `bson:"val"`
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
