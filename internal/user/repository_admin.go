package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

func (r *Repository) CountAdminsByUsername(ctx context.Context, username string) (int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return 0, err
	}

	var count int64
	if err := db.Model(&Admin{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count admins by username %s: %w", username, err)
	}
	return count, nil
}

func (r *Repository) CountAdminsByUserID(ctx context.Context, userID int64) (int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return 0, err
	}

	var count int64
	if err := db.Model(&Admin{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count admins by user_id %d: %w", userID, err)
	}
	return count, nil
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

func (r *Repository) ListUsers(ctx context.Context, page, size int, nickname string) ([]User, int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, 0, err
	}

	query := db.Model(&User{})
	if strings.TrimSpace(nickname) != "" {
		query = query.Where("nickname = ?", nickname)
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

func (r *Repository) FindCourseFileByKey(ctx context.Context, key string) (*CourseFile, error) {
	coll, err := r.courseCollection()
	if err != nil {
		return nil, err
	}

	var course CourseFile
	if err := coll.FindOne(ctx, bson.M{"key": key}).Decode(&course); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find course by key %s: %w", key, err)
	}
	return &course, nil
}

func (r *Repository) courseCollection() (*mongo.Collection, error) {
	if r.mongoDB == nil {
		return nil, errors.New("mongo db not initialized")
	}
	return r.mongoDB.Collection("campus_course"), nil
}

type CourseFile struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Key      string             `bson:"key"`
	FilePath string             `bson:"filePath"`
	Val      []byte             `bson:"val"`
}
