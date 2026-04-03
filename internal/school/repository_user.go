package school

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

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
	loginResp *JWLoginData,
	encryptedPassword string,
) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}

	if err := db.Model(&campusUser{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"stu_is_check": true,
			"stu_num":      req.SchoolID,
			"stu_cla":      loginResp.Major,
			"stu_name":     loginResp.Name,
			"stu_pwd":      encryptedPassword,
			"school":       req.School,
		}).Error; err != nil {
		return fmt.Errorf("save authentication for user %d: %w", userID, err)
	}
	return nil
}
