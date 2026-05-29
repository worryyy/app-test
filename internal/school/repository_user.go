package school

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

const schoolAccountTypeAnonymous = "anonymous"

func (r *Repository) FindUserByID(ctx context.Context, id int64) (*campusUser, error) {
	if id <= 0 {
		return nil, nil
	}

	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var user campusUser
	if err := db.Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("query user by id %d: %w", id, err)
	}
	return &user, nil
}

func (r *Repository) SaveAuthentication(
	ctx context.Context,
	userID int64,
	req AuthenticationReq,
	encryptedPassword string,
	name string,
	major string,
) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}

	updates := map[string]any{
		"stu_is_check": true,
		"stu_num":      req.SchoolID,
		"stu_pwd":      encryptedPassword,
		"school":       req.School,
	}
	if name != "" {
		updates["stu_name"] = name
	}
	if major != "" {
		updates["stu_cla"] = major
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&campusUser{}).
			Where("id = ?", userID).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("save authentication for user %d: %w", userID, err)
		}
		if err := tx.Model(&campusUser{}).
			Where("root_user_id = ? AND account_type = ?", userID, schoolAccountTypeAnonymous).
			Update("stu_is_check", true).Error; err != nil {
			return fmt.Errorf("sync anonymous authentication for user %d: %w", userID, err)
		}
		return nil
	})
}
