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
	ErrRTKNotExisted  = errors.New("refresh token 不存在,或已过期")
	ErrRTKUsed        = errors.New("refresh token 已使用")
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
	return &u, nil
}

func (s *Service) GetByOpenID(ctx context.Context, openID string) (*User, error) {
	var u User
	err := s.db.WithContext(ctx).Where("openId = ?", openID).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by openid %s: %w", openID, err)
	}
	return &u, nil
}

func (s *Service) Edit(ctx context.Context, userID int64, req *User) error {
	if req == nil {
		return result.ErrParam
	}

	updates := map[string]interface{}{}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Tag != "" {
		updates["tag"] = req.Tag
	}
	if req.Gender != "" {
		updates["gender"] = req.Gender
	}
	if req.Signature != "" {
		updates["signature"] = req.Signature
	}
	if len(updates) == 0 {
		return nil
	}

	if err := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return fmt.Errorf("edit user %d: %w", userID, err)
	}

	if s.producer != nil {
		accountType := 0
		if req.AccountType != "" {
			accountType = mapAccountType(req.AccountType)
		}
		msg := mq.TopicUserUpdateMsg{
			UserID:      strconv.FormatInt(userID, 10),
			NickName:    req.Nickname,
			Avatar:      req.Avatar,
			AccountType: accountType,
		}
		if err := s.producer.SendUpdateTopicUser(ctx, msg); err != nil {
			s.logger.Warn("send topic user update mq failed", zap.Error(err), zap.Int64("userID", userID))
		}
		commentMsg := mq.CommentUserUpdateMsg{
			UserID:      msg.UserID,
			NickName:    msg.NickName,
			Avatar:      msg.Avatar,
			AccountType: msg.AccountType,
		}
		if err := s.producer.SendUpdateCommentUser(ctx, commentMsg); err != nil {
			s.logger.Warn("send comment user update mq failed", zap.Error(err), zap.Int64("userID", userID))
		}
	}
	return nil
}

func (s *Service) WechatLogin(ctx context.Context, code string) (string, string, *User, error) {
	if s.wxClient == nil || s.jwtHelper == nil {
		return "", "", nil, errors.New("user service dependencies not initialized")
	}
	resp, err := s.wxClient.Jscode2Session(ctx, code)
	if err != nil {
		return "", "", nil, fmt.Errorf("wx jscode2session: %w", err)
	}

	u, err := s.GetByOpenID(ctx, resp.OpenID)
	if err != nil {
		return "", "", nil, err
	}

	if u == nil {
		u = &User{
			OpenID:      resp.OpenID,
			Nickname:    s.randomNickname(),
			Avatar:      s.pickDefaultAvatar(),
			Power:       0,
			AccountType: "base",
			Tag:         "student",
			Gender:      "保密",
		}

		if err := s.db.WithContext(ctx).Create(u).Error; err != nil {
			return "", "", nil, fmt.Errorf("create user: %w", err)
		}
		u.RootUserID = u.ID
		if err := s.db.WithContext(ctx).Model(u).Update("rootUserId", u.ID).Error; err != nil {
			return "", "", nil, fmt.Errorf("update root user id: %w", err)
		}
	}

	token, refreshToken, err := s.jwtHelper.GenerateTokenPair(&jwtutil.TokenUser{
		ID:          u.ID,
		OpenID:      u.OpenID,
		Power:       u.Power,
		AccountType: u.AccountType,
		RootUserID:  u.RootUserID,
	})
	if err != nil {
		return "", "", nil, fmt.Errorf("generate token pair: %w", err)
	}

	if s.redis != nil {
		today := time.Now().Format("20060102")
		if err := s.redis.PFAdd(ctx, rediskey.ActiveDay(today), u.ID).Err(); err != nil {
			s.logger.Warn("record active user failed", zap.Error(err), zap.Int64("userID", u.ID))
		}
	}

	u.StuPwd = ""
	return token, refreshToken, u, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	if s.jwtHelper == nil {
		return "", "", errors.New("jwt helper not initialized")
	}

	refreshKey := rediskey.RefreshToken(sha1Hex(refreshToken))
	status, err := s.redis.Get(ctx, refreshKey).Result()
	if err != nil {
		return "", "", fmt.Errorf("refresh token: %w", ErrRTKNotExisted)
	}
	if status == rediskey.TokenStatusUsed {
		return "", "", ErrRTKUsed
	}

	if err := s.redis.Set(ctx, refreshKey, rediskey.TokenStatusUsed, 3*24*time.Hour).Err(); err != nil {
		return "", "", fmt.Errorf("mark refresh token used: %w", err)
	}

	claims, err := s.jwtHelper.Parse(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("parse refresh token: %w", err)
	}

	u, err := s.GetByID(ctx, claims.UserID)
	if err != nil {
		return "", "", err
	}
	if u == nil {
		return "", "", ErrUserNotFound
	}

	token, newRefreshToken, err := s.jwtHelper.GenerateTokenPair(&jwtutil.TokenUser{
		ID:          u.ID,
		OpenID:      u.OpenID,
		Power:       u.Power,
		AccountType: u.AccountType,
		RootUserID:  u.RootUserID,
	})
	if err != nil {
		return "", "", err
	}
	return token, newRefreshToken, nil
}

func (s *Service) randomNickname() string {
	prefix := "用户"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%s%06d", prefix, r.Intn(1000000))
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

func toStringID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func sha1Hex(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func mapAccountType(accountType string) int {
	switch accountType {
	case "official":
		return 2
	case "anonymous":
		return 3
	default:
		return 1
	}
}
