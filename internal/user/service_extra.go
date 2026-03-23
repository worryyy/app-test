package user

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/encrypt"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) RandomNickname() string {
	return s.randomNickname()
}

func (s *Service) PreAuthentication(ctx context.Context, stuNum, stuPwd string) error {
	if stuNum == "" || stuPwd == "" {
		return result.ErrParam
	}
	return nil
}

func (s *Service) Authenticate(ctx context.Context, userID int64, stuNum, stuPwd string) error {
	if stuNum == "" || stuPwd == "" {
		return result.ErrParam
	}
	encPwd := stuPwd
	if s.cfg != nil && s.cfg.Encryption.Key != "" {
		if v, err := encrypt.AESEncrypt(stuPwd, s.cfg.Encryption.Key); err == nil {
			encPwd = v
		}
	}
	if err := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"stuNum":     stuNum,
		"stuPwd":     encPwd,
		"stuIsCheck": true,
	}).Error; err != nil {
		return fmt.Errorf("authenticate user: %w", err)
	}
	return nil
}

func (s *Service) ReAuthentication(ctx context.Context, userID int64, stuNum, stuPwd string) error {
	return s.Authenticate(ctx, userID, stuNum, stuPwd)
}

func (s *Service) DelAuthentication(ctx context.Context, userID int64) error {
	if err := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"stuNum":     "",
		"stuPwd":     "",
		"stuName":    "",
		"stuCla":     "",
		"stuIsCheck": false,
	}).Error; err != nil {
		return fmt.Errorf("delete authentication: %w", err)
	}
	return nil
}

func (s *Service) CheckLogin(ctx context.Context, userID int64) (bool, error) {
	u, err := s.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if u == nil {
		return false, ErrUserNotFound
	}
	return u.StuIsCheck, nil
}

func (s *Service) GetCourseByWeeks(ctx context.Context, userID int64, term string, weeks []int) ([]map[string]interface{}, error) {
	_ = ctx
	_ = userID
	_ = term
	_ = weeks
	return []map[string]interface{}{}, nil
}

func (s *Service) GetExam(ctx context.Context, userID int64) ([]map[string]interface{}, error) {
	_ = ctx
	_ = userID
	return []map[string]interface{}{}, nil
}

func (s *Service) GetExamScore(ctx context.Context, userID int64) ([]map[string]interface{}, error) {
	_ = ctx
	_ = userID
	return []map[string]interface{}{}, nil
}

func (s *Service) OfficialLogin(ctx context.Context, username, password string) (string, string, *User, error) {
	if username == "" || password == "" {
		return "", "", nil, result.ErrParam
	}
	encPwd := password
	if s.cfg != nil && s.cfg.Encryption.Key != "" {
		if v, err := encrypt.AESEncrypt(password, s.cfg.Encryption.Key); err == nil {
			encPwd = v
		}
	}

	var u User
	err := s.db.WithContext(ctx).Where("stuNum = ? AND accountType = ?", username, "official").First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", nil, ErrUserNotFound
	}
	if err != nil {
		return "", "", nil, fmt.Errorf("query official user: %w", err)
	}
	if u.StuPwd != encPwd {
		return "", "", nil, errors.New("账号或密码错误")
	}

	token, refreshToken, err := s.jwtHelper.GenerateTokenPair(&jwtutil.TokenUser{
		ID:          u.ID,
		OpenID:      u.OpenID,
		Power:       u.Power,
		AccountType: u.AccountType,
		RootUserID:  u.RootUserID,
	})
	if err != nil {
		return "", "", nil, err
	}
	u.StuPwd = ""
	return token, refreshToken, &u, nil
}

func (s *Service) SubmitOfficialCertification(ctx context.Context, userID int64, name, reason string) error {
	doc := OfficialCertification{
		UserID:    strconv.FormatInt(userID, 10),
		Name:      name,
		Reason:    reason,
		Status:    0,
		CreatedAt: time.Now(),
	}
	if _, err := s.mongoDB.Collection("campus_official_certification").InsertOne(ctx, doc); err != nil {
		return fmt.Errorf("insert official certification: %w", err)
	}
	return nil
}

func (s *Service) GenerateUnlimitedWXACode(ctx context.Context, scene, page string) ([]byte, error) {
	_ = ctx
	_ = scene
	_ = page
	return []byte{}, nil
}
