package file

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) GetByMD5(ctx context.Context, md5Value string) (*File, error) {
	var f File
	if err := s.fileColl().FindOne(ctx, bson.M{"md5": md5Value}).Decode(&f); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("get file by md5: %w", err)
	}
	return &f, nil
}

func (s *Service) GetDownloadURL(ctx context.Context, md5Value string, showOrigin bool) (string, error) {
	f, err := s.GetByMD5(ctx, md5Value)
	if err != nil {
		return "", err
	}
	if f == nil {
		return "", result.ErrNotExisted
	}
	if !showOrigin {
		return s.compressFileURL(md5Value), nil
	}
	return s.fileURL(md5Value), nil
}

func (s *Service) maxUploadBytes() int64 {
	maxMB := defaultMaxUploadMB
	if s.cfg != nil && s.cfg.Custom.MaxFileSizeMB > 0 {
		maxMB = s.cfg.Custom.MaxFileSizeMB
	}
	return int64(maxMB) * 1024 * 1024
}

func normalizeContentType(header *multipart.FileHeader) (string, error) {
	contentType := ""
	if header != nil && header.Header != nil {
		contentType = strings.ToLower(strings.TrimSpace(header.Header.Get("Content-Type")))
	}
	switch contentType {
	case "", "application/octet-stream":
		return "image/png", nil
	case "image/png", "image/jpeg", "image/x-icon":
		return contentType, nil
	default:
		return "", result.NewBizError(result.CodeParamError, "图片格式只能是[image/png image/jpeg image/x-icon application/octet-stream]")
	}
}

func (s *Service) findByUserAndMD5(ctx context.Context, userID, md5Value string) (*File, error) {
	var current File
	err := s.fileColl().FindOne(ctx, bson.M{"userId": userID, "md5": md5Value}).Decode(&current)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find file by user and md5: %w", err)
	}
	return &current, nil
}

func (s *Service) fileURL(md5Value string) string {
	if s.cfg == nil || s.cfg.COS.BaseCDN == "" {
		return md5Value
	}
	return fmt.Sprintf("%s%s", ensureSuffixSlash(s.cfg.COS.BaseCDN), md5Value)
}

func (s *Service) compressFileURL(md5Value string) string {
	if s.cfg == nil || s.cfg.COS.CompressBaseCDN == "" {
		return md5Value
	}
	return fmt.Sprintf("%s%s", ensureSuffixSlash(s.cfg.COS.CompressBaseCDN), s.compressObjectKey(md5Value))
}

func (s *Service) compressObjectKey(md5Value string) string {
	compress := "webp"
	if s.cfg != nil && strings.TrimSpace(s.cfg.COS.Compress) != "" {
		compress = strings.TrimSpace(s.cfg.COS.Compress)
	}
	return md5Value + "." + compress
}

func ensureSuffixSlash(v string) string {
	if v == "" || strings.HasSuffix(v, "/") {
		return v
	}
	return v + "/"
}
