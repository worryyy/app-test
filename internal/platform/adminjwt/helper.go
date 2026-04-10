package adminjwt

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/rediskey"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/snowflake"
)

const (
	tokenSubjectAccess  = "access"
	tokenSubjectRefresh = "refresh"
)

var (
	ErrTokenEmpty      = errors.New("authorization not found")
	ErrTokenNotExisted = errors.New("token not existed or expired")
	ErrTokenInvalid    = errors.New("token invalid")
	ErrTokenUsed       = errors.New("token already used")
	ErrSessionInvalid  = errors.New("admin session invalid")
	ErrUserInvalid     = errors.New("token user invalid")
	ErrUserNotExisted  = errors.New("user not existed")
)

type Claims struct {
	AdminID   int64  `json:"admin_id"`
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Power     int    `json:"power"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

type TokenUser struct {
	AdminID   int64
	UserID    int64
	Username  string
	Power     int
	SessionID string
}

type Helper struct {
	cfg config.AdminJWTConfig
	rds *redis.Client
}

func NewHelper(cfg config.AdminJWTConfig, rds *redis.Client) *Helper {
	return &Helper{cfg: cfg, rds: rds}
}

func (h *Helper) GenerateTokenPair(u *TokenUser) (token, refreshToken string, err error) {
	if u == nil {
		return "", "", ErrUserNotExisted
	}
	if u.AdminID <= 0 || u.UserID <= 0 || strings.TrimSpace(u.Username) == "" {
		return "", "", ErrUserInvalid
	}

	sessionID := strings.TrimSpace(u.SessionID)
	if sessionID == "" {
		sessionID = snowflake.Generate().String()
	}

	now := time.Now()
	claims := &Claims{
		AdminID:   u.AdminID,
		UserID:    u.UserID,
		Username:  u.Username,
		Power:     u.Power,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    h.cfg.Issue,
			Subject:   tokenSubjectAccess,
			ExpiresAt: jwt.NewNumericDate(now.Add(h.accessTokenTTL())),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        snowflake.Generate().String(),
		},
	}

	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.cfg.Secret))
	if err != nil {
		return "", "", fmt.Errorf("sign token: %w", err)
	}

	refreshClaims := *claims
	refreshClaims.Subject = tokenSubjectRefresh
	refreshClaims.ExpiresAt = jwt.NewNumericDate(now.Add(h.refreshTokenTTL()))
	refreshClaims.ID = snowflake.Generate().String()

	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, &refreshClaims).SignedString([]byte(h.cfg.Secret))
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	if h.rds != nil {
		ctx := context.Background()
		if err := h.rds.Set(ctx, rediskey.AdminToken(sha1Hex(token)), rediskey.TokenStatusOK, h.accessTokenTTL()).Err(); err != nil {
			return "", "", fmt.Errorf("save admin token status: %w", err)
		}
		if err := h.rds.Set(ctx, rediskey.AdminRefreshToken(sha1Hex(refreshToken)), rediskey.TokenStatusOK, h.refreshTokenTTL()).Err(); err != nil {
			return "", "", fmt.Errorf("save admin refresh token status: %w", err)
		}
		if err := h.rds.Set(ctx, rediskey.AdminSession(u.AdminID), sessionID, h.refreshTokenTTL()).Err(); err != nil {
			return "", "", fmt.Errorf("save admin session: %w", err)
		}
	}

	return token, refreshToken, nil
}

func (h *Helper) Parse(token string) (*Claims, error) {
	claims := &Claims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(h.cfg.Secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}
	if !parsedToken.Valid {
		return nil, ErrTokenInvalid
	}
	if claims.AdminID <= 0 || claims.UserID <= 0 || strings.TrimSpace(claims.Username) == "" || strings.TrimSpace(claims.SessionID) == "" {
		return nil, ErrTokenInvalid
	}
	if strings.TrimSpace(claims.Issuer) != strings.TrimSpace(h.cfg.Issue) {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

func (h *Helper) ParseAndVerifyAccess(ctx context.Context, token string, rds *redis.Client) (*Claims, error) {
	return h.parseAndVerify(ctx, token, rediskey.AdminToken, tokenSubjectAccess, false, rds)
}

func (h *Helper) ParseAndVerifyRefresh(ctx context.Context, token string, rds *redis.Client) (*Claims, error) {
	return h.parseAndVerify(ctx, token, rediskey.AdminRefreshToken, tokenSubjectRefresh, true, rds)
}

func (h *Helper) ConsumeRefreshToken(ctx context.Context, token string, rds *redis.Client) error {
	if token == "" {
		return ErrTokenEmpty
	}

	redisClient := h.redisClient(rds)
	if redisClient == nil {
		return nil
	}

	key := rediskey.AdminRefreshToken(sha1Hex(token))
	status, err := redisClient.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return ErrTokenNotExisted
	}
	if err != nil {
		return fmt.Errorf("get refresh token status: %w", err)
	}
	if status == rediskey.TokenStatusUsed {
		return ErrTokenUsed
	}
	if err := redisClient.Set(ctx, key, rediskey.TokenStatusUsed, h.refreshTokenTTL()).Err(); err != nil {
		return fmt.Errorf("mark admin refresh token used: %w", err)
	}
	return nil
}

func (h *Helper) Logout(ctx context.Context, adminID int64, rds *redis.Client) error {
	if adminID <= 0 {
		return ErrUserInvalid
	}

	redisClient := h.redisClient(rds)
	if redisClient == nil {
		return nil
	}

	if err := redisClient.Del(ctx, rediskey.AdminSession(adminID)).Err(); err != nil {
		return fmt.Errorf("delete admin session: %w", err)
	}
	return nil
}

func (h *Helper) parseAndVerify(
	ctx context.Context,
	token string,
	keyBuilder func(string) string,
	subject string,
	checkUsed bool,
	rds *redis.Client,
) (*Claims, error) {
	if token == "" {
		return nil, ErrTokenEmpty
	}
	if len(strings.Split(token, ".")) != 3 {
		return nil, ErrTokenInvalid
	}

	redisClient := h.redisClient(rds)
	if redisClient != nil {
		key := keyBuilder(sha1Hex(token))
		status, err := redisClient.Get(ctx, key).Result()
		if errors.Is(err, redis.Nil) {
			return nil, ErrTokenNotExisted
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrTokenNotExisted, err)
		}
		if checkUsed && status == rediskey.TokenStatusUsed {
			return nil, ErrTokenUsed
		}
	}

	claims, err := h.Parse(token)
	if err != nil {
		return nil, err
	}
	if claims.Subject != subject {
		return nil, ErrTokenInvalid
	}

	if redisClient != nil {
		sessionID, err := redisClient.Get(ctx, rediskey.AdminSession(claims.AdminID)).Result()
		if errors.Is(err, redis.Nil) {
			return nil, ErrSessionInvalid
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrSessionInvalid, err)
		}
		if sessionID != claims.SessionID {
			return nil, ErrSessionInvalid
		}
	}
	return claims, nil
}

func (h *Helper) accessTokenTTL() time.Duration {
	if h.cfg.TokenMinutes <= 0 {
		return 60 * time.Minute
	}
	return time.Duration(h.cfg.TokenMinutes) * time.Minute
}

func (h *Helper) refreshTokenTTL() time.Duration {
	if h.cfg.RefreshTokenMinutes <= 0 {
		return 48 * time.Hour
	}
	return time.Duration(h.cfg.RefreshTokenMinutes) * time.Minute
}

func (h *Helper) redisClient(rds *redis.Client) *redis.Client {
	if rds != nil {
		return rds
	}
	return h.rds
}

func sha1Hex(s string) string {
	hash := sha1.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}
