package user

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/rediskey"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/wxutil"
)

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrRTKNotExisted  = result.ErrRTKNotExisted
	ErrRTKUsed        = result.ErrRTKUsed
	ErrFollowSelf     = errors.New("不能关注自己")
	ErrIdentityDenied = errors.New("身份切换无权限")
)

type Service struct {
	db        *gorm.DB
	mongoDB   *mongo.Database
	redis     *redis.Client
	cfg       *config.Config
	logger    *zap.Logger
	jwtHelper *jwtutil.Helper
	wxClient  *wxutil.Client
	jwClient  *JWClient
	producer  *mq.Producer
}

func NewService(db *gorm.DB, mongoDB *mongo.Database, rds *redis.Client, cfg *config.Config, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}

	s := &Service{
		db:      db,
		mongoDB: mongoDB,
		redis:   rds,
		cfg:     cfg,
		logger:  logger,
	}

	if cfg != nil {
		s.jwtHelper = jwtutil.NewHelper(cfg.JWT, rds)
		s.wxClient = wxutil.NewClient(cfg.WX, logger)
		s.jwClient = NewJWClient(cfg, logger)
	}

	return s
}

func (s *Service) SetJWTHelper(helper *jwtutil.Helper) {
	s.jwtHelper = helper
}

func (s *Service) SetWXClient(client *wxutil.Client) {
	s.wxClient = client
}

func (s *Service) SetProducer(producer *mq.Producer) {
	s.producer = producer
}

func (s *Service) GetByID(ctx context.Context, id int64) (*User, error) {
	var u User
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id %d: %w", id, err)
	}
	s.ensureUserDefaults(&u)
	return &u, nil
}

func (s *Service) GetByOpenID(ctx context.Context, openID string) (*User, error) {
	var u User
	err := s.db.WithContext(ctx).Where("open_id = ?", openID).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by openid %s: %w", openID, err)
	}
	s.ensureUserDefaults(&u)
	return &u, nil
}

