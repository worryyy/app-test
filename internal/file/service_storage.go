package file

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

func (s *Service) GetByMD5(ctx context.Context, md5Value string) (*File, error) {
	if strings.TrimSpace(md5Value) == "" {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	file, err := s.repo.FindFileByMD5(ctx, md5Value)
	if err != nil {
		return nil, bizerr.InternalWrap("查询文件失败", err)
	}
	return file, nil
}

func (s *Service) GetDownloadURL(ctx context.Context, md5Value string, showOrigin bool) (string, error) {
	f, err := s.GetByMD5(ctx, md5Value)
	if err != nil {
		return "", err
	}
	if f == nil {
		return "", ErrFileNotFound
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
		return "", ErrUnsupportedFileFormat
	}
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
