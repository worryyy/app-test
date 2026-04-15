package user

import (
	"context"
	"fmt"
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

func (s *Service) ListUsers(
	ctx context.Context,
	page, size int,
	filter AdminListUsersFilter,
) (*PageResult[User], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = s.defaultPageSize()
	}

	list, total, err := s.repo.ListUsers(ctx, page, size, filter)
	if err != nil {
		return nil, err
	}
	return NewPageResult(s.sanitizeUsers(list), total, page, size), nil
}

func (s *Service) ClearAuthentication(ctx context.Context, userID int64) error {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	cleared, err := s.repo.ClearAuthenticationByRootUserID(ctx, rootUserID(user))
	if err != nil {
		return err
	}
	if !cleared {
		return ErrAuthenticationClearFailed
	}
	return nil
}

func (s *Service) PreAuthentication(ctx context.Context, userID int64, nickname, pwd string) error {
	if err := s.legacyPreAuthError(userID, nickname, pwd); err != nil {
		return err
	}
	resp, err := s.PreAuthenticationBatch(ctx, AdminBatchPreAuthReq{
		Password: pwd,
		Items: []AdminPreAuthItem{
			{UserID: userID, NickName: nickname},
		},
	})
	if err != nil {
		return err
	}
	if resp.SuccessCount != 1 || len(resp.Results) == 0 || !resp.Results[0].Success {
		message := "预认证更新失败"
		if len(resp.Results) > 0 && strings.TrimSpace(resp.Results[0].Message) != "" {
			message = resp.Results[0].Message
		}
		return bizerr.Biz(message)
	}
	return nil
}

func (s *Service) PreAuthenticationBatch(
	ctx context.Context,
	req AdminBatchPreAuthReq,
) (*AdminBatchPreAuthResp, error) {
	if strings.TrimSpace(req.Password) != s.adminPreAuthPassword() {
		return nil, bizerr.Biz("预认证密码错误")
	}
	if len(req.Items) == 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	resp := &AdminBatchPreAuthResp{
		Total:   len(req.Items),
		Results: make([]AdminPreAuthResult, 0, len(req.Items)),
	}

	for _, item := range req.Items {
		result := AdminPreAuthResult{UserID: item.UserID}
		userID := item.UserID
		nickname := strings.TrimSpace(item.NickName)
		switch {
		case userID <= 0 || nickname == "":
			result.Message = errMsgInvalidParam
		default:
			updated, err := s.repo.MarkPreAuthenticated(ctx, userID, nickname)
			if err != nil {
				return nil, bizerr.InternalWrap("预认证失败", err)
			}
			if updated {
				result.Success = true
				result.Message = preAuthSuccessMessage
			} else {
				result.Message = fmt.Sprintf("userId=%d 预认证更新失败", userID)
			}
		}

		if result.Success {
			resp.SuccessCount++
		} else {
			resp.FailureCount++
		}
		resp.Results = append(resp.Results, result)
	}
	return resp, nil
}

func (s *Service) adminPreAuthPassword() string {
	if s.cfg == nil {
		return defaultAdminPreAuthPassword
	}

	pwd := strings.TrimSpace(s.cfg.Admin.PreAuthPassword)
	if pwd == "" || pwd == "replace-me" {
		return defaultAdminPreAuthPassword
	}
	return pwd
}

func (s *Service) legacyPreAuthError(userID int64, nickname, pwd string) error {
	if userID <= 0 || strings.TrimSpace(nickname) == "" {
		return bizerr.Param(errMsgInvalidParam)
	}
	if strings.TrimSpace(pwd) != s.adminPreAuthPassword() {
		return bizerr.Biz("\u9884\u8ba4\u8bc1\u5bc6\u7801\u9519\u8bef")
	}
	return nil
}
