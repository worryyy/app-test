package jwtutil

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

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/snowflake"
)

var (
	ErrTokenEmpty      = errors.New("authorization 找不到")
	ErrTokenNotExisted = errors.New("token 不存在,或已过期")
	ErrTokenInvalid    = errors.New("token invalid")
	ErrUserNotExisted  = errors.New("user not existed")
)

type Claims struct {
	UserID      int64  `json:"user_id"`
	OpenID      string `json:"open_id"`
	Power       int    `json:"power"`
	AccountType string `json:"account_type"`
	RootUserID  int64  `json:"root_user_id"`
	jwt.RegisteredClaims
}

type Helper struct {
	cfg config.JWTConfig
	rds *redis.Client
}

type TokenUser struct {
	ID          int64
	OpenID      string
	Power       int
	AccountType string
	RootUserID  int64
}

func NewHelper(cfg config.JWTConfig, rds *redis.Client) *Helper {
	return &Helper{cfg: cfg, rds: rds}
}

func (h *Helper) GenerateTokenPair(u *TokenUser) (token, refreshToken string, err error) {
	if u == nil {
		return "", "", ErrUserNotExisted
	}

	now := time.Now()
	rootUserID := u.RootUserID
	if rootUserID == 0 {
		rootUserID = u.ID
	}

	claims := &Claims{
		UserID:      u.ID,
		OpenID:      u.OpenID,
		Power:       u.Power,
		AccountType: u.AccountType,
		RootUserID:  rootUserID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    h.cfg.Issue,
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(h.cfg.TokenMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        snowflake.Generate().String(),
		},
	}

	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.cfg.Secret))
	if err != nil {
		return "", "", fmt.Errorf("sign token: %w", err)
	}

	refreshClaims := *claims
	refreshClaims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Duration(h.cfg.RefreshTokenMinutes) * time.Minute))
	refreshClaims.ID = snowflake.Generate().String()

	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, &refreshClaims).SignedString([]byte(h.cfg.Secret))
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	if h.rds != nil {
		ctx := context.Background()
		tokenTTL := time.Duration(h.cfg.TokenMinutes) * time.Minute
		refreshTTL := time.Duration(h.cfg.RefreshTokenMinutes) * time.Minute

		if err := h.rds.Set(ctx, rediskey.Token(sha1Hex(token)), rediskey.TokenStatusOK, tokenTTL).Err(); err != nil {
			return "", "", fmt.Errorf("save token status: %w", err)
		}
		if err := h.rds.Set(ctx, rediskey.RefreshToken(sha1Hex(refreshToken)), rediskey.TokenStatusOK, refreshTTL).Err(); err != nil {
			return "", "", fmt.Errorf("save refresh token status: %w", err)
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
	return claims, nil
}

func (h *Helper) ParseAndVerify(ctx context.Context, token string, rds *redis.Client) (*Claims, error) {
	if token == "" {
		return nil, ErrTokenEmpty
	}
	if len(strings.Split(token, ".")) != 3 {
		return nil, ErrTokenInvalid
	}

	redisClient := rds
	if redisClient == nil {
		redisClient = h.rds
	}
	if redisClient != nil {
		key := rediskey.Token(sha1Hex(token))
		if err := redisClient.Get(ctx, key).Err(); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrTokenNotExisted, err)
		}
	}

	claims, err := h.Parse(token)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func sha1Hex(s string) string {
	hash := sha1.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}
