package user

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

type Repository struct {
	db      *gorm.DB
	mongoDB *mongo.Database
}

func NewRepository(db *gorm.DB, mongoDB *mongo.Database) *Repository {
	return &Repository{
		db:      db,
		mongoDB: mongoDB,
	}
}

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

func (r *Repository) DeleteUser(ctx context.Context, id int64) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	if err := db.Delete(&User{}, id).Error; err != nil {
		return fmt.Errorf("delete user %d: %w", id, err)
	}
	return nil
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

func (r *Repository) gormDB(ctx context.Context) (*gorm.DB, error) {
	if r.db == nil {
		return nil, errors.New("gorm db not initialized")
	}
	return r.db.WithContext(ctx), nil
}
