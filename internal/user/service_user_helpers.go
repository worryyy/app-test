package user

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
)

const (
	accountTypeBase      = "base"
	accountTypeAnonymous = "anonymous"
	accountTypeOfficial  = "official"

	anonymousNicknameUpdateHourLimit = 72
	defaultAdminSecondaryPassword    = "pyhtip-nyxqen-6rigvE"
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

func (s *Service) defaultPageSize() int {
	if s.cfg != nil && s.cfg.Custom.PageSize > 0 {
		return s.cfg.Custom.PageSize
	}
	return 15
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
	user, err := s.repo.FindUserByRootAndAccountType(ctx, rootUserID, accountType)
	if err != nil {
		return nil, err
	}
	s.ensureUserDefaults(user)
	return user, nil
}

func (s *Service) persistLastSwitch(ctx context.Context, rootUserID, targetUserID int64) error {
	return s.repo.UpdateUserLastSwitch(ctx, rootUserID, targetUserID)
}

func (s *Service) buildTokenUser(identity, rootUser *User) *jwtutil.TokenUser {
	return s.buildTokenUserWithPower(identity, rootUser, 0)
}

func (s *Service) buildAdminTokenUser(identity, rootUser *User) *jwtutil.TokenUser {
	power := 0
	if identity != nil {
		power = identity.Power
	}
	return s.buildTokenUserWithPower(identity, rootUser, power)
}

func (s *Service) buildTokenUserWithPower(identity, rootUser *User, power int) *jwtutil.TokenUser {
	if identity == nil || rootUser == nil {
		return nil
	}

	s.ensureUserDefaults(identity)
	s.ensureUserDefaults(rootUser)
	return &jwtutil.TokenUser{
		ID:          identity.ID,
		OpenID:      rootUser.OpenID,
		Power:       power,
		AccountType: identity.AccountType,
		RootUserID:  rootUser.ID,
	}
}
