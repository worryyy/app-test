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
		Update("stu_is_check", true)
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
		"stu_is_check": true,
		"stu_num":      req.SchoolID,
		"stuCla":       loginResp.Major,
		"stuName":      loginResp.Name,
		"stu_pwd":      encPwd,
		"school":       req.School,
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
		"stu_num":      "",
		"stu_pwd":      "",
		"stuName":      "",
		"stuCla":       "",
		"stu_is_check": false,
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
	var out struct {
		IsLoginSnake bool   `json:"is_login"`
		IsLoginCamel bool   `json:"isLogin"`
		Major        string `json:"major"`
		Name         string `json:"name"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &JWLoginData{
		IsLogin: out.IsLoginSnake || out.IsLoginCamel,
		Major:   out.Major,
		Name:    out.Name,
	}, nil
}
