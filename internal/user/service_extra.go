package user

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/encrypt"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) RandomNickname(accountType string) (string, error) {
	switch accountType {
	case accountTypeBase:
		return randomHumorousID(), nil
	case accountTypeAnonymous:
		return randomAnonymousID(), nil
	default:
		return "", result.ErrParam
	}
}

func (s *Service) PreAuthentication(ctx context.Context, userID int64, nickname, pwd string) error {
	if userID <= 0 || strings.TrimSpace(nickname) == "" {
		return result.ErrParam
	}
	if pwd != "zjb&bjz" {
		return result.NewBizError(result.CodeFail, "预认证密码错误")
	}

	res := s.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ? AND nickname = ?", userID, nickname).
		Update("stuIsCheck", true)
	if res.Error != nil {
		return fmt.Errorf("pre-authenticate user: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return result.NewBizError(result.CodeUnknownError, "预认证更新失败")
	}
	return nil
}

func (s *Service) Authenticate(
	ctx context.Context,
	userID int64,
	req AuthenticationReq,
) (*JWLoginData, error) {
	loginResp, err := s.checkJWLogin(ctx, req.SchoolID, req.Password)
	if err != nil {
		return nil, err
	}

	encPwd, err := s.encryptAES(req.Password)
	if err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"stuIsCheck": true,
		"stuNum":     req.SchoolID,
		"stuCla":     loginResp.Major,
		"stuName":    loginResp.Name,
		"stuPwd":     encPwd,
		"school":     req.School,
	}).Error; err != nil {
		return nil, fmt.Errorf("authenticate user: %w", err)
	}
	return loginResp, nil
}

func (s *Service) ReAuthentication(
	ctx context.Context,
	userID int64,
	req AuthenticationReq,
) (*JWLoginData, error) {
	return s.Authenticate(ctx, userID, req)
}

func (s *Service) DelAuthentication(ctx context.Context, userID int64) error {
	if err := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"stuNum":     "",
		"stuPwd":     "",
		"stuName":    "",
		"stuCla":     "",
		"stuIsCheck": false,
	}).Error; err != nil {
		return fmt.Errorf("delete authentication: %w", err)
	}
	return nil
}

func (s *Service) CheckLogin(ctx context.Context, req CheckLoginReq) (*JWLoginData, error) {
	return s.checkJWLogin(ctx, req.SchoolID, req.Password)
}

func (s *Service) GetCourseByWeeks(ctx context.Context, req UserCourseReq) (*JWCommonResp, error) {
	if s.jwClient == nil {
		return nil, fmt.Errorf("jw client not initialized")
	}
	return s.jwClient.GetCourseByWeeks(ctx, req.StartDate, req.Week, JWGetCourseReq{
		Term:     req.Term,
		SchoolID: req.SchoolID,
		Password: req.Password,
	})
}

func (s *Service) GetExam(ctx context.Context, req ExamReq) (*JWCommonResp, error) {
	if s.jwClient == nil {
		return nil, fmt.Errorf("jw client not initialized")
	}
	return s.jwClient.GetExam(ctx, JWGetExamReq{
		SchoolID: req.SchoolID,
		Password: req.Password,
		XNXQID:   req.XNXQID,
	})
}

func (s *Service) GetExamScore(ctx context.Context, req ExamScoreReq) (*JWCommonResp, error) {
	if s.jwClient == nil {
		return nil, fmt.Errorf("jw client not initialized")
	}
	return s.jwClient.GetExamScore(ctx, JWGetExamScoreReq{
		SchoolID: req.SchoolID,
		Password: req.Password,
		SS:       req.SS,
	})
}