func (s *Service) GetUserProfile(ctx context.Context, targetUserID int64) (*UserProfile, error) {
	user, err := s.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, result.ErrNotExisted
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
	updates := map[string]interface{}{}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Gender != "" {
		updates["gender"] = req.Gender
	}
	if req.Signature != "" {
		updates["signature"] = req.Signature
	}
	if len(updates) == 0 {
		return s.sanitizeUserByID(ctx, userID)
	}

	if err := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("edit user %d: %w", userID, err)
	}

	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if s.producer != nil {
		msg := mq.TopicUserUpdateMsg{
			UserID:      strconv.FormatInt(userID, 10),
			NickName:    req.Nickname,
			Avatar:      req.Avatar,
			Gender:      req.Gender,
			Signature:   req.Signature,
			AccountType: mapAccountType(user.AccountType),
		}
		if err := s.producer.SendUpdateTopicUser(ctx, msg); err != nil {
			s.logger.Warn("send topic user update mq failed", zap.Error(err), zap.Int64("userID", userID))
		}
		commentMsg := mq.CommentUserUpdateMsg(msg)
		if err := s.producer.SendUpdateCommentUser(ctx, commentMsg); err != nil {
			s.logger.Warn("send comment user update mq failed", zap.Error(err), zap.Int64("userID", userID))
		}
	}
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

	u, err := s.GetByOpenID(ctx, resp.OpenID)
	if err != nil {
		return "", "", nil, nil, false, err
	}

	isNew := false
	if u == nil {
		isNew = true
		u = &User{
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

		if err := s.db.WithContext(ctx).Create(u).Error; err != nil {
			return "", "", nil, nil, false, fmt.Errorf("create user: %w", err)
		}
		u.RootUserID = u.ID
		u.LastSwitchID = &u.ID
		if err := s.db.WithContext(ctx).Model(u).Updates(map[string]interface{}{
			"root_user_id":   u.ID,
			"last_switch_id": u.ID,
			"account_type":  accountTypeBase,
		}).Error; err != nil {
			return "", "", nil, nil, false, fmt.Errorf("update root user id: %w", err)
		}
	}

	rootUser, err := s.getRootUser(ctx, u)
	if err != nil {
		return "", "", nil, nil, false, err
	}
	activeIdentity, err := s.resolveActiveIdentity(ctx, rootUser)
	if err != nil {
		return "", "", nil, nil, false, err
	}

	token, refreshToken, err := s.jwtHelper.GenerateTokenPair(s.buildTokenUser(activeIdentity, rootUser))
	if err != nil {
		return "", "", nil, nil, false, fmt.Errorf("generate token pair: %w", err)
	}

	if s.redis != nil {
		today := time.Now().Format("20060102")
		if err := s.redis.PFAdd(ctx, rediskey.ActiveDay(today), rootUser.ID).Err(); err != nil {
			s.logger.Warn("record active user failed", zap.Error(err), zap.Int64("userID", rootUser.ID))
		}
	}

	return token, refreshToken, s.sanitizeUser(rootUser), s.sanitizeUser(activeIdentity), isNew, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, string, *User, error) {
	if s.jwtHelper == nil {
		return "", "", nil, errors.New("jwt helper not initialized")
	}
	if s.redis == nil {
		return "", "", nil, fmt.Errorf("refresh token store not initialized")
	}

	refreshKey := rediskey.RefreshToken(sha1Hex(refreshToken))
	status, err := s.redis.Get(ctx, refreshKey).Result()
	if err != nil {
		return "", "", nil, fmt.Errorf("refresh token: %w", ErrRTKNotExisted)
	}
	if status == rediskey.TokenStatusUsed {
		return "", "", nil, ErrRTKUsed
	}

	refreshTTL := 48 * time.Hour
	if s.cfg != nil && s.cfg.JWT.RefreshTokenMinutes > 0 {
		refreshTTL = time.Duration(s.cfg.JWT.RefreshTokenMinutes) * time.Minute
	}
	if err := s.redis.Set(ctx, refreshKey, rediskey.TokenStatusUsed, refreshTTL).Err(); err != nil {
		return "", "", nil, fmt.Errorf("mark refresh token used: %w", result.ErrRTKError)
	}

	claims, err := s.jwtHelper.Parse(refreshToken)
	if err != nil {
		return "", "", nil, fmt.Errorf("parse refresh token: %w", err)
	}

	u, err := s.GetByID(ctx, claims.UserID)
	if err != nil {
		return "", "", nil, err
	}
	if u == nil {
		return "", "", nil, ErrUserNotFound
	}
	rootUser, err := s.getRootUser(ctx, u)
	if err != nil {
		return "", "", nil, err
	}

	token, newRefreshToken, err := s.jwtHelper.GenerateTokenPair(s.buildTokenUser(u, rootUser))
	if err != nil {
		return "", "", nil, err
	}
	return token, newRefreshToken, s.sanitizeUser(u), nil
}

func (s *Service) pickDefaultAvatar() string {
	if s.cfg == nil {
		return ""
	}
	avatars := strings.Split(s.cfg.Custom.DefaultAvatar, ",")
	clean := make([]string, 0, len(avatars))
	for _, a := range avatars {
		a = strings.TrimSpace(a)
		if a != "" {
			clean = append(clean, a)
		}
	}
	if len(clean) == 0 {
		return ""
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return clean[r.Intn(len(clean))]
}

func sha1Hex(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func mapAccountType(accountType string) int {
	switch accountType {
	case accountTypeOfficial:
		return 2
	case accountTypeAnonymous:
		return 3
	default:
		return 1
	}
}

func rootUserID(u *User) int64 {
	if u == nil {
		return 0
	}
	if u.RootUserID > 0 {
		return u.RootUserID
	}
	return u.ID
}
