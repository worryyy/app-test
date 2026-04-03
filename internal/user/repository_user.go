package user

import (
	"context"
	"fmt"
)

func (r *Repository) UpdateUserProfile(ctx context.Context, userID int64, req UserEditReq) error {
	updates := map[string]any{}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Gender != "" {
		updates["gender"] = req.Gender
	}
	if req.Signature != "" {
		updates["signature"] = req.Signature
	}
	return r.updateUserFields(ctx, userID, updates)
}

func (r *Repository) MarkPreAuthenticated(ctx context.Context, userID int64, nickname string) (bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}

	res := db.
		Model(&User{}).
		Where("id = ? AND nickname = ?", userID, nickname).
		Update("stu_is_check", true)
	if res.Error != nil {
		return false, fmt.Errorf("pre-authenticate user %d: %w", userID, res.Error)
	}
	return res.RowsAffected > 0, nil
}

func (r *Repository) SaveAuthentication(
	ctx context.Context,
	userID int64,
	req AuthenticationReq,
	loginResp *JWLoginData,
	encryptedPassword string,
) error {
	return r.updateUserFields(ctx, userID, map[string]any{
		"stu_is_check": true,
		"stu_num":      req.SchoolID,
		"stu_cla":      loginResp.Major,
		"stu_name":     loginResp.Name,
		"stu_pwd":      encryptedPassword,
		"school":       req.School,
	})
}

func (r *Repository) DeleteAuthentication(ctx context.Context, userID int64) error {
	return r.updateUserFields(ctx, userID, map[string]any{
		"stu_num":      "",
		"stu_pwd":      "",
		"stu_name":     "",
		"stu_cla":      "",
		"stu_is_check": false,
	})
}

func (r *Repository) ClearAuthentication(ctx context.Context, userID int64) error {
	return r.updateUserFields(ctx, userID, map[string]any{
		"stu_is_check": false,
		"stu_name":     "",
		"stu_cla":      "",
		"stu_num":      "",
	})
}

func (r *Repository) UpdateAnonymousNickname(
	ctx context.Context,
	userID, updatedBy int64,
	nickname string,
) error {
	updates := map[string]any{
		"nickname": nickname,
	}
	if updatedBy > 0 {
		updates["updated_by"] = updatedBy
	}
	return r.updateUserFields(ctx, userID, updates)
}

func (r *Repository) UpdateAdminUser(
	ctx context.Context,
	userID, operatorID int64,
	req AdminEditUserReq,
) error {
	updates := map[string]any{}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Power != nil {
		updates["power"] = *req.Power
	}
	if req.StuNum != "" {
		updates["stu_num"] = req.StuNum
	}
	if req.StuName != "" {
		updates["stu_name"] = req.StuName
	}
	if req.StuCla != "" {
		updates["stu_cla"] = req.StuCla
	}
	if req.StuIsCheck != nil {
		updates["stu_is_check"] = *req.StuIsCheck
	}
	if len(updates) == 0 {
		return nil
	}
	if operatorID > 0 {
		updates["updated_by"] = operatorID
	}
	return r.updateUserFields(ctx, userID, updates)
}
