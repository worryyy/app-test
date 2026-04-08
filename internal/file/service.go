package file

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"mime/multipart"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/integration/cosutil"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
)

const defaultMaxUploadMB = 10

type Service struct {
	repo      *Repository
	cfg       *config.Config
	logger    *zap.Logger
	cosClient *cosutil.Client
}

func NewService(db *gorm.DB, mongoDB *mongo.Database, _ *redis.Client, cfg *config.Config, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}

	s := &Service{
		repo:   NewRepository(db, mongoDB),
		cfg:    cfg,
		logger: logger,
	}
	if cfg != nil {
		s.cosClient = cosutil.NewClient(cfg.COS, logger)
	}
	return s
}

func (s *Service) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, userID string) (string, error) {
	if file == nil || strings.TrimSpace(userID) == "" {
		return "", bizerr.Param(errMsgInvalidParam)
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			s.logger.Warn("close upload file failed", zap.Error(closeErr))
		}
	}()

	maxBytes := s.maxUploadBytes()
	if maxBytes > 0 && header != nil && header.Size > maxBytes {
		return "", ErrFileLimited
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return "", bizerr.InternalWrap("读取上传文件失败", err)
	}
	if maxBytes > 0 && int64(len(data)) > maxBytes {
		return "", ErrFileLimited
	}

	contentType, err := normalizeContentType(header)
	if err != nil {
		return "", err
	}

	md5Hash := md5.Sum(data)
	md5Str := hex.EncodeToString(md5Hash[:])

	existing, err := s.repo.FindFileByUserAndMD5(ctx, userID, md5Str)
	if err != nil {
		return "", bizerr.InternalWrap("查询文件失败", err)
	}
	if existing != nil {
		if err := s.repo.IncrementFileRefCount(ctx, existing.ID, 1); err != nil {
			return "", bizerr.InternalWrap("更新文件引用数失败", err)
		}
		return md5Str, nil
	}

	if s.cosClient != nil {
		compress := ""
		if s.cfg != nil {
			compress = s.cfg.COS.Compress
		}
		if _, err := s.cosClient.PutWithImageProcess(ctx, md5Str, data, contentType, compress); err != nil {
			return "", bizerr.InternalWrap("上传文件失败", err)
		}
	}

	doc := &File{
		MD5:      md5Str,
		IsPublic: false,
		UserID:   userID,
		RefCount: 1,
	}
	if err := s.repo.CreateFile(ctx, doc); err != nil {
		return "", bizerr.InternalWrap("保存文件记录失败", err)
	}
	return md5Str, nil
}

func (s *Service) Delete(ctx context.Context, md5Value, userID string, force bool) error {
	if strings.TrimSpace(md5Value) == "" || (!force && strings.TrimSpace(userID) == "") {
		return bizerr.Param(errMsgInvalidParam)
	}

	var (
		current *File
		err     error
	)
	if force {
		current, err = s.GetByMD5(ctx, md5Value)
	} else {
		current, err = s.repo.FindFileByUserAndMD5(ctx, userID, md5Value)
		if err != nil {
			return bizerr.InternalWrap("查询文件失败", err)
		}
	}
	if err != nil {
		return err
	}
	if current == nil {
		return ErrFileNotFound
	}

	nextRef := current.RefCount - 1
	if force {
		nextRef = 0
	}
	if nextRef <= 0 {
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

	if err := s.repo.UpdateFileRefCount(ctx, current.ID, nextRef); err != nil {
		return bizerr.InternalWrap("更新文件引用数失败", err)
	}
	return nil
}

func (s *Service) ListPublic(ctx context.Context, page, size int) ([]File, error) {
	files, err := s.repo.FindFilesPage(ctx, bson.M{"isPublic": true}, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询公开文件失败", err)
	}
	return files, nil
}

func (s *Service) ListAll(ctx context.Context, page, size int) ([]File, error) {
	files, err := s.repo.FindFilesPage(ctx, bson.M{}, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询文件列表失败", err)
	}
	for i := range files {
		files[i].CreatedTime = files[i].ID.Timestamp().UnixMilli()
	}
	return files, nil
}

func (s *Service) SetPublic(ctx context.Context, ids []string, isPublic bool) (int64, error) {
	if len(ids) == 0 {
		return 0, bizerr.Param(errMsgInvalidParam)
	}

	objectIDs := make([]primitive.ObjectID, 0, len(ids))
	seen := make(map[primitive.ObjectID]struct{}, len(ids))
	for _, id := range ids {
		if !primitive.IsValidObjectID(id) {
			continue
		}
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		if _, ok := seen[objectID]; ok {
			continue
		}
		seen[objectID] = struct{}{}
		objectIDs = append(objectIDs, objectID)
	}
	if len(objectIDs) == 0 {
		return 0, bizerr.Param(errMsgInvalidParam)
	}

	modified, err := s.repo.UpdateFilesPublic(ctx, objectIDs, isPublic)
	if err != nil {
		return 0, bizerr.InternalWrap("更新文件公开状态失败", err)
	}
	if modified <= 0 {
		return 0, ErrFileUpdateFailed
	}
	return modified, nil
}
