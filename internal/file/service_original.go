package file

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

const AcademicMaxUploadBytes int64 = 50 * 1024 * 1024

type OriginalUpload struct {
	MD5          string
	OriginalName string
	MIME         string
	Size         int64
}

var academicExtensions = map[string]struct{}{
	".pdf": {}, ".doc": {}, ".docx": {}, ".ppt": {}, ".pptx": {},
	".xls": {}, ".xlsx": {}, ".txt": {}, ".png": {}, ".jpg": {},
	".jpeg": {}, ".gif": {}, ".webp": {},
}

func (s *Service) UploadAcademicDocument(ctx context.Context, stream multipart.File, header *multipart.FileHeader, userID string) (*OriginalUpload, error) {
	if stream == nil || header == nil || strings.TrimSpace(userID) == "" {
		return nil, bizerr.Param(errMsgInvalidParam)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			s.logger.Warn("close document upload failed", zap.Error(err))
		}
	}()
	if header.Size > AcademicMaxUploadBytes {
		return nil, ErrFileLimited
	}
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(header.Filename)))
	if _, ok := academicExtensions[ext]; !ok {
		return nil, ErrUnsupportedDocumentFormat
	}
	data, err := io.ReadAll(io.LimitReader(stream, AcademicMaxUploadBytes+1))
	if err != nil {
		return nil, bizerr.InternalWrap("读取上传文件失败", err)
	}
	if int64(len(data)) > AcademicMaxUploadBytes {
		return nil, ErrFileLimited
	}
	mimeType := mimetype.Detect(data).String()
	if !validAcademicDocument(data, ext, mimeType) {
		return nil, ErrUnsupportedDocumentFormat
	}
	hash := md5.Sum(data)
	md5Value := hex.EncodeToString(hash[:])
	existing, err := s.repo.FindFileByMD5(ctx, md5Value)
	if err != nil {
		return nil, bizerr.InternalWrap("查询文件失败", err)
	}
	if existing != nil {
		if err := s.repo.IncrementFileRefCount(ctx, existing.ID, 1); err != nil {
			return nil, bizerr.InternalWrap("更新文件引用数失败", err)
		}
		return &OriginalUpload{MD5: md5Value, OriginalName: header.Filename, MIME: mimeType, Size: int64(len(data))}, nil
	}
	if s.cosClient != nil {
		if _, err := s.cosClient.Put(ctx, md5Value, data, mimeType); err != nil {
			return nil, bizerr.InternalWrap("上传文件失败", err)
		}
	}
	if err := s.repo.CreateFile(ctx, &File{MD5: md5Value, UserID: userID, RefCount: 1}); err != nil {
		return nil, bizerr.InternalWrap("保存文件记录失败", err)
	}
	return &OriginalUpload{MD5: md5Value, OriginalName: header.Filename, MIME: mimeType, Size: int64(len(data))}, nil
}

func (s *Service) ReleaseReference(ctx context.Context, md5Value string) error {
	current, err := s.GetByMD5(ctx, md5Value)
	if err != nil {
		return err
	}
	if current == nil {
		return ErrFileNotFound
	}
	if current.RefCount > 1 {
		return s.repo.UpdateFileRefCount(ctx, current.ID, current.RefCount-1)
	}
	if err := s.repo.DeleteFile(ctx, current.ID); err != nil {
		return bizerr.InternalWrap("删除文件记录失败", err)
	}
	if s.cosClient != nil {
		if err := s.cosClient.Delete(ctx, md5Value); err != nil {
			s.logger.Warn("delete cos object failed", zap.Error(err), zap.String("md5", md5Value))
		}
	}
	return nil
}

func (s *Service) OriginalURL(md5Value string) string { return s.fileURL(md5Value) }

func validAcademicDocument(data []byte, ext, mimeType string) bool {
	switch ext {
	case ".docx", ".xlsx", ".pptx":
		if mimeType != "application/zip" && !strings.Contains(mimeType, "officedocument") {
			return false
		}
		return validOOXML(data, ext)
	case ".pdf":
		return mimeType == "application/pdf"
	case ".doc":
		return mimeType == "application/msword" || mimeType == "application/octet-stream"
	case ".xls":
		return strings.Contains(mimeType, "excel") || mimeType == "application/x-ole-storage" || mimeType == "application/octet-stream"
	case ".ppt":
		return strings.Contains(mimeType, "powerpoint") || mimeType == "application/x-ole-storage" || mimeType == "application/octet-stream"
	case ".txt":
		return strings.HasPrefix(mimeType, "text/plain")
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return strings.HasPrefix(mimeType, "image/")
	default:
		return false
	}
}

func validOOXML(data []byte, ext string) bool {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return false
	}
	wanted := map[string]string{".docx": "word/", ".xlsx": "xl/", ".pptx": "ppt/"}[ext]
	hasContentTypes, hasPayload := false, false
	for _, entry := range reader.File {
		if entry.Name == "[Content_Types].xml" {
			hasContentTypes = true
		}
		if strings.HasPrefix(entry.Name, wanted) {
			hasPayload = true
		}
	}
	return hasContentTypes && hasPayload
}