func (s *Service) OfficialLogin(ctx context.Context, loginAccount, loginPassword string) (string, string, *User, error) {
	encPwd, err := s.encryptAES(loginPassword)
	if err != nil {
		return "", "", nil, err
	}

	var user User
	err = s.db.WithContext(ctx).
		Where("stuNum = ? AND stuPwd = ?", loginAccount, encPwd).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", nil, result.NewBizError(result.CodeFail, "登录账号或密码错误")
	}
	if err != nil {
		return "", "", nil, fmt.Errorf("query official user: %w", err)
	}
	if !strings.HasPrefix(user.OpenID, "official:") {
		return "", "", nil, result.NewBizError(result.CodeFail, "该账号不是官方账号")
	}

	rootUser, err := s.getRootUser(ctx, &user)
	if err != nil {
		return "", "", nil, err
	}
	token, refreshToken, err := s.jwtHelper.GenerateTokenPair(s.buildTokenUser(&user, rootUser))
	if err != nil {
		return "", "", nil, result.NewBizError(result.CodeFail, "登录失败，请重试")
	}
	return token, refreshToken, s.sanitizeUser(&user), nil
}

func (s *Service) SubmitOfficialCertification(ctx context.Context, req OfficialCertReq) (*OfficialCertification, error) {
	exists, err := s.mongoDB.Collection("campus_official_certification").CountDocuments(
		ctx,
		map[string]interface{}{"loginAccount": req.LoginAccount},
	)
	if err != nil {
		return nil, fmt.Errorf("check official certification account: %w", err)
	}
	if exists > 0 {
		return nil, result.NewBizError(result.CodeFail, "该登录账号已被申请，请使用其他账号")
	}

	var existingUser User
	err = s.db.WithContext(ctx).Where("stuNum = ?", req.LoginAccount).First(&existingUser).Error
	if err == nil {
		return nil, result.NewBizError(result.CodeFail, "该登录账号已被使用，请使用其他账号")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check existing official login account: %w", err)
	}

	encryptedPassword, err := s.encryptAES(req.LoginPassword)
	if err != nil {
		return nil, result.NewBizError(result.CodeFail, "提交失败，密码加密异常")
	}

	now := time.Now()
	doc := &OfficialCertification{
		AvatarURL:         req.AvatarURL,
		FullName:          req.FullName,
		ShortName:         req.ShortName,
		Nature:            req.Nature,
		Introduction:      req.Introduction,
		ResponsiblePerson: req.ResponsiblePerson,
		WechatAccount:     req.WechatAccount,
		LoginAccount:      req.LoginAccount,
		LoginPassword:     encryptedPassword,
		Status:            certificationStatusPending,
		RejectReason:      "",
		ReviewedBy:        0,
		ReviewedAt:        nil,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	res, err := s.mongoDB.Collection("campus_official_certification").InsertOne(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("insert official certification: %w", err)
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		doc.ID = oid
	}
	return doc, nil
}

func (s *Service) GenerateUnlimitedWXACode(ctx context.Context, scene, page string) (string, error) {
	if s.wxClient == nil {
		return "", errors.New("wx client not initialized")
	}
	data, err := s.wxClient.UnlimitedWXACode(ctx, scene, page)
	if err != nil {
		return "", fmt.Errorf("generate unlimited wxa code: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (s *Service) checkJWLogin(ctx context.Context, schoolID, password string) (*JWLoginData, error) {
	if s.jwClient == nil {
		return nil, fmt.Errorf("jw client not initialized")
	}
	resp, err := s.jwClient.CheckLogin(ctx, schoolID, password)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, result.NewBizError(result.CodeFail, "登陆失败")
	}
	if resp.Code != 200 {
		return nil, result.NewBizError(result.CodeFail, resp.Message)
	}

	dataMap, err := toJWLoginData(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("decode jw login response: %w", err)
	}
	return dataMap, nil
}

func (s *Service) encryptAES(raw string) (string, error) {
	if s.cfg == nil || strings.TrimSpace(s.cfg.Encryption.Key) == "" {
		return raw, nil
	}
	encrypted, err := encrypt.AESEncrypt(raw, s.cfg.Encryption.Key)
	if err != nil {
		return "", fmt.Errorf("encrypt password: %w", err)
	}
	return encrypted, nil
}

func toJWLoginData(data interface{}) (*JWLoginData, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var out JWLoginData
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
