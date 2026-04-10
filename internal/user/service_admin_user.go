package user

import (
	"context"
	"strings"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
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
			return bizerr.Biz("openId:" + user.OpenID + "\u5df2\u5b58\u5728")
		}
	}
	return s.repo.CreateUser(ctx, user)
}

func (s *Service) EditAdminUser(ctx context.Context, userID, operatorID int64, req AdminEditUserReq) error {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return bizerr.Biz("\u4e0d\u5b58\u5728")
	}

	hasChange := req.Nickname != "" || req.Avatar != "" || req.Power != nil ||
		req.StuNum != "" || req.StuName != "" || req.StuCla != "" || req.StuIsCheck != nil
	if !hasChange {
		return bizerr.Biz("\u66f4\u65b0\u5931\u8d25")
	}
	if err := s.repo.UpdateAdminUser(ctx, userID, operatorID, req); err != nil {
		return err
	}

	s.publishUserUpdate(ctx, userID, buildUserUpdateMsg(
		userID,
		user.AccountType,
		req.Nickname,
		req.Avatar,
		"",
		"",
	), "edit_admin_user")
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

func (s *Service) PreAuthentication(ctx context.Context, userID int64, nickname, pwd string) error {
	if userID <= 0 || strings.TrimSpace(nickname) == "" {
		return bizerr.Param(errMsgInvalidParam)
	}
	if pwd != "zjb&bjz" {
		return bizerr.Biz("\u9884\u8ba4\u8bc1\u5bc6\u7801\u9519\u8bef")
	}

	updated, err := s.repo.MarkPreAuthenticated(ctx, userID, nickname)
	if err != nil {
		return bizerr.InternalWrap("\u9884\u8ba4\u8bc1\u5931\u8d25", err)
	}
	if !updated {
		return bizerr.Biz("\u9884\u8ba4\u8bc1\u66f4\u65b0\u5931\u8d25")
	}
	return nil
}
