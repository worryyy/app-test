package file

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/cosutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

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

func (s *Service) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, userID string) (string, string, error) {
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", "", fmt.Errorf("read upload file: %w", err)
	}
	md5Hash := md5.Sum(data)
	md5Str := hex.EncodeToString(md5Hash[:])

	coll := s.fileColl()
	var existing File
	err = coll.FindOneAndUpdate(
		ctx,
		bson.M{"md5": md5Str},
		bson.M{"$inc": bson.M{"refCount": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&existing)
	if err == nil {
		return md5Str, s.fileURL(md5Str), nil
	}
	if err != mongo.ErrNoDocuments {
		return "", "", fmt.Errorf("query existing file: %w", err)
	}

	var url string
	if s.cosClient != nil {
		url, err = s.cosClient.PutWithImageProcess(ctx, md5Str, data, s.cfg.COS.Compress)
		if err != nil {
			contentType := ""
			if header != nil && header.Header != nil {
				contentType = header.Header.Get("Content-Type")
			}
			url, err = s.cosClient.Put(ctx, md5Str, data, contentType)
			if err != nil {
				return "", "", fmt.Errorf("cos upload: %w", err)
			}
		}
	} else {
		url = s.fileURL(md5Str)
	}

	doc := File{
		MD5:      md5Str,
		IsPublic: false,
		UserID:   userID,
		RefCount: 1,
	}
	if _, err := coll.InsertOne(ctx, doc); err != nil {
		return "", "", fmt.Errorf("insert file record: %w", err)
	}
	return md5Str, url, nil
}

func (s *Service) Delete(ctx context.Context, md5Value, userID string, force bool) error {
	coll := s.fileColl()
	filter := bson.M{"md5": md5Value}
	if !force && userID != "" {
		filter["userId"] = userID
	}

	var current File
	if err := coll.FindOne(ctx, filter).Decode(&current); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil
		}
		return fmt.Errorf("find file before delete: %w", err)
	}

	nextRef := current.RefCount - 1
	if force {
		nextRef = 0
	}
	if nextRef <= 0 {
		if _, err := coll.DeleteOne(ctx, bson.M{"_id": current.ID}); err != nil {
			return fmt.Errorf("delete file record: %w", err)
		}
		if s.cosClient != nil {
			if err := s.cosClient.Delete(ctx, md5Value); err != nil {
				s.logger.Warn("delete cos object failed", zap.Error(err), zap.String("md5", md5Value))
			}
		}
		return nil
	}

	if _, err := coll.UpdateByID(ctx, current.ID, bson.M{"$set": bson.M{"refCount": nextRef}}); err != nil {
		return fmt.Errorf("decrease refCount: %w", err)
	}
	return nil
}

func (s *Service) ListPublic(ctx context.Context, page, size int) (*result.CusPage[File], error) {
	return s.list(ctx, bson.M{"isPublic": true}, page, size)
}

func (s *Service) ListAll(ctx context.Context, page, size int) (*result.CusPage[File], error) {
	return s.list(ctx, bson.M{}, page, size)
}

func (s *Service) SetPublic(ctx context.Context, md5List []string, isPublic bool) error {
	if len(md5List) == 0 {
		return nil
	}
	_, err := s.fileColl().UpdateMany(ctx, bson.M{"md5": bson.M{"$in": md5List}}, bson.M{"$set": bson.M{"isPublic": isPublic}})
	if err != nil {
		return fmt.Errorf("set file public: %w", err)
	}
	return nil
}

func (s *Service) list(ctx context.Context, filter bson.M, page, size int) (*result.CusPage[File], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	total, err := s.fileColl().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count files: %w", err)
	}

	cur, err := s.fileColl().Find(ctx, filter, options.Find().
		SetSkip(int64((page-1)*size)).
		SetLimit(int64(size)).
		SetSort(bson.M{"_id": -1}))
	if err != nil {
		return nil, fmt.Errorf("find files: %w", err)
	}
	defer cur.Close(ctx)

	var files []File
	if err := cur.All(ctx, &files); err != nil {
		return nil, fmt.Errorf("decode files: %w", err)
	}
	return result.NewCusPage(files, total, page, size), nil
}

func (s *Service) fileColl() *mongo.Collection {
	return s.mongoDB.Collection("campus_file")
}

func (s *Service) fileURL(md5Value string) string {
	if s.cfg == nil {
		return md5Value
	}
	if s.cfg.COS.BaseCDN == "" {
		return md5Value
	}
	return fmt.Sprintf("%s%s", ensureSuffixSlash(s.cfg.COS.BaseCDN), md5Value)
}

func ensureSuffixSlash(v string) string {
	if v == "" {
		return v
	}
	if v[len(v)-1] == '/' {
		return v
	}
	return v + "/"
}

func now() time.Time {
	return time.Now()
}
