package school

import (
	"context"
	"encoding/json"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/encrypt"
)

func (s *Service) GetUserByID(ctx context.Context, id int64) (*campusUser, error) {
	user, err := s.repo.FindUserByID(ctx, id)
	if err != nil {
		return nil, bizerr.InternalWrap("查询用户失败", err)
	}
	return user, nil
}

func (s *Service) checkJWLogin(ctx context.Context, schoolID, password string) (*JWCommonResp, error) {
	resp, err := s.jwClient.CheckLogin(ctx, schoolID, password)
	if err != nil {
		return nil, bizerr.InternalWrap("登录教务系统失败", err)
	}
	if err := ensureJWRespSuccess(resp, "登录失败"); err != nil {
		return nil, err
	}

	loggedIn, _, _, err := decodeJWLoginMeta(resp.Data)
	if err != nil {
		return nil, bizerr.InternalWrap("解析教务登录结果失败", err)
	}
	if !loggedIn {
		return nil, bizerr.Biz(resp.Message)
	}
	return resp, nil
}

func (s *Service) encryptAES(raw string) (string, error) {
	if s.cfg == nil || strings.TrimSpace(s.cfg.Encryption.Key) == "" {
		return raw, nil
	}

	encrypted, err := encrypt.AESEncrypt(raw, s.cfg.Encryption.Key)
	if err != nil {
		return "", bizerr.InternalWrap("加密密码失败", err)
	}
	return encrypted, nil
}

func decodeJWLoginMeta(data any) (bool, string, string, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return false, "", "", err
	}

	var out struct {
		IsLoginSnake bool   `json:"is_login"`
		IsLoginCamel bool   `json:"isLogin"`
		Major        string `json:"major"`
		Name         string `json:"name"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return false, "", "", err
	}

	return out.IsLoginSnake || out.IsLoginCamel, out.Name, out.Major, nil
}

func ensureJWRespSuccess(resp *JWCommonResp, fallback string) error {
	if resp == nil {
		return bizerr.Internal(fallback)
	}
	if resp.Code == 200 {
		return nil
	}

	msg := strings.TrimSpace(resp.Message)
	if msg == "" {
		msg = fallback
	}
	return bizerr.Biz(msg)
}

func parseTermObjectID(termID string) (primitive.ObjectID, error) {
	termID = strings.TrimSpace(termID)
	if !primitive.IsValidObjectID(termID) {
		return primitive.NilObjectID, bizerr.Param(errMsgInvalidParam)
	}

	oid, err := primitive.ObjectIDFromHex(termID)
	if err != nil {
		return primitive.NilObjectID, bizerr.Param(errMsgInvalidParam)
	}
	return oid, nil
}

func (s *Service) termByID(ctx context.Context, termID string) (*Term, error) {
	oid, err := parseTermObjectID(termID)
	if err != nil {
		return nil, err
	}

	term, err := s.repo.FindTermByID(ctx, oid)
	if err != nil {
		return nil, bizerr.InternalWrap("查询学期失败", err)
	}
	if term == nil {
		return nil, ErrTermNotFound
	}
	return term, nil
}

func validateTerm(term *Term) error {
	if term == nil || strings.TrimSpace(term.Term) == "" || strings.TrimSpace(term.StartDate) == "" || term.TotalWeeks <= 0 {
		return bizerr.Param(errMsgInvalidParam)
	}
	return nil
}
