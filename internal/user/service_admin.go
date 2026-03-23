package user

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/encrypt"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) AdminLogin(ctx context.Context, req *AdminLoginReq) (string, string, *User, error) {
	if req == nil {
		return "", "", nil, result.ErrParam
	}
	if s.jwtHelper == nil {
		return "", "", nil, errors.New("jwt helper not initialized")
	}

	lockKey := rediskey.AdminLoginLock(req.Username)
	failCountKey := rediskey.AdminLoginFail(req.Username)

	locked, err := s.redis.Exists(ctx, lockKey).Result()
	if err != nil {
		return "", "", nil, fmt.Errorf("check admin lock: %w", err)
	}
	if locked > 0 {
		return "", "", nil, errors.New("账号已锁定，请明天后再试")
	}

	if s.cfg != nil && s.cfg.Admin.SecondaryPassword != "" && req.SecondaryPassword != s.cfg.Admin.SecondaryPassword {
		remaining, countErr := s.handleLoginFail(ctx, failCountKey, lockKey)
		if countErr != nil {
			return "", "", nil, countErr
		}
		return "", "", nil, fmt.Errorf("二级密码错误，今日还有 %d 次机会", remaining)
	}

	var admin Admin
	err = s.db.WithContext(ctx).Where("username = ?", req.Username).First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		legacyUser, lErr := s.loadLegacyAdmin(ctx, req.Username, req.Password)
		if lErr != nil {
			return "", "", nil, lErr
		}
		if legacyUser == nil {
			remaining, countErr := s.handleLoginFail(ctx, failCountKey, lockKey)
			if countErr != nil {
				return "", "", nil, countErr
			}
			return "", "", nil, fmt.Errorf("账号或密码错误，今日还有 %d 次机会", remaining)
		}
		admin, err = s.migrateLegacyAdmin(ctx, legacyUser, req.Username, req.Password)
		if err != nil {
			return "", "", nil, err
		}
	} else if err != nil {
		return "", "", nil, fmt.Errorf("query admin: %w", err)
	}

	if admin.Password != md5Hex(req.Password) {
		remaining, countErr := s.handleLoginFail(ctx, failCountKey, lockKey)
		if countErr != nil {
			return "", "", nil, countErr
		}
		return "", "", nil, fmt.Errorf("账号或密码错误，今日还有 %d 次机会", remaining)
	}

	u, err := s.GetByID(ctx, admin.UserID)
	if err != nil {
		return "", "", nil, err
	}
	if u == nil {
		return "", "", nil, errors.New("管理员关联用户不存在")
	}

	if err := s.redis.Del(ctx, failCountKey).Err(); err != nil {
		return "", "", nil, fmt.Errorf("clear admin fail count: %w", err)
	}

	u.Power = admin.Power
	u.StuPwd = ""
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
	return token, refreshToken, u, nil
}

func (s *Service) handleLoginFail(ctx context.Context, failCountKey, lockKey string) (int, error) {
	count, err := s.redis.Incr(ctx, failCountKey).Result()
	if err != nil {
		return 0, fmt.Errorf("increase login fail count: %w", err)
	}
	if count == 1 {
		if err := s.redis.Expire(ctx, failCountKey, 24*time.Hour).Err(); err != nil {
			return 0, fmt.Errorf("set fail count ttl: %w", err)
		}
	}
	if count >= 10 {
		if err := s.redis.Set(ctx, lockKey, "locked", 24*time.Hour).Err(); err != nil {
			return 0, fmt.Errorf("set admin lock: %w", err)
		}
	}
	remaining := int(10 - count)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

func (s *Service) loadLegacyAdmin(ctx context.Context, stuNum, rawPwd string) (*User, error) {
	if s.cfg == nil {
		return nil, nil
	}
	encPwd, err := encrypt.AESEncrypt(rawPwd, s.cfg.Encryption.Key)
	if err != nil {
		return nil, fmt.Errorf("legacy password encrypt: %w", err)
	}

	var u User
	err = s.db.WithContext(ctx).
		Where("stuNum = ? AND stuPwd = ? AND power >= 8", stuNum, encPwd).
		First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load legacy admin: %w", err)
	}
	return &u, nil
}

func (s *Service) migrateLegacyAdmin(ctx context.Context, u *User, username, rawPwd string) (Admin, error) {
	admin := Admin{
		UserID:   u.ID,
		Username: username,
		Password: md5Hex(rawPwd),
		Power:    resolveAdminPower(u.Power),
	}
	if err := s.db.WithContext(ctx).Create(&admin).Error; err != nil {
		return Admin{}, fmt.Errorf("migrate legacy admin: %w", err)
	}
	return admin, nil
}

func resolveAdminPower(power int) int {
	if power >= 999 {
		return 999
	}
	return 2
}

func md5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
