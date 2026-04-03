package user

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
)

func (s *Service) CreateUser(ctx context.Context, user *User) error {
	if user == nil {
		return bizerr.Param(errMsgInvalidParam)
	}
	if user.OpenID != "" {
		count, err := s.repo.CountUsersByOpenID(ctx, user.OpenID)
		if err != nil {
			return err
		}
		if count > 0 {
			return bizerr.Biz("openId:" + user.OpenID + "已存在")
		}
	}
	return s.repo.CreateUser(ctx, user)
}

func (s *Service) AddAdmin(ctx context.Context, userID int64, username, password string, power *int) error {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return bizerr.Biz("关联用户不存在")
	}

	count, err := s.repo.CountAdminsByUsername(ctx, username)
	if err != nil {
		return err
	}
	if count > 0 {
		return bizerr.Biz("用户名重复")
	}

	count, err = s.repo.CountAdminsByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if count > 0 {
		return bizerr.Biz("管理员已存在")
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
	if err := s.repo.CreateAdmin(ctx, &admin); err != nil {
		return bizerr.Biz("添加失败，请重试")
	}
	return nil
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	return s.repo.DeleteUser(ctx, id)
}

func (s *Service) EditAdminUser(ctx context.Context, userID, operatorID int64, req AdminEditUserReq) error {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return bizerr.Biz("不存在")
	}

	hasChange := req.Nickname != "" || req.Avatar != "" || req.Power != nil ||
		req.StuNum != "" || req.StuName != "" || req.StuCla != "" || req.StuIsCheck != nil
	if !hasChange {
		return bizerr.Biz("更新失败")
	}
	if err := s.repo.UpdateAdminUser(ctx, userID, operatorID, req); err != nil {
		return err
	}

	if s.producer != nil {
		msg := TopicUserUpdateMsg{
			UserID:      fmt.Sprintf("%d", userID),
			NickName:    req.Nickname,
			Avatar:      req.Avatar,
			AccountType: mapAccountType(user.AccountType),
		}
		if err := s.producer.SendTopicUserUpdate(ctx, msg); err != nil {
			s.logger.Warn("send admin topic user update mq failed", zap.Error(err), zap.Int64("userID", userID))
		}
		if err := s.producer.SendCommentUserUpdate(ctx, CommentUserUpdateMsg(msg)); err != nil {
			s.logger.Warn("send admin comment user update mq failed", zap.Error(err), zap.Int64("userID", userID))
		}
	}
	return nil
}

func (s *Service) ListUsers(ctx context.Context, page, size int, name string) (*PageResult[User], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = s.defaultPageSize()
	}

	list, total, err := s.repo.ListUsers(ctx, page, size, name)
	if err != nil {
		return nil, err
	}
	return NewPageResult(s.sanitizeUsers(list), total, page, size), nil
}

func (s *Service) ClearAuthentication(ctx context.Context, userID int64) error {
	return s.repo.ClearAuthentication(ctx, userID)
}

func (s *Service) GetCourseFileByKey(ctx context.Context, key string) (*CourseFile, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	course, err := s.repo.FindCourseFileByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if course == nil {
		return nil, bizerr.NotFound("获取课表失败,请联系管理员")
	}
	return course, nil
}
