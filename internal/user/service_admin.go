package user

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/adminjwt"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/encrypt"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/rediskey"
)

func (s *Service) AdminLogin(ctx context.Context, req *AdminLoginReq) (string, string, *User, error) {
	if req == nil {
		return "", "", nil, bizerr.Param(errMsgInvalidParam)
	}
	if s.adminJWT == nil {
		return "", "", nil, bizerr.Internal("admin jwt helper not initialized")
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

	matched, needsUpgrade := verifyAdminPassword(admin.Password, req.Password)
	if !matched {
		remaining, countErr := s.handleLoginFail(ctx, failCountKey, lockKey)
		if countErr != nil {
			return "", "", nil, countErr
		}
		return "", "", nil, bizerr.Biz("账号或密码错误，今日还有 " + strconv.Itoa(remaining) + " 次机会")
	}

	if needsUpgrade {
		s.tryUpgradeAdminPasswordHash(ctx, admin, req.Password)
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

	token, refreshToken, err := s.adminJWT.GenerateTokenPair(s.buildAdminAuthTokenUser(admin))
	if err != nil {
		return "", "", nil, bizerr.InternalWrap("生成管理员登录令牌失败", err)
	}
	return token, refreshToken, s.sanitizeUser(user), nil
}

func (s *Service) AdminRefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	if s.adminJWT == nil {
		return "", "", bizerr.Internal("admin jwt helper not initialized")
	}
	if s.redis == nil {
		return "", "", bizerr.Internal("redis client not initialized")
	}

	claims, err := s.adminJWT.ParseAndVerifyRefresh(ctx, refreshToken, s.redis)
	if err != nil {
		return "", "", mapAdminAuthError(err)
	}
	if err := s.adminJWT.ConsumeRefreshToken(ctx, refreshToken, s.redis); err != nil {
		return "", "", mapAdminAuthError(err)
	}

	admin, err := s.repo.FindAdminByID(ctx, claims.AdminID)
	if err != nil {
		return "", "", bizerr.InternalWrap("查询管理员失败", err)
	}
	if admin == nil || admin.UserID != claims.UserID {
		return "", "", bizerr.Forbidden("管理员权限不足")
	}

	user, err := s.GetByID(ctx, admin.UserID)
	if err != nil {
		return "", "", err
	}
	if user == nil {
		return "", "", ErrUserNotFound
	}

	tokenUser := s.buildAdminAuthTokenUser(admin)
	if tokenUser == nil {
		return "", "", bizerr.Internal("admin token user invalid")
	}
	tokenUser.SessionID = claims.SessionID

	token, newRefreshToken, err := s.adminJWT.GenerateTokenPair(tokenUser)
	if err != nil {
		return "", "", bizerr.InternalWrap("生成管理员刷新令牌失败", err)
	}
	return token, newRefreshToken, nil
}

func (s *Service) AdminLogout(ctx context.Context, claims *adminjwt.Claims) error {
	if claims == nil || claims.AdminID <= 0 {
		return bizerr.Unauthorized(errMsgUserNotLogin)
	}
	if s.adminJWT == nil {
		return bizerr.Internal("admin jwt helper not initialized")
	}
	return mapAdminAuthError(s.adminJWT.Logout(ctx, claims.AdminID, s.redis))
}

func (s *Service) AdminUserToken(ctx context.Context, claims *adminjwt.Claims) (string, string, error) {
	if claims == nil || claims.AdminID <= 0 || claims.UserID <= 0 {
		return "", "", bizerr.Unauthorized(errMsgUserNotLogin)
	}
	if s.jwtHelper == nil {
		return "", "", bizerr.Internal("jwt helper not initialized")
	}

	user, err := s.GetByID(ctx, claims.UserID)
	if err != nil {
		return "", "", err
	}
	if user == nil {
		return "", "", ErrUserNotFound
	}

	token, refreshToken, err := s.jwtHelper.GenerateTokenPair(s.buildAdminUserJWTUser(user))
	if err != nil {
		return "", "", bizerr.InternalWrap("generate admin user token failed", err)
	}
	return token, refreshToken, nil
}

func (s *Service) buildAdminUserJWTUser(user *User) *jwtutil.TokenUser {
	return s.buildTokenUser(user, user)
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
			return 0, bizerr.InternalWrap("锁定管理员帐号失败", err)
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
		return nil, bizerr.InternalWrap("查询旧管理员帐号失败", err)
	}
	return user, nil
}

func (s *Service) migrateLegacyAdmin(ctx context.Context, user *User, username, rawPwd string) (Admin, error) {
	hashedPassword, err := hashAdminPassword(rawPwd)
	if err != nil {
		return Admin{}, bizerr.InternalWrap("加密管理员密码失败", err)
	}

	admin := Admin{
		UserID:   user.ID,
		Username: username,
		Password: hashedPassword,
		Power:    resolveAdminPower(user.Power),
	}
	if err := s.repo.CreateAdmin(ctx, &admin); err != nil {
		return Admin{}, bizerr.InternalWrap("迁移旧管理员帐号失败", err)
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

func hashAdminPassword(raw string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func verifyAdminPassword(storedHash, raw string) (matched, needsUpgrade bool) {
	if strings.HasPrefix(storedHash, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(raw)) == nil, false
	}
	return storedHash == md5Hex(raw), true
}

func (s *Service) tryUpgradeAdminPasswordHash(ctx context.Context, admin *Admin, rawPassword string) {
	if admin == nil || admin.ID <= 0 {
		return
	}

	hashedPassword, err := hashAdminPassword(rawPassword)
	if err != nil {
		s.logger.Warn("hash admin password failed", zap.Error(err), zap.Int64("adminID", admin.ID))
		return
	}
	if err := s.repo.UpdateAdminPasswordHash(ctx, admin.ID, hashedPassword); err != nil {
		s.logger.Warn("upgrade admin password hash failed", zap.Error(err), zap.Int64("adminID", admin.ID))
		return
	}
	admin.Password = hashedPassword
}

func mapAdminAuthError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, adminjwt.ErrTokenNotExisted):
		return bizerr.NotFound("admin refresh_token 不存在, 或已过期")
	case errors.Is(err, adminjwt.ErrTokenUsed):
		return bizerr.Biz("admin refresh_token 已使用")
	case errors.Is(err, adminjwt.ErrSessionInvalid):
		return bizerr.Unauthorized("admin session 无效")
	case errors.Is(err, adminjwt.ErrTokenEmpty), errors.Is(err, adminjwt.ErrTokenInvalid), errors.Is(err, adminjwt.ErrUserInvalid):
		return bizerr.Unauthorized("admin token 无效")
	default:
		return bizerr.InternalWrap("admin auth failed", err)
	}
}
