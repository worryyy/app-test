package user

import (
	"context"
	"fmt"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
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
		return nil, result.NewBizError(result.CodeFail, "匿名身份已存在，无法重复创建")
	}

	anonymous := &User{
		Avatar:       s.defaultAnonymousAvatar(),
		CreatedBy:    rootUser.ID,
		UpdatedBy:    rootUser.ID,
		Nickname:     randomAnonymousID(),
		OpenID:       fmt.Sprintf("%s:anon:%d", rootUser.OpenID, rootUser.ID),
		Power:        0,
		AccountType:  accountTypeAnonymous,
		RootUserID:   rootUser.ID,
		StuIsCheck:   true,
		Tag:          rootUser.Tag,
		Gender:       rootUser.Gender,
		Signature:    "",
		LastSwitchID: nil,
	}

	if err := s.db.WithContext(ctx).Create(anonymous).Error; err != nil {
		return nil, fmt.Errorf("create anonymous identity: %w", err)
	}
	anonymous.LastSwitchID = &anonymous.ID
	if err := s.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", anonymous.ID).
		Update("last_switch_id", anonymous.ID).Error; err != nil {
		return nil, fmt.Errorf("update anonymous last switch id: %w", err)
	}

	return buildIdentity(anonymous), nil
}

func (s *Service) UpdateAnonymousNickname(ctx context.Context, rootUserID int64, nickname string) error {
	if nickname == "" {
		return result.ErrParam
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
		return result.NewBizError(result.CodeFail, "匿名身份不存在")
	}

	if !anonymous.UpdatedAt.IsZero() {
		hoursSinceUpdate := int(time.Since(anonymous.UpdatedAt).Hours())
		if hoursSinceUpdate < anonymousNicknameUpdateHourLimit {
			return result.NewBizError(
				result.CodeFail,
				fmt.Sprintf("昵称修改还需等待 %d 小时", anonymousNicknameUpdateHourLimit-hoursSinceUpdate),
			)
		}
	}

	if err := s.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", anonymous.ID).
		Updates(map[string]interface{}{
			"nickname":  nickname,
			"updatedBy": rootUser.ID,
		}).Error; err != nil {
		return fmt.Errorf("update anonymous nickname: %w", err)
	}
	return nil
}

func (s *Service) ListIdentities(ctx context.Context, rootUserID int64) (*IdentityListResp, error) {
	rootUser, err := s.GetByID(ctx, rootUserID)
	if err != nil {
		return nil, err
	}
	if rootUser == nil {
		return nil, ErrUserNotFound
	}

	var users []User
	if err := s.db.WithContext(ctx).
		Where("root_user_id = ?", rootUserID).
		Order("id ASC").
		Find(&users).Error; err != nil {
		return nil, fmt.Errorf("list identities: %w", err)
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

func (s *Service) SwitchIdentity(ctx context.Context, rootID int64, targetUserID int64) (string, string, *User, int64, error) {
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

	if accountType == "" {
		return "", "", nil, 0, result.ErrParam
	}
	if accountType != accountTypeBase && accountType != accountTypeAnonymous && accountType != accountTypeOfficial {
		return "", "", nil, 0, result.NewBizError(result.CodeFail, "account_type 非法")
	}
	if accountType == accountTypeBase {
		return s.SwitchIdentity(ctx, rootUser.ID, rootUser.ID)
	}

	targetUser, err := s.getIdentityByType(ctx, rootUser.ID, accountType)
	if err != nil {
		return "", "", nil, 0, err
	}
	if targetUser == nil {
		return "", "", nil, 0, result.NewBizError(result.CodeFail, "目标身份不存在，请先创建")
	}
	return s.SwitchIdentity(ctx, rootUser.ID, targetUser.ID)
}

func (s *Service) defaultAnonymousAvatar() string {
	if s.cfg == nil {
		return ""
	}
	return s.cfg.Custom.DefaultAnonymousAvatar
}
