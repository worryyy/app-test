package other

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) AddMerchantTheme(ctx context.Context, themeID string) (string, error) {
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

func (s *Service) AddSupport(ctx context.Context, support *FrontendSupport) (string, error) {
	count, err := s.mongoDB.Collection("campus_frontend_support").CountDocuments(ctx, bson.M{"key": support.Key})
	if err != nil {
		return "", fmt.Errorf("count support by key: %w", err)
	}
	if count > 0 {
		return "", result.NewBizError(result.CodeFail, "key = "+support.Key+"has existed")
	}

	res, err := s.mongoDB.Collection("campus_frontend_support").InsertOne(ctx, support)
	if err != nil {
		return "", fmt.Errorf("add support: %w", err)
	}
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("support id invalid")
	}
	return oid.Hex(), nil
}

func (s *Service) UpdateSupport(ctx context.Context, support *FrontendSupport) error {
	if support == nil {
		return nil
	}
	if support.ID.IsZero() {
		update := bson.M{"val": support.Val}
		if support.KeyDesc != "" {
			update["keyDesc"] = support.KeyDesc
		}
		res, err := s.mongoDB.Collection("campus_frontend_support").UpdateOne(ctx, bson.M{"key": support.Key}, bson.M{"$set": update})
		if err != nil {
			return fmt.Errorf("update support by key: %w", err)
		}
		if res.MatchedCount == 0 {
			return result.NewBizError(result.CodeFail, support.Key+" not existed")
		}
		return nil
	}
	_, err := s.mongoDB.Collection("campus_frontend_support").UpdateByID(ctx, support.ID, bson.M{"$set": bson.M{
		"key":     support.Key,
		"val":     support.Val,
		"keyDesc": support.KeyDesc,
	}})
	if err != nil {
		return fmt.Errorf("update support: %w", err)
	}
	return nil
}

func (s *Service) DeleteSupport(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid support id: %w", err)
	}
	if _, err := s.mongoDB.Collection("campus_frontend_support").DeleteOne(ctx, bson.M{"_id": oid}); err != nil {
		return fmt.Errorf("delete support: %w", err)
	}
	return nil
}

func (s *Service) ListSupport(ctx context.Context) ([]FrontendSupport, error) {
	cur, err := s.mongoDB.Collection("campus_frontend_support").Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"_id": -1}))
	if err != nil {
		return nil, fmt.Errorf("list support: %w", err)
	}
	defer func() {
		if closeErr := cur.Close(ctx); closeErr != nil {
			s.logger.Warn("close support cursor failed", zap.Error(closeErr))
		}
	}()

	var list []FrontendSupport
	if err := cur.All(ctx, &list); err != nil {
		return nil, fmt.Errorf("decode support: %w", err)
	}
	return list, nil
}

func (s *Service) GetSupportByKey(ctx context.Context, key string) (*FrontendSupport, error) {
	var support FrontendSupport
	err := s.mongoDB.Collection("campus_frontend_support").FindOne(ctx, bson.M{"key": key}).Decode(&support)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, result.ErrNotExisted
		}
		return nil, fmt.Errorf("get support by key: %w", err)
	}
	return &support, nil
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
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

func (s *Service) DeleteTask(ctx context.Context, id int64) error {
	if err := s.db.WithContext(ctx).Delete(&Task{}, id).Error; err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}

func (s *Service) UpdateTask(ctx context.Context, id int64, name string) error {
	task, err := s.GetTask(ctx, id)
	if err != nil {
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
	if err := s.db.WithContext(ctx).Model(&Task{}).Where("id = ?", id).Updates(task).Error; err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

func (s *Service) GetTask(ctx context.Context, id int64) (*Task, error) {
	var t Task
	if err := s.db.WithContext(ctx).First(&t, id).Error; err != nil {
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
