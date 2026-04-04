package user

import (
	"context"
	"time"
)

func (r *Repository) UpdateUserProfile(ctx context.Context, userID int64, req UserEditReq) error {
	updates := buildUserProfileUpdates(userID, req, time.Now())
	return r.updateUserFields(ctx, userID, updates)
}

func buildUserProfileUpdates(userID int64, req UserEditReq, now time.Time) map[string]any {
	req = normalizeUserEditReq(req)
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
	if len(updates) == 0 {
		return updates
	}
	if userID > 0 {
		updates["updated_by"] = userID
	}
	updates["updated_at"] = now
	return updates
}

func (r *Repository) UpdateAnonymousNickname(
	ctx context.Context,
	userID, updatedBy int64,
	nickname string,
) error {
	updates := map[string]any{
		"nickname":   nickname,
		"updated_at": time.Now(),
	}
	if updatedBy > 0 {
		updates["updated_by"] = updatedBy
	}
	return r.updateUserFields(ctx, userID, updates)
}
