package school

import (
	"context"
	"encoding/json"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/genproto/googleapis/rpc/code"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/encrypt"
)

func (s *Service) GetUserByID(ctx context.Context, id int64) (*campusUser, error) {
	user, err := s.repo.FindUserByID(ctx, id)
	if err != nil {
		return nil, bizerr.InternalWrap("查询用户失败", err)
	}
	return user, nil
}

func (s *Service) checkJWLogin(ctx context.Context, schoolID, password string) (*JWCommonResp, error) {
	var resp *JWCommonResp
	resp, err := s.jwClient.CheckLogin(ctx, schoolID, password)
	if err != nil {
		return nil, bizerr.InternalWrap("登录教务系统失败", err)
	}
	if resp == nil {
		return nil, bizerr.Internal("登录教务系统失败: 响应为空")
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
