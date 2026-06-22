package user

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/integration/wxutil"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/adminjwt"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/rediskey"
)

var (
	ErrUserNotFound   = bizerr.NotFound("用户不存在")
	ErrRTKNotExisted  = bizerr.NotFound("refresh_token 不存在, 或已过期")
	ErrRTKUsed        = bizerr.Biz("refresh_token 已使用")
	ErrIdentityDenied = bizerr.Forbidden("身份切换无权限")
)

var ErrAuthenticationClearFailed = bizerr.Biz("清除认证失败")

type Service struct {
	repo       *Repository
	redis      *redis.Client
	cfg        *config.Config
	logger     *zap.Logger
	jwtHelper  *jwtutil.Helper
	adminJWT   *adminjwt.Helper
	wxClient   *wxutil.Client
	producer   EventProducer
	identityMu sync.Mutex
}

func NewService(db *gorm.DB, mongoDB *mongo.Database, rds *redis.Client, cfg *config.Config, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}

	s := &Service{
		repo:   NewRepository(db, mongoDB),
		redis:  rds,
		cfg:    cfg,
		logger: logger,
	}

	if cfg != nil {
		s.jwtHelper = jwtutil.NewHelper(cfg.JWT, rds)
		s.adminJWT = adminjwt.NewHelper(cfg.AdminJWT, rds)
		s.wxClient = wxutil.NewClient(cfg.WX, logger)
	}

	return s
}

func (s *Service) SetProducer(producer EventProducer) {
	s.producer = producer
}

func (s *Service) GetByID(ctx context.Context, id int64) (*User, error) {
	user, err := s.repo.FindUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.normalizedUser(user), nil
}

func (s *Service) GetByOpenID(ctx context.Context, openID string) (*User, error) {
	user, err := s.repo.FindUserByOpenID(ctx, openID)
	if err != nil {
		return nil, err
	}
	return s.normalizedUser(user), nil
}

func (s *Service) GetUserProfile(ctx context.Context, targetUserID int64) (*UserProfile, error) {
	user, err := s.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return &UserProfile{
		Avatar:    user.Avatar,
		Nickname:  user.Nickname,
		Gender:    user.Gender,
		StuCla:    user.StuCla,
		Signature: user.Signature,
	}, nil
}

func (s *Service) Edit(ctx context.Context, userID int64, req UserEditReq) (*User, error) {
	req = normalizeUserEditReq(req)
	if req.Nickname == "" && req.Avatar == "" && req.Gender == "" && req.Signature == "" {
		return s.sanitizeUserByID(ctx, userID)
	}

	if err := s.repo.UpdateUserProfile(ctx, userID, req); err != nil {
		return nil, err
	}

	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	s.publishUserUpdate(ctx, userID, buildUserUpdateMsg(
		userID,
		user.AccountType,
		req.Nickname,
		req.Avatar,
		req.Gender,
		req.Signature,
	), "edit_profile")
	return s.sanitizeUser(user), nil
}

