package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
)

func (s *Service) CreateAnonymousIdentity(ctx context.Context, rootUserID int64) (*Identity, error) {
	rootUser, err := s.GetByID(ctx, rootUserID)
	if err != nil {
		return nil, err
	}
	if rootUser == nil {
		return nil, ErrUserNotFound
	}
	s.ensureUserDefaults(rootUser)

	existing, err := s.getIdentityByType(ctx, rootUser.ID, accountTypeAnonymous)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, bizerr.Biz("匿名身份已存在，无法重复创建")
	}

	anonymous := &User{
		Avatar:      s.defaultAnonymousAvatar(),
		CreatedBy:   rootUser.ID,
		UpdatedBy:   rootUser.ID,
		Nickname:    randomAnonymousID(),
		OpenID:      fmt.Sprintf("%s:anon:%d", rootUser.OpenID, rootUser.ID),
		Power:       0,
		AccountType: accountTypeAnonymous,
		RootUserID:  rootUser.ID,
		StuIsCheck:  true,
		Tag:         rootUser.Tag,
		Gender:      rootUser.Gender,
		Signature:   "",
	}

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

func (s *Service) UpdateAnonymousNickname(ctx context.Context, rootUserID int64, nickname string) error {
	if strings.TrimSpace(nickname) == "" {
		return bizerr.Param(errMsgInvalidParam)
	}

	rootUser, err := s.GetByID(ctx, rootUserID)
	if err != nil {
		return err
	}
	if rootUser == nil {
		return ErrUserNotFound
	}

	anonymous, err := s.getIdentityByType(ctx, rootUserID, accountTypeAnonymous)
	if err != nil {
		return err
	}
	if anonymous == nil {
		return bizerr.Biz("匿名身份不存在")
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
	rootUser, err := s.GetByID(ctx, rootUserID)
	if err != nil {
		return nil, err
	}
	if rootUser == nil {
		return nil, ErrUserNotFound
	}

	users, err := s.repo.FindUsersByRootUserID(ctx, rootUserID)
	if err != nil {
		return nil, err
	}

	identities := make([]*Identity, 0, len(users))
	hasAnonymous := false
	for i := range users {
		s.ensureUserDefaults(&users[i])
		identities = append(identities, buildIdentity(&users[i]))
		if users[i].AccountType == accountTypeAnonymous {
			hasAnonymous = true
		}
	}

	return &IdentityListResp{
		RootUserID:   rootUserID,
		Identities:   identities,
		HasAnonymous: hasAnonymous,
	}, nil
}

func (s *Service) SwitchIdentity(ctx context.Context, rootID, targetUserID int64) (string, string, *User, int64, error) {
	if s.jwtHelper == nil {
		return "", "", nil, 0, fmt.Errorf("jwt helper not initialized")
	}

	rootUser, err := s.GetByID(ctx, rootID)
	if err != nil {
		return "", "", nil, 0, err
	}
	targetUser, err := s.GetByID(ctx, targetUserID)
	if err != nil {
		return "", "", nil, 0, err
	}
	if rootUser == nil || targetUser == nil {
		return "", "", nil, 0, ErrUserNotFound
	}
	if rootUserID(targetUser) != rootUser.ID {
		return "", "", nil, 0, ErrIdentityDenied
	}

	if err := s.persistLastSwitch(ctx, rootUser.ID, targetUser.ID); err != nil {
		return "", "", nil, 0, err
	}

	token, refreshToken, err := s.jwtHelper.GenerateTokenPair(s.buildTokenUser(targetUser, rootUser))
	if err != nil {
		return "", "", nil, 0, err
	}
	return token, refreshToken, s.sanitizeUser(targetUser), rootUser.ID, nil
}

func (s *Service) SwitchIdentityByAccountType(
	ctx context.Context,
	rootUserID int64,
	accountType string,
) (string, string, *User, int64, error) {
	rootUser, err := s.GetByID(ctx, rootUserID)
	if err != nil {
		return "", "", nil, 0, err
	}
	if rootUser == nil {
		return "", "", nil, 0, ErrUserNotFound
	}

	accountType = strings.TrimSpace(accountType)
	if accountType == "" {
		return "", "", nil, 0, bizerr.Param(errMsgInvalidParam)
	}
	if accountType != accountTypeBase && accountType != accountTypeAnonymous && accountType != accountTypeOfficial {
		return "", "", nil, 0, bizerr.Biz("account_type 非法")
	}
	if accountType == accountTypeBase {
		return s.SwitchIdentity(ctx, rootUser.ID, rootUser.ID)
	}

	targetUser, err := s.getIdentityByType(ctx, rootUser.ID, accountType)
	if err != nil {
		return "", "", nil, 0, err
	}
	if targetUser == nil {
		return "", "", nil, 0, bizerr.Biz("目标身份不存在，请先创建")
	}
	return s.SwitchIdentity(ctx, rootUser.ID, targetUser.ID)
}

func (s *Service) defaultAnonymousAvatar() string {
	if s.cfg == nil {
		return ""
	}
	return s.cfg.Custom.DefaultAnonymousAvatar
}
