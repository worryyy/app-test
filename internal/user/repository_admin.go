package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func (r *Repository) FindAdminByUsername(ctx context.Context, username string) (*Admin, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var admin Admin
	if err := db.Where("username = ?", username).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("query admin by username %s: %w", username, err)
	}
	return &admin, nil
}

func (r *Repository) FindAdminByID(ctx context.Context, id int64) (*Admin, error) {
	if id <= 0 {
		return nil, nil
	}

	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var admin Admin
	if err := db.Where("id = ?", id).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("query admin by id %d: %w", id, err)
	}
	return &admin, nil
}

func (r *Repository) FindLegacyAdminUser(
	ctx context.Context,
	stuNum, encryptedPassword string,
	minPower int,
) (*User, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var user User
	if err := db.
		Where(colStuNum+" = ? AND "+colStuPwd+" = ? AND "+colPower+" >= ?", stuNum, encryptedPassword, minPower).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("query legacy admin user by stu_num %s: %w", stuNum, err)
	}
	return &user, nil
}

func (r *Repository) CreateAdmin(ctx context.Context, admin *Admin) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	if err := db.Create(admin).Error; err != nil {
		return fmt.Errorf("create admin: %w", err)
	}
	return nil
}

func (r *Repository) ListUsers(ctx context.Context, page, size int, filter AdminListUsersFilter) ([]User, int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, 0, err
	}

	query := db.Model(&User{})
	if filter.ID > 0 {
		query = query.Where("id = ?", filter.ID)
	}
	if strings.TrimSpace(filter.StuNum) != "" {
		query = query.Where("stu_num = ?", filter.StuNum)
	}
	if strings.TrimSpace(filter.Nickname) != "" {
		query = query.Where("nickname LIKE ?", "%"+filter.Nickname+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	var users []User
	if err := query.
		Offset((page - 1) * size).
		Limit(size).
		Order("id DESC").
		Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	return users, total, nil
}

func (r *Repository) UpdateAdminPasswordHash(ctx context.Context, adminID int64, hashedPassword string) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	if err := db.Model(&Admin{}).Where("id = ?", adminID).Update("password", hashedPassword).Error; err != nil {
		return fmt.Errorf("update admin password hash %d: %w", adminID, err)
	}
	return nil
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

func (r *Repository) ClearAuthenticationByRootUserID(ctx context.Context, rootUserID int64) (bool, error) {
	if rootUserID <= 0 {
		return false, nil
	}

	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}

	res := db.
		Model(&User{}).
		Where("root_user_id = ? OR id = ?", rootUserID, rootUserID).
		Updates(map[string]any{
			"stu_is_check": false,
			"stu_name":     "",
			"stu_cla":      "",
			"stu_num":      "",
			"stu_pwd":      "",
			"school":       "",
		})
	if res.Error != nil {
		return false, fmt.Errorf("clear authentication by root_user_id %d: %w", rootUserID, res.Error)
	}
	return res.RowsAffected > 0, nil
}
