package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

func (s *Service) CreateAnonymousIdentity(ctx context.Context, rootUserID int64) (*Identity, error) {
	rootUser, err := s.requireRootIdentity(ctx, rootUserID)
	if err != nil {
		return nil, err
	}

	s.identityMu.Lock()
	defer s.identityMu.Unlock()

	anonymous, err := s.getIdentityByType(ctx, rootUser.ID, accountTypeAnonymous)
	if err != nil {
		return nil, err
	}
	if anonymous != nil {
		return nil, bizerr.Biz("匿名身份已存在，无法重复创建")
	}

	anonymous = newAnonymousUser(rootUser, s.defaultAnonymousAvatar())

	if err := s.repo.CreateUser(ctx, anonymous); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateUserLastSwitch(ctx, anonymous.ID, anonymous.ID); err != nil {
		return nil, err
	}

	lastSwitchID := anonymous.ID
	anonymous.LastSwitchID = &lastSwitchID
	return buildIdentity(anonymous), nil
}

func newAnonymousUser(rootUser *User, avatar string) *User {
	if rootUser == nil {
		return nil
	}
	return &User{
		Avatar:               avatar,
		CreatedBy:            rootUser.ID,
		UpdatedBy:            rootUser.ID,
		Nickname:             randomAnonymousID(),
		OpenID:               fmt.Sprintf("%s:anon:%d", rootUser.OpenID, rootUser.ID),
		Power:                0,
		AccountType:          accountTypeAnonymous,
		RootUserID:           rootUser.ID,
		StuIsCheck:           rootUser.StuIsCheck,
		ProvisionalExpiresAt: rootUser.ProvisionalExpiresAt,
		Tag:                  rootUser.Tag,
		Gender:               rootUser.Gender,
	}
}

func (s *Service) UpdateAnonymousNickname(ctx context.Context, rootUserID int64, nickname string) error {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		return bizerr.Param(errMsgInvalidParam)
	}

	rootUser, err := s.requireRootIdentity(ctx, rootUserID)
	if err != nil {
		return err
	}

	anonymous, err := s.requireAnonymousIdentity(ctx, rootUser.ID)
	if err != nil {
		return err
	}

	if !anonymous.UpdatedAt.IsZero() {
		hoursSinceUpdate := int(time.Since(anonymous.UpdatedAt).Hours())
		if hoursSinceUpdate < anonymousNicknameUpdateHourLimit {
			return bizerr.Biz(
				fmt.Sprintf("昵称修改还需等待 %d 小时", anonymousNicknameUpdateHourLimit-hoursSinceUpdate),
			)
		}
	}

	return s.repo.UpdateAnonymousNickname(ctx, anonymous.ID, rootUser.ID, nickname)
}

func (s *Service) ListIdentities(ctx context.Context, rootUserID int64) (*IdentityListResp, error) {
	rootUser, err := s.requireRootIdentity(ctx, rootUserID)
	if err != nil {
		return nil, err
	}

	users, err := s.repo.FindUsersByRootUserID(ctx, rootUser.ID)
	if err != nil {
		return nil, err
	}

	identities := make([]*Identity, 0, len(users))
	hasAnonymous := false
	for i := range users {
		normalized := s.normalizedUser(&users[i])
		identities = append(identities, buildIdentity(normalized))
		if normalized != nil && normalized.AccountType == accountTypeAnonymous {
			hasAnonymous = true
		}
	}

	return &IdentityListResp{
		RootUserID:   rootUser.ID,
		Identities:   identities,
		HasAnonymous: hasAnonymous,
	}, nil
}

func (s *Service) SwitchIdentity(ctx context.Context, rootID, targetUserID int64) (string, string, *User, int64, error) {
	rootUser, err := s.requireRootIdentity(ctx, rootID)
	if err != nil {
		return "", "", nil, 0, err
	}

	targetUser, err := s.GetByID(ctx, targetUserID)
	if err != nil {
		return "", "", nil, 0, err
	}
	if targetUser == nil {
		return "", "", nil, 0, ErrUserNotFound
	}

	return s.switchToIdentity(ctx, rootUser, targetUser)
}

func (s *Service) SwitchIdentityByAccountType(
	ctx context.Context,
	rootUserID int64,
	accountType string,
) (string, string, *User, int64, error) {
	rootUser, err := s.requireRootIdentity(ctx, rootUserID)
	if err != nil {
		return "", "", nil, 0, err
	}

	targetUser, err := s.targetIdentityByAccountType(ctx, rootUser, accountType)
	if err != nil {
		return "", "", nil, 0, err
	}

	return s.switchToIdentity(ctx, rootUser, targetUser)
}

func (s *Service) RandomNickname(accountType string) (string, error) {
	switch strings.TrimSpace(accountType) {
	case accountTypeBase:
		return randomHumorousID(), nil
	case accountTypeAnonymous:
		return randomAnonymousID(), nil
	default:
		return "", bizerr.Param("accountType 非法")
	}
}

func (s *Service) requireRootIdentity(ctx context.Context, rootUserID int64) (*User, error) {
	rootUser, err := s.GetByID(ctx, rootUserID)
	if err != nil {
		return nil, err
	}
	if rootUser == nil {
		return nil, ErrUserNotFound
	}
	return rootUser, nil
}

func (s *Service) requireAnonymousIdentity(ctx context.Context, rootUserID int64) (*User, error) {
	anonymous, err := s.getIdentityByType(ctx, rootUserID, accountTypeAnonymous)
	if err != nil {
		return nil, err
	}
	if anonymous == nil {
		return nil, bizerr.Biz("匿名身份不存在")
	}
	return anonymous, nil
}

func (s *Service) targetIdentityByAccountType(ctx context.Context, rootUser *User, accountType string) (*User, error) {
	accountType = strings.TrimSpace(accountType)
	if accountType == "" {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	switch accountType {
	case accountTypeBase:
		return rootUser, nil
	case accountTypeAnonymous:
		targetUser, err := s.getIdentityByType(ctx, rootUser.ID, accountType)
		if err != nil {
			return nil, err
		}
		if targetUser == nil {
			return nil, bizerr.Biz("目标身份不存在，请先创建")
		}
		return targetUser, nil
	default:
		return nil, bizerr.Biz("accountType 非法")
	}
}

func (s *Service) switchToIdentity(
	ctx context.Context,
	rootUser *User,
	targetUser *User,
) (string, string, *User, int64, error) {
	if s.jwtHelper == nil {
		return "", "", nil, 0, bizerr.Internal("jwt helper not initialized")
	}
	if rootUserID(targetUser) != rootUser.ID {
		return "", "", nil, 0, ErrIdentityDenied
	}

	if err := s.persistLastSwitch(ctx, rootUser.ID, targetUser.ID); err != nil {
		return "", "", nil, 0, err
	}

	token, refreshToken, err := s.jwtHelper.GenerateTokenPair(s.buildTokenUser(targetUser, rootUser))
	if err != nil {
		return "", "", nil, 0, bizerr.InternalWrap("生成身份切换令牌失败", err)
	}

	return token, refreshToken, s.sanitizeUser(targetUser), rootUser.ID, nil
}

func (s *Service) defaultAnonymousAvatar() string {
	if s.cfg == nil {
		return ""
	}
	return s.cfg.Custom.DefaultAnonymousAvatar
}
