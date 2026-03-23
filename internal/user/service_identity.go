package user

import (
	"context"
	"fmt"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
)

func (s *Service) CreateAnonymousIdentity(ctx context.Context, currentUserID int64, nickname string) (*User, error) {
	baseUser, err := s.GetByID(ctx, currentUserID)
	if err != nil {
		return nil, err
	}
	if baseUser == nil {
		return nil, ErrUserNotFound
	}

	rootID := baseUser.RootUserID
	if rootID == 0 {
		rootID = baseUser.ID
	}

	if nickname == "" {
		nickname = s.randomNickname()
	}

	u := &User{
		OpenID:      baseUser.OpenID,
		Nickname:    nickname,
		Avatar:      s.cfg.Custom.DefaultAnonymousAvatar,
		Power:       baseUser.Power,
		AccountType: "anonymous",
		Tag:         baseUser.Tag,
		Gender:      baseUser.Gender,
		RootUserID:  rootID,
	}

	if err := s.db.WithContext(ctx).Create(u).Error; err != nil {
		return nil, fmt.Errorf("create anonymous identity: %w", err)
	}
	u.StuPwd = ""
	return u, nil
}

func (s *Service) UpdateAnonymousNickname(ctx context.Context, userID int64, nickname string) error {
	if nickname == "" {
		return nil
	}
	if err := s.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ? AND accountType = ?", userID, "anonymous").
		Update("nickname", nickname).Error; err != nil {
		return fmt.Errorf("update anonymous nickname: %w", err)
	}
	return nil
}

func (s *Service) ListIdentities(ctx context.Context, currentUserID int64) ([]User, error) {
	current, err := s.GetByID(ctx, currentUserID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, ErrUserNotFound
	}

	rootID := current.RootUserID
	if rootID == 0 {
		rootID = current.ID
	}

	var users []User
	if err := s.db.WithContext(ctx).Where("rootUserId = ?", rootID).Order("id ASC").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("list identities: %w", err)
	}
	for i := range users {
		users[i].StuPwd = ""
	}
	return users, nil
}

func (s *Service) SwitchIdentity(ctx context.Context, currentUserID, targetUserID int64) (string, string, error) {
	if s.jwtHelper == nil {
		return "", "", fmt.Errorf("jwt helper not initialized")
	}

	current, err := s.GetByID(ctx, currentUserID)
	if err != nil {
		return "", "", err
	}
	target, err := s.GetByID(ctx, targetUserID)
	if err != nil {
		return "", "", err
	}
	if current == nil || target == nil {
		return "", "", ErrUserNotFound
	}

	curRoot := current.RootUserID
	if curRoot == 0 {
		curRoot = current.ID
	}
	targetRoot := target.RootUserID
	if targetRoot == 0 {
		targetRoot = target.ID
	}
	if curRoot != targetRoot {
		return "", "", ErrIdentityDenied
	}

	current.LastSwitchID = &target.ID
	if err := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", current.ID).Update("lastSwitchId", target.ID).Error; err != nil {
		return "", "", fmt.Errorf("update last switch id: %w", err)
	}

	return s.jwtHelper.GenerateTokenPair(&jwtutil.TokenUser{
		ID:          target.ID,
		OpenID:      target.OpenID,
		Power:       target.Power,
		AccountType: target.AccountType,
		RootUserID:  target.RootUserID,
	})
}