func (s *Service) WechatLogin(ctx context.Context, code string) (string, string, *User, *User, bool, error) {
	if s.wxClient == nil || s.jwtHelper == nil {
		return "", "", nil, nil, false, errors.New("user service dependencies not initialized")
	}

	resp, err := s.wxClient.Jscode2Session(ctx, code)
	if err != nil {
		return "", "", nil, nil, false, fmt.Errorf("wx jscode2session: %w", err)
	}

	user, err := s.GetByOpenID(ctx, resp.OpenID)
	if err != nil {
		return "", "", nil, nil, false, err
	}

	isNew := false
	if user == nil {
		isNew = true
		user = &User{
			OpenID:      resp.OpenID,
			Nickname:    randomHumorousID(),
			Avatar:      s.pickDefaultAvatar(),
			Power:       0,
			AccountType: accountTypeBase,
			Tag:         "student",
			Gender:      "保密",
			CreatedBy:   0,
			UpdatedBy:   0,
		}

		if err := s.repo.CreateUser(ctx, user); err != nil {
			return "", "", nil, nil, false, err
		}
		if err := s.repo.InitializeRootIdentity(ctx, user.ID); err != nil {
			return "", "", nil, nil, false, err
		}

		lastSwitchID := user.ID
		user.RootUserID = user.ID
		user.LastSwitchID = &lastSwitchID
	}

	rootUser, err := s.getRootUser(ctx, user)
	if err != nil {
		return "", "", nil, nil, false, err
	}
	if err := s.maybeGrantProvisionalCert(ctx, rootUser); err != nil {
		return "", "", nil, nil, false, fmt.Errorf("grant provisional cert: %w", err)
	}
	activeIdentity, err := s.resolveActiveIdentity(ctx, rootUser)
	if err != nil {
		return "", "", nil, nil, false, err
	}

	token, refreshToken, err := s.jwtHelper.GenerateTokenPair(s.buildTokenUser(activeIdentity, rootUser))
	if err != nil {
		return "", "", nil, nil, false, fmt.Errorf("generate token pair: %w", err)
	}
	return token, refreshToken, s.sanitizeUser(rootUser), s.sanitizeUser(activeIdentity), isNew, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, string, *User, error) {
	if s.jwtHelper == nil {
		return "", "", nil, errors.New("jwt helper not initialized")
	}
	if s.redis == nil {
		return "", "", nil, errors.New("refresh token store not initialized")
	}

	refreshKey := rediskey.RefreshToken(sha1Hex(refreshToken))
	status, err := s.redis.Get(ctx, refreshKey).Result()
	if errors.Is(err, redis.Nil) {
		return "", "", nil, ErrRTKNotExisted
	}
	if err != nil {
		return "", "", nil, fmt.Errorf("get refresh token status: %w", err)
	}
	if status == rediskey.TokenStatusUsed {
		return "", "", nil, ErrRTKUsed
	}

	refreshTTL := 48 * time.Hour
	if s.cfg != nil && s.cfg.JWT.RefreshTokenMinutes > 0 {
		refreshTTL = time.Duration(s.cfg.JWT.RefreshTokenMinutes) * time.Minute
	}
	if err := s.redis.Set(ctx, refreshKey, rediskey.TokenStatusUsed, refreshTTL).Err(); err != nil {
		return "", "", nil, fmt.Errorf("mark refresh token used: %w", err)
	}

	claims, err := s.jwtHelper.Parse(refreshToken)
	if err != nil {
		return "", "", nil, fmt.Errorf("parse refresh token: %w", err)
	}

	user, err := s.GetByID(ctx, claims.UserID)
	if err != nil {
		return "", "", nil, err
	}
	if user == nil {
		return "", "", nil, ErrUserNotFound
	}

	rootUser, err := s.getRootUser(ctx, user)
	if err != nil {
		return "", "", nil, err
	}

	token, newRefreshToken, err := s.jwtHelper.GenerateTokenPair(s.buildTokenUser(user, rootUser))
	if err != nil {
		return "", "", nil, err
	}
	return token, newRefreshToken, s.sanitizeUser(user), nil
}

func (s *Service) pickDefaultAvatar() string {
	if s.cfg == nil {
		return ""
	}

	avatars := strings.Split(s.cfg.Custom.DefaultAvatar, ",")
	clean := make([]string, 0, len(avatars))
	for _, avatar := range avatars {
		avatar = strings.TrimSpace(avatar)
		if avatar != "" {
			clean = append(clean, avatar)
		}
	}
	if len(clean) == 0 {
		return ""
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return avatarURL(s.cfg.COS.BaseCDN, clean[r.Intn(len(clean))])
}

// avatarURL 把配置里的裸 md5 key 拼成可直接加载的 CDN 完整 URL；
// 已是 http(s) 链接或未配置 base_cdn 时原样返回，避免破坏存量完整 URL。
func avatarURL(baseCDN, avatar string) string {
	avatar = strings.TrimSpace(avatar)
	if avatar == "" {
		return ""
	}
	if strings.HasPrefix(avatar, "http://") || strings.HasPrefix(avatar, "https://") {
		return avatar
	}
	baseCDN = strings.TrimSpace(baseCDN)
	if baseCDN == "" {
		return avatar
	}
	if !strings.HasSuffix(baseCDN, "/") {
		baseCDN += "/"
	}
	return baseCDN + avatar
}

func sha1Hex(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func mapAccountType(accountType string) int {
	switch strings.TrimSpace(accountType) {
	case accountTypeAnonymous:
		return 3
	default:
		return 1
	}
}

func rootUserID(user *User) int64 {
	if user == nil {
		return 0
	}
	if user.RootUserID > 0 {
		return user.RootUserID
	}
	return user.ID
}
