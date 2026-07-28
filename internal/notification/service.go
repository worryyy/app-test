package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/snowflake"
)

var (
	ErrInvalidCategory      = bizerr.Param("通知分类不合法")
	ErrNotificationNotFound = bizerr.NotFound("通知不存在")
	ErrNotificationType     = bizerr.Param("type不能为空")
)

type IdentityResolver interface {
	ResolveRootUserID(ctx context.Context, userID int64) (int64, error)
	ListIdentityIDs(ctx context.Context, rootUserID int64) ([]int64, error)
}

type Service struct {
	repo       *Repository
	redis      *redis.Client
	logger     *zap.Logger
	identities IdentityResolver
}

func NewService(mongoDB *mongo.Database, rds *redis.Client, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{repo: NewRepository(mongoDB), redis: rds, logger: logger}
}

func (s *Service) SetIdentityResolver(resolver IdentityResolver) {
	s.identities = resolver
}

func (s *Service) EnsureIndexes(ctx context.Context) error {
	return s.repo.EnsureIndexes(ctx)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*Document, bool, error) {
	if input.ReceiverRootUserID <= 0 || !validCategory(strings.TrimSpace(input.Category)) || strings.TrimSpace(input.EventType) == "" {
		return nil, false, bizerr.Param("通知参数错误")
	}
	createdTime := input.CreatedTime
	if createdTime.IsZero() {
		createdTime = time.Now()
	}
	id := snowflake.Generate().String()
	doc := &Document{
		ID: id, EventID: strings.TrimSpace(input.EventID), ReceiverRootUserID: input.ReceiverRootUserID,
		ReceiverID: input.ReceiverID, SenderID: input.SenderID, Category: input.Category,
		EventType: input.EventType, Type: input.EventType, Title: input.Title, Content: input.Content,
		ResourceType: input.ResourceType, ResourceID: input.ResourceID, Extra: input.Extra,
		CreatedTime: createdTime, IsRead: false,
	}
	inserted, err := s.repo.Insert(ctx, doc)
	if err != nil || !inserted {
		return doc, inserted, err
	}
	s.publish(ctx, doc)
	return doc, true, nil
}

func (s *Service) CreateLegacy(ctx context.Context, input LegacyInput) error {
	targetUserID, err := strconv.ParseInt(strings.TrimSpace(input.TargetUserID), 10, 64)
	if err != nil || targetUserID <= 0 {
		return bizerr.Param("通知接收者不合法")
	}
	rootUserID := targetUserID
	if s.identities != nil {
		resolved, resolveErr := s.identities.ResolveRootUserID(ctx, targetUserID)
		if resolveErr != nil {
			return bizerr.InternalWrap("查询通知主账号失败", resolveErr)
		}
		if resolved > 0 {
			rootUserID = resolved
		}
	}
	createdTime := input.CreatedTime
	if createdTime.IsZero() {
		createdTime = time.Now()
	}
	eventID := ""
	if input.EventID > 0 {
		eventID = "mq:" + strconv.FormatInt(input.EventID, 10)
	}
	doc := &Document{
		ID: snowflake.Generate().String(), EventID: eventID, ReceiverRootUserID: rootUserID,
		ReceiverID: input.TargetUserID, SenderID: input.SenderUserID, Category: CategorySocial,
		EventType: input.Type, Type: input.Type, Content: input.Content,
		TopicID: input.TopicID, CommentID: input.CommentID, CreatedTime: createdTime,
	}
	inserted, err := s.repo.Insert(ctx, doc)
	if err != nil || !inserted {
		return err
	}
	s.publish(ctx, doc)
	return nil
}

func (s *Service) List(ctx context.Context, rootUserID int64, category string, page, size int) (*pagination.PageResult[Response], error) {
	if rootUserID <= 0 {
		return nil, bizerr.Param("用户参数错误")
	}
	category = strings.TrimSpace(category)
	if category != "" && !validCategory(category) {
		return nil, ErrInvalidCategory
	}
	page, size = normalizePage(page, size)
	identityIDs, err := s.identityIDs(ctx, rootUserID)
	if err != nil {
		return nil, err
	}
	docs, total, err := s.repo.List(ctx, rootUserID, identityIDs, category, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询通知失败", err)
	}
	items := make([]Response, 0, len(docs))
	for _, doc := range docs {
		items = append(items, doc.response())
	}
	return pagination.NewPageResult(items, total, page, size), nil
}

func (s *Service) UnreadCounts(ctx context.Context, rootUserID int64) (map[string]int64, error) {
	identityIDs, err := s.identityIDs(ctx, rootUserID)
	if err != nil {
		return nil, err
	}
	counts, err := s.repo.UnreadCategories(ctx, rootUserID, identityIDs)
	if err != nil {
		return nil, bizerr.InternalWrap("查询通知未读数失败", err)
	}
	for category := range categories {
		if _, ok := counts[category]; !ok {
			counts[category] = 0
		}
	}
	return counts, nil
}

