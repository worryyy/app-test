package user

import (
	"context"
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


