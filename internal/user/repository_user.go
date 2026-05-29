package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

func (r *Repository) FindUserByID(ctx context.Context, id int64) (*User, error) {
	if id <= 0 {
		return nil, nil
	}

	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var user User
	if err := db.Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("query user by id %d: %w", id, err)
	}
	return &user, nil
}

func (r *Repository) FindUserByOpenID(ctx context.Context, openID string) (*User, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var user User
	if err := db.Where("open_id = ?", openID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("query user by open_id %s: %w", openID, err)
	}
	return &user, nil
}

func (r *Repository) FindUserByRootAndAccountType(
	ctx context.Context,
	rootUserID int64,
	accountType string,
) (*User, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var user User
	if err := db.
		Where(colRootUserID+" = ? AND "+colAccountType+" = ?", rootUserID, accountType).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("query user by root_user_id/account_type: %w", err)
	}
	return &user, nil
}

func (r *Repository) FindUsersByRootUserID(ctx context.Context, rootUserID int64) ([]User, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := db.
		Where("root_user_id = ? OR id = ?", rootUserID, rootUserID).
		Order("id ASC").
		Find(&users).Error; err != nil {
		return nil, fmt.Errorf("list users by root_user_id %d: %w", rootUserID, err)
	}
	return users, nil
}

func (r *Repository) CreateUser(ctx context.Context, user *User) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	if err := db.Create(user).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *Repository) CountUsersByOpenID(ctx context.Context, openID string) (int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return 0, err
	}

	var count int64
	if err := db.Model(&User{}).Where("open_id = ?", openID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count users by open_id %s: %w", openID, err)
	}
	return count, nil
}

func (r *Repository) InitializeRootIdentity(ctx context.Context, userID int64) error {
	return r.updateUserFields(ctx, userID, map[string]any{
		"root_user_id":   userID,
		"last_switch_id": userID,
		"account_type":   accountTypeBase,
	})
}

func (r *Repository) UpdateUserLastSwitch(ctx context.Context, userID, targetUserID int64) error {
	if userID == 0 || targetUserID == 0 {
		return nil
	}
	return r.updateUserFields(ctx, userID, map[string]any{
		"last_switch_id": targetUserID,
	})
}

func (r *Repository) UpdateProvisionalExpiresAt(ctx context.Context, userID int64, expiresAt time.Time) error {
	return r.updateUserFields(ctx, userID, map[string]any{
		"provisional_expires_at": expiresAt,
		"updated_at":             time.Now(),
	})
}

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

func (r *Repository) updateUserFields(ctx context.Context, userID int64, updates map[string]any) error {
	if userID <= 0 || len(updates) == 0 {
		return nil
	}

	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	if err := db.Model(&User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return fmt.Errorf("update user %d: %w", userID, err)
	}
	return nil
}