func (s *Service) MarkOneRead(ctx context.Context, rootUserID int64, id string) error {
	if strings.TrimSpace(id) == "" {
		return bizerr.Param("通知ID不能为空")
	}
	identityIDs, err := s.identityIDs(ctx, rootUserID)
	if err != nil {
		return err
	}
	ok, err := s.repo.MarkOneRead(ctx, rootUserID, identityIDs, id)
	if err != nil {
		return bizerr.InternalWrap("更新通知已读状态失败", err)
	}
	if !ok {
		return ErrNotificationNotFound
	}
	return nil
}

func (s *Service) MarkRead(ctx context.Context, rootUserID int64, category string) (int64, error) {
	category = strings.TrimSpace(category)
	if category != "" && !validCategory(category) {
		return 0, ErrInvalidCategory
	}
	identityIDs, err := s.identityIDs(ctx, rootUserID)
	if err != nil {
		return 0, err
	}
	count, err := s.repo.MarkRead(ctx, rootUserID, identityIDs, category)
	if err != nil {
		return 0, bizerr.InternalWrap("更新通知已读状态失败", err)
	}
	return count, nil
}

func (s *Service) ListLegacy(ctx context.Context, currentUserID, rootUserID int64, typ string, page, size int) (any, error) {
	if strings.TrimSpace(typ) == "" {
		return nil, ErrNotificationType
	}
	page, size = normalizePage(page, size)
	identityIDs, err := s.identityIDs(ctx, rootUserID)
	if err != nil {
		return nil, err
	}
	identityIDs = append(identityIDs, currentUserID)
	docs, total, err := s.repo.ListLegacy(ctx, rootUserID, identityIDs, typ, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询通知失败", err)
	}
	items := make([]LegacyResponse, 0, len(docs))
	for _, doc := range docs {
		items = append(items, doc.legacyResponse())
	}
	if len(items) == 0 {
		return []LegacyResponse{}, nil
	}
	if err := s.repo.MarkLatestLegacyRead(ctx, rootUserID, identityIDs, typ); err != nil {
		return nil, bizerr.InternalWrap("更新通知已读状态失败", err)
	}
	return pagination.NewPageResult(items, total, page, size), nil
}

func (s *Service) LatestLegacy(ctx context.Context, currentUserID, rootUserID int64, typ string) (*LegacyResponse, error) {
	if strings.TrimSpace(typ) == "" {
		return nil, ErrNotificationType
	}
	identityIDs, err := s.identityIDs(ctx, rootUserID)
	if err != nil {
		return nil, err
	}
	identityIDs = append(identityIDs, currentUserID)
	doc, err := s.repo.LatestLegacy(ctx, rootUserID, identityIDs, typ)
	if err != nil {
		return nil, bizerr.InternalWrap("查询最新通知失败", err)
	}
	if doc == nil {
		return nil, nil
	}
	response := doc.legacyResponse()
	return &response, nil
}

func (s *Service) HaveUnreadLegacy(ctx context.Context, currentUserID, rootUserID int64, typ string) (bool, error) {
	latest, err := s.LatestLegacy(ctx, currentUserID, rootUserID, typ)
	return latest != nil && !latest.IsRead, err
}

func (s *Service) StartSubscriber(ctx context.Context, fn func(Broadcast)) (func() error, error) {
	if s.redis == nil {
		return func() error { return nil }, nil
	}
	pubsub := s.redis.Subscribe(ctx, BroadcastChannel)
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, fmt.Errorf("subscribe notification broadcasts: %w", err)
	}
	go func() {
		for message := range pubsub.Channel() {
			var event Broadcast
			if err := json.Unmarshal([]byte(message.Payload), &event); err != nil {
				s.logger.Warn("decode notification broadcast failed", zap.Error(err))
				continue
			}
			fn(event)
		}
	}()
	return pubsub.Close, nil
}

func (s *Service) publish(ctx context.Context, doc *Document) {
	if s.redis == nil || doc == nil {
		return
	}
	event := Broadcast{RootUserID: doc.ReceiverRootUserID, LegacyReceiverID: doc.ReceiverID, Notification: doc.response(), Legacy: doc.legacyResponse()}
	data, err := json.Marshal(event)
	if err != nil {
		s.logger.Warn("marshal notification broadcast failed", zap.Error(err))
		return
	}
	if err := s.redis.Publish(ctx, BroadcastChannel, data).Err(); err != nil {
		s.logger.Warn("publish notification broadcast failed", zap.Error(err))
	}
}

func (s *Service) identityIDs(ctx context.Context, rootUserID int64) ([]int64, error) {
	if rootUserID <= 0 {
		return nil, bizerr.Param("主账号参数错误")
	}
	if s.identities == nil {
		return []int64{rootUserID}, nil
	}
	ids, err := s.identities.ListIdentityIDs(ctx, rootUserID)
	if err != nil {
		return nil, bizerr.InternalWrap("查询账号身份失败", err)
	}
	if len(ids) == 0 {
		ids = []int64{rootUserID}
	}
	return ids, nil
}

func normalizePage(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
