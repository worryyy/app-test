package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"gorm.io/gorm"
)

const (
	accountTypeBase      = "base"
	accountTypeAnonymous = "anonymous"
	accountTypeOfficial  = "official"

	anonymousNicknameUpdateHourLimit = 72
	defaultAdminSecondaryPassword    = "pyhtip-nyxqen-6rigvE"

	certificationStatusPending  = "PENDING"
	certificationStatusApproved = "APPROVED"
	certificationStatusRejected = "REJECTED"
)

func (s *Service) sanitizeUser(user *User) *User {
	if user == nil {
		return nil
	}
	s.ensureUserDefaults(user)
	copyUser := *user
	copyUser.StuPwd = ""
	return &copyUser
}

func (s *Service) sanitizeUsers(users []User) []User {
	if len(users) == 0 {
		return []User{}
	}
	out := make([]User, 0, len(users))
	for i := range users {
		s.ensureUserDefaults(&users[i])
		users[i].StuPwd = ""
		out = append(out, users[i])
	}
	return out
}

func (s *Service) sanitizeUserByID(ctx context.Context, userID int64) (*User, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.sanitizeUser(user), nil
}

func (s *Service) ensureUserDefaults(user *User) {
	if user == nil {
		return
	}
	if user.RootUserID == 0 {
		user.RootUserID = user.ID
	}
	if user.AccountType == "" {
		if user.RootUserID == user.ID {
			user.AccountType = accountTypeBase
		} else {
			user.AccountType = accountTypeAnonymous
		}
	}
	if user.LastSwitchID == nil && user.RootUserID == user.ID {
		id := user.ID
		user.LastSwitchID = &id
	}
}

func (s *Service) getRootUser(ctx context.Context, user *User) (*User, error) {
	if user == nil {
		return nil, nil
	}
	s.ensureUserDefaults(user)
	if user.RootUserID == 0 || user.RootUserID == user.ID {
		return user, nil
	}
	rootUser, err := s.GetByID(ctx, user.RootUserID)
	if err != nil {
		return nil, err
	}
	if rootUser == nil {
		return user, nil
	}
	s.ensureUserDefaults(rootUser)
	return rootUser, nil
}

func (s *Service) resolveActiveIdentity(ctx context.Context, rootUser *User) (*User, error) {
	if rootUser == nil {
		return nil, nil
	}
	s.ensureUserDefaults(rootUser)
	if rootUser.LastSwitchID == nil || *rootUser.LastSwitchID == rootUser.ID {
		return rootUser, s.persistLastSwitch(ctx, rootUser.ID, rootUser.ID)
	}

	target, err := s.GetByID(ctx, *rootUser.LastSwitchID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return rootUser, s.persistLastSwitch(ctx, rootUser.ID, rootUser.ID)
	}
	s.ensureUserDefaults(target)
	if rootUserID(target) != rootUser.ID {
		return rootUser, s.persistLastSwitch(ctx, rootUser.ID, rootUser.ID)
	}
	return target, nil
}

func (s *Service) getIdentityByType(ctx context.Context, rootUserID int64, accountType string) (*User, error) {
	var user User
	err := s.db.WithContext(ctx).
		Where("root_user_id = ? AND accountType = ?", rootUserID, accountType).
		First(&user).Error
	if err != nil {
		if isRecordNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find identity by type: %w", err)
	}
	s.ensureUserDefaults(&user)
	return &user, nil
}

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func (s *Service) persistLastSwitch(ctx context.Context, rootUserID, targetUserID int64) error {
	if rootUserID == 0 || targetUserID == 0 {
		return nil
	}
	if err := s.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", rootUserID).
		Update("last_switch_id", targetUserID).Error; err != nil {
		return fmt.Errorf("update last switch id: %w", err)
	}
	return nil
}

func (s *Service) buildTokenUser(identity, rootUser *User) *jwtutil.TokenUser {
	if identity == nil || rootUser == nil {
		return nil
	}
	s.ensureUserDefaults(identity)
	s.ensureUserDefaults(rootUser)
	return &jwtutil.TokenUser{
		ID:          identity.ID,
		OpenID:      rootUser.OpenID,
		Power:       rootUser.Power,
		AccountType: identity.AccountType,
		RootUserID:  rootUser.ID,
	}
}
