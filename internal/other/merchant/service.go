package merchant

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Service struct {
	db      *gorm.DB
	mongoDB *mongo.Database
	cfg     *config.Config
	logger  *zap.Logger
}

func NewService(db *gorm.DB, mongoDB *mongo.Database, cfg *config.Config, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		db:      db,
		mongoDB: mongoDB,
		cfg:     cfg,
		logger:  logger,
	}
}

func (s *Service) defaultPageSize() int {
	if s != nil && s.cfg != nil && s.cfg.Custom.PageSize > 0 {
		return s.cfg.Custom.PageSize
	}
	return 15
}

func (s *Service) AddMerchantTheme(ctx context.Context, themeID string) (string, error) {
	// Validate theme exists in campus_theme_id collection (matches Java's themeService.existed)
	count, err := s.mongoDB.Collection("campus_theme_id").CountDocuments(ctx, bson.M{"themeId": themeID})
	if err != nil {
		return "", fmt.Errorf("check theme existence: %w", err)
	}
	if count == 0 {
		return "", result.NewBizError(result.CodeFail, "主题不存在")
	}

	existing, err := s.getMerchantThemeByThemeID(ctx, themeID)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return existing.ID.Hex(), nil
	}

	doc := MerchantTheme{ThemeID: themeID}
	res, err := s.mongoDB.Collection("campus_merchant_theme").InsertOne(ctx, doc)
	if err != nil {
		return "", fmt.Errorf("add merchant theme: %w", err)
	}
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("merchant theme id invalid")
	}
	return oid.Hex(), nil
}

func (s *Service) DeleteMerchantTheme(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid merchant theme id: %w", err)
	}
	if _, err := s.mongoDB.Collection("campus_merchant_theme").DeleteOne(ctx, bson.M{"_id": oid}); err != nil {
		return fmt.Errorf("delete merchant theme: %w", err)
	}
	return nil
}

func (s *Service) ListMerchantThemes(ctx context.Context) ([]MerchantTheme, error) {
	cur, err := s.mongoDB.Collection("campus_merchant_theme").Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"_id": -1}))
	if err != nil {
		return nil, fmt.Errorf("list merchant themes: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close merchant theme cursor failed", zap.Error(closeErr))
		}
	}()

	var list []MerchantTheme
	if err := cur.All(ctx, &list); err != nil {
		return nil, fmt.Errorf("decode merchant themes: %w", err)
	}
	return list, nil
}

func (s *Service) CreateTask(ctx context.Context, name string) error {
	existed, err := s.taskFuncExists(ctx, name, 0)
	if err != nil {
		return err
	}
	if existed {
		return result.NewBizError(result.CodeFail, name+"已存在")
	}

	t := Task{
		Status:    0,
		Detail:    "",
		Parent:    0,
		Func:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(&t).Error; err != nil {
		return result.NewBizError(result.CodeFail, "添加失败")
	}
	return nil
}

func (s *Service) DeleteTask(ctx context.Context, id int64) error {
	var task Task
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result.NewBizError(result.CodeFail, fmt.Sprintf("id:%d不存在", id))
		}
		return fmt.Errorf("query task before delete: %w", err)
	}

	res := s.db.WithContext(ctx).Delete(&Task{}, id)
	if res.Error != nil {
		return fmt.Errorf("delete task: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return result.NewBizError(result.CodeFail, "删除失败")
	}
	return nil
}

func (s *Service) UpdateTask(ctx context.Context, id int64, name string) error {
	task, err := s.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if task == nil {
		return result.NewBizError(result.CodeFail, "不存在")
	}

	if name != "" {
		existed, err := s.taskFuncExists(ctx, name, id)
		if err != nil {
			return err
		}
		if existed {
			return result.NewBizError(result.CodeFail, "已存在")
		}
		task.Func = name
	}
	task.UpdatedAt = time.Now()
	res := s.db.WithContext(ctx).Model(&Task{}).Where("id = ?", id).Updates(task)
	if res.Error != nil {
		return fmt.Errorf("update task: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return result.NewBizError(result.CodeFail, "更新失败")
	}
	return nil
}

func (s *Service) GetTask(ctx context.Context, id int64) (*Task, error) {
	var t Task
	if err := s.db.WithContext(ctx).First(&t, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get task: %w", err)
	}
	return &t, nil
}

func (s *Service) ListTasks(ctx context.Context, page, size int) (*result.PageResult[Task], error) {
	if size <= 0 {
		size = s.defaultPageSize()
	}
	var total int64
	if err := s.db.WithContext(ctx).Model(&Task{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count tasks: %w", err)
	}
	var list []Task
	if err := s.db.WithContext(ctx).Offset((page - 1) * size).Limit(size).Order("id DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return result.NewPage(list, total, page, size), nil
}

func (s *Service) taskFuncExists(ctx context.Context, name string, excludeID int64) (bool, error) {
	query := s.db.WithContext(ctx).Model(&Task{}).Where("func = ?", name)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("count task func: %w", err)
	}
	return count > 0, nil
}

func (s *Service) getMerchantThemeByThemeID(ctx context.Context, themeID string) (*MerchantTheme, error) {
	var doc MerchantTheme
	err := s.mongoDB.Collection("campus_merchant_theme").FindOne(ctx, bson.M{"themeId": themeID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get merchant theme by theme id: %w", err)
	}
	return &doc, nil
}
