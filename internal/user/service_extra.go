package user

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

func (s *Service) RandomNickname(accountType string) (string, error) {
	switch strings.TrimSpace(accountType) {
	case accountTypeBase:
		return randomHumorousID(), nil
	case accountTypeAnonymous:
		return randomAnonymousID(), nil
	default:
		return "", bizerr.Param("accountType 非法")
	}
}

func (s *Service) GenerateUnlimitedWXACode(ctx context.Context, scene, page string) (string, error) {
	if s.wxClient == nil {
		return "", bizerr.Internal("wx client not initialized")
	}

	data, err := s.wxClient.UnlimitedWXACode(ctx, scene, page)
	if err != nil {
		return "", bizerr.InternalWrap("生成小程序码失败", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
