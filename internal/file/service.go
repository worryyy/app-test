package file

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/cosutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/responses"

)

const defaultMaxUploadMB = 10

type Service struct {
	db        *gorm.DB
	mongoDB   *mongo.Database
	redis     *redis.Client
	cfg       *config.Config
	logger    *zap.Logger
	cosClient *cosutil.Client
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
		s.cosClient = cosutil.NewClient(cfg.COS, logger)
	}
	return s
}

func (s *Service) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, userID string) (string, error) {
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			s.logger.Warn("close upload file failed", zap.Error(closeErr))
		}
	}()

	maxBytes := s.maxUploadBytes()
	if maxBytes > 0 && header != nil && header.Size > maxBytes {
		return "", result.ErrFileLimited
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("read upload file: %w", err)
	}
	if maxBytes > 0 && int64(len(data)) > maxBytes {
		return "", result.ErrFileLimited
	}
	contentType, err := normalizeContentType(header)
	if err != nil {
		return "", err
	}

	md5Hash := md5.Sum(data)
	md5Str := hex.EncodeToString(md5Hash[:])

	existing, err := s.findByUserAndMD5(ctx, userID, md5Str)
	if err != nil {
		return "", err
	}
	if existing != nil {
		if _, err := s.fileColl().UpdateByID(ctx, existing.ID, bson.M{"$inc": bson.M{"refCount": 1}}); err != nil {
			return "", fmt.Errorf("increase file ref count: %w", err)
		}
		return md5Str, nil
	}

	if s.cosClient != nil {
		_, err = s.cosClient.PutWithImageProcess(ctx, md5Str, data, contentType, s.cfg.COS.Compress)
		if err != nil {
			return "", fmt.Errorf("cos upload: %w", err)
		}
	}

	doc := File{
		MD5:      md5Str,
		IsPublic: false,
		UserID:   userID,
		RefCount: 1,
	}
	if _, err := s.fileColl().InsertOne(ctx, doc); err != nil {
		return "", fmt.Errorf("insert file record: %w", err)
	}
	return md5Str, nil
}

func (s *Service) Delete(ctx context.Context, md5Value, userID string, force bool) error {
	current, err := s.findByUserAndMD5(ctx, userID, md5Value)
	if err != nil {
		return err
	}
	if force {
		current, err = s.GetByMD5(ctx, md5Value)
		if err != nil {
			return err
		}
	}
	if current == nil {
		return result.ErrNotExisted
	}

	nextRef := current.RefCount - 1
	if force {
		nextRef = 0
	}
	if nextRef <= 0 {
		if _, err := s.fileColl().DeleteOne(ctx, bson.M{"_id": current.ID}); err != nil {
			return fmt.Errorf("delete file record: %w", err)
		}
		if s.cosClient != nil {
			if err := s.cosClient.Delete(ctx, md5Value); err != nil {
				s.logger.Warn("delete cos object failed", zap.Error(err), zap.String("md5", md5Value))
			}
		}
		return nil
	}

	if _, err := s.fileColl().UpdateByID(ctx, current.ID, bson.M{"$set": bson.M{"refCount": nextRef}}); err != nil {
		return fmt.Errorf("decrease refCount: %w", err)
	}
	return nil
}

func (s *Service) ListPublic(ctx context.Context, page, size int) ([]File, error) {
	return s.list(ctx, bson.M{"isPublic": true}, page, size, false)
}

func (s *Service) ListAll(ctx context.Context, page, size int) ([]File, error) {
	return s.list(ctx, bson.M{}, page, size, true)
}

func (s *Service) SetPublic(ctx context.Context, ids []string, isPublic bool) (int64, error) {
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
		return 0, nil
	}
	res, err := s.fileColl().UpdateMany(ctx, bson.M{"_id": bson.M{"$in": objectIDs}}, bson.M{"$set": bson.M{"isPublic": isPublic}})
	if err != nil {
		return 0, fmt.Errorf("set file public: %w", err)
	}
	return res.ModifiedCount, nil
}

func (s *Service) list(ctx context.Context, filter bson.M, page, size int, withCreatedTime bool) ([]File, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	cur, err := s.fileColl().Find(ctx, filter, options.Find().
		SetSkip(int64((page-1)*size)).
		SetLimit(int64(size)).
		SetSort(bson.M{"_id": -1}))
	if err != nil {
		return nil, fmt.Errorf("find files: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close file cursor failed", zap.Error(closeErr))
		}
	}()

	var files []File
	if err := cur.All(ctx, &files); err != nil {
		return nil, fmt.Errorf("decode files: %w", err)
	}
	if files == nil {
		return []File{}, nil
	}
	if withCreatedTime {
		for i := range files {
			files[i].CreatedTime = files[i].ID.Timestamp().UnixMilli()
		}
	}
	return files, nil
}

func (s *Service) fileColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_file")
}
