package user

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/encrypt"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
)

func (s *Service) AdminLogin(ctx context.Context, req *AdminLoginReq) (string, string, *User, error) {
	if req == nil {
		return "", "", nil, bizerr.Param(errMsgInvalidParam)
	}
	if s.jwtHelper == nil {
		return "", "", nil, bizerr.Internal("jwt helper not initialized")
	}
	if s.redis == nil {
		return "", "", nil, bizerr.Internal("redis client not initialized")
	}

	lockKey := rediskey.AdminLoginLock(req.Username)
	failCountKey := rediskey.AdminLoginFail(req.Username)

	locked, err := s.redis.Exists(ctx, lockKey).Result()
	if err != nil {
		return "", "", nil, bizerr.InternalWrap("检查管理员账号锁定状态失败", err)
	}
	if locked > 0 {
		return "", "", nil, bizerr.Biz("账号已锁定，请明天后再试")
	}

	if req.SecondaryPassword != s.adminSecondaryPassword() {
		remaining, countErr := s.handleLoginFail(ctx, failCountKey, lockKey)
		if countErr != nil {
			return "", "", nil, countErr
		}
		return "", "", nil, bizerr.Biz("二级密码错误，今日还有 " + strconv.Itoa(remaining) + " 次机会")
	}

	admin, err := s.repo.FindAdminByUsername(ctx, req.Username)
	if err != nil {
		return "", "", nil, bizerr.InternalWrap("查询管理员失败", err)
	}
	if admin == nil {
		legacyUser, legacyErr := s.loadLegacyAdmin(ctx, req.Username, req.Password)
		if legacyErr != nil {
			return "", "", nil, legacyErr
		}
		if legacyUser == nil {
			remaining, countErr := s.handleLoginFail(ctx, failCountKey, lockKey)
			if countErr != nil {
				return "", "", nil, countErr
			}
			return "", "", nil, bizerr.Biz("账号或密码错误，今日还有 " + strconv.Itoa(remaining) + " 次机会")
		}

		migratedAdmin, migrateErr := s.migrateLegacyAdmin(ctx, legacyUser, req.Username, req.Password)
		if migrateErr != nil {
			return "", "", nil, migrateErr
		}
		admin = &migratedAdmin
	}

	if admin.Password != md5Hex(req.Password) {
		remaining, countErr := s.handleLoginFail(ctx, failCountKey, lockKey)
		if countErr != nil {
			return "", "", nil, countErr
		}
		return "", "", nil, bizerr.Biz("账号或密码错误，今日还有 " + strconv.Itoa(remaining) + " 次机会")
	}

	user, err := s.GetByID(ctx, admin.UserID)
	if err != nil {
		return "", "", nil, err
	}
	if user == nil {
		return "", "", nil, bizerr.Biz("管理员关联用户不存在或已失效")
	}

	if err := s.redis.Del(ctx, failCountKey).Err(); err != nil {
		return "", "", nil, bizerr.InternalWrap("清理管理员登录失败计数失败", err)
	}

	user.Power = resolveAdminPower(admin.Power)
	rootUser, err := s.getRootUser(ctx, user)
	if err != nil {
		return "", "", nil, err
	}

	token, refreshToken, err := s.jwtHelper.GenerateTokenPair(s.buildTokenUser(user, rootUser))
	if err != nil {
		return "", "", nil, bizerr.InternalWrap("生成管理员登录令牌失败", err)
	}
	return token, refreshToken, s.sanitizeUser(user), nil
}

func (s *Service) handleLoginFail(ctx context.Context, failCountKey, lockKey string) (int, error) {
	if s.redis == nil {
		return 0, bizerr.Internal("redis client not initialized")
	}

	count, err := s.redis.Incr(ctx, failCountKey).Result()
	if err != nil {
		return 0, bizerr.InternalWrap("增加管理员登录失败次数失败", err)
	}
	if count == 1 {
		if err := s.redis.Expire(ctx, failCountKey, 24*time.Hour).Err(); err != nil {
			return 0, bizerr.InternalWrap("设置管理员登录失败次数过期时间失败", err)
		}
	}
	if count >= 10 {
		if err := s.redis.Set(ctx, lockKey, "locked", 24*time.Hour).Err(); err != nil {
			return 0, bizerr.InternalWrap("锁定管理员账号失败", err)
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
		return nil, bizerr.InternalWrap("加密旧管理员密码失败", err)
	}

	user, err := s.repo.FindLegacyAdminUser(ctx, stuNum, encPwd, 8)
	if err != nil {
		return nil, bizerr.InternalWrap("查询旧管理员账号失败", err)
	}
	return user, nil
}

func (s *Service) migrateLegacyAdmin(ctx context.Context, user *User, username, rawPwd string) (Admin, error) {
	admin := Admin{
		UserID:   user.ID,
		Username: username,
		Password: md5Hex(rawPwd),
		Power:    resolveAdminPower(user.Power),
	}
	if err := s.repo.CreateAdmin(ctx, &admin); err != nil {
		return Admin{}, bizerr.InternalWrap("迁移旧管理员账号失败", err)
	}
	return admin, nil
}

func resolveAdminPower(power int) int {
	if power <= 0 {
		return 2
	}
	return power
}

func md5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func (s *Service) adminSecondaryPassword() string {
	if s.cfg == nil {
		return defaultAdminSecondaryPassword
	}

	pwd := strings.TrimSpace(s.cfg.Admin.SecondaryPassword)
	if pwd == "" || pwd == "replace-me" {
		return defaultAdminSecondaryPassword
	}
	return pwd
}
