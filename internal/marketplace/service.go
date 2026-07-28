package marketplace

import (
	"context"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/snowflake"
	"github.com/Milchstrassse/Ecampus-go/internal/sensitive"
)

type SellerVerifier interface {
	VerifyMarketplaceSeller(context.Context, int64) error
}

type CapabilityChecker interface {
	CheckCapability(context.Context, int64, int64, string) error
}

type Notifier interface {
	NotifyMarketplace(context.Context, int64, string, string, string, string) error
}

type Service struct {
	repo         *Repository
	verifier     SellerVerifier
	capabilities CapabilityChecker
	notifier     Notifier
	filter       sensitive.Filter
	gateway      Gateway
	logger       *zap.Logger
	now          func() time.Time
}

func NewService(db *gorm.DB, gateway Gateway, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	if gateway == nil {
		gateway = disabledGateway{}
	}
	return &Service{repo: NewRepository(db), gateway: gateway, logger: logger, now: time.Now}
}

func (s *Service) SetSellerVerifier(value SellerVerifier)       { s.verifier = value }
func (s *Service) SetCapabilityChecker(value CapabilityChecker) { s.capabilities = value }
func (s *Service) SetNotifier(value Notifier)                   { s.notifier = value }
func (s *Service) SetSensitiveFilter(value sensitive.Filter)    { s.filter = value }

func (s *Service) ListCategories(ctx context.Context, includeDisabled bool) ([]CategoryResponse, error) {
	items, err := s.repo.ListCategories(ctx, includeDisabled)
	if err != nil {
		return nil, bizerr.InternalWrap("查询商品类目失败", err)
	}
	responses := make([]CategoryResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, categoryResponse(item))
	}
	return responses, nil
}

func (s *Service) SaveCategory(ctx context.Context, categoryID string, req CategoryReq) (*CategoryResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, bizerr.Param("类目名称不能为空")
	}
	rate := DefaultCommissionRateBps
	if req.CommissionRateBps != nil {
		rate = *req.CommissionRateBps
	}
	now := s.now()
	item := &Category{ID: snowflake.Generate().Int64(), Name: name, CommissionRateBps: rate, Status: StatusActive, CreatedAt: now, UpdatedAt: now}
	if strings.TrimSpace(categoryID) != "" {
		id, err := parseMarketplaceID(categoryID)
		if err != nil {
			return nil, err
		}
		existing, err := s.repo.FindCategory(ctx, id)
		if err != nil {
			return nil, bizerr.InternalWrap("查询商品类目失败", err)
		}
		if existing == nil {
			return nil, ErrCategoryNotFound
		}
		item.ID, item.Status, item.CreatedAt = existing.ID, existing.Status, existing.CreatedAt
		if req.CommissionRateBps == nil {
			item.CommissionRateBps = existing.CommissionRateBps
		}
	}
	if err := s.repo.SaveCategory(ctx, item); err != nil {
		return nil, bizerr.InternalWrap("保存商品类目失败", err)
	}
	response := categoryResponse(*item)
	return &response, nil
}

func (s *Service) CreateItem(ctx context.Context, userID, rootUserID int64, req ItemReq) (*ItemResponse, error) {
	if err := s.checkNewListing(ctx, userID, rootUserID); err != nil {
		return nil, err
	}
	item, err := s.itemFromRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	now := s.now()
	item.ID, item.SellerRootUserID, item.SellerUserID, item.CreatedAt, item.UpdatedAt = snowflake.Generate().Int64(), rootUserID, userID, now, now
	if err := s.repo.CreateItem(ctx, item); err != nil {
		return nil, bizerr.InternalWrap("创建商品失败", err)
	}
	category, _ := s.repo.FindCategory(ctx, item.CategoryID)
	if category != nil {
		item.Category = *category
	}
	response := itemResponse(*item)
	return &response, nil
}

func (s *Service) UpdateItem(ctx context.Context, userID, rootUserID int64, itemID string, req ItemReq) (*ItemResponse, error) {
	if err := s.checkNewListing(ctx, userID, rootUserID); err != nil {
		return nil, err
	}
	id, err := parseMarketplaceID(itemID)
	if err != nil {
		return nil, err
	}
	current, err := s.repo.FindItem(ctx, id)
	if err != nil || current == nil || current.SellerRootUserID != rootUserID {
		return nil, ErrItemNotFound
	}
	item, err := s.itemFromRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	item.ID, item.UpdatedAt = id, s.now()
	if strings.TrimSpace(req.Status) == "" {
		item.Status = current.Status
	}
	ok, err := s.repo.UpdateItem(ctx, item, rootUserID)
	if err != nil {
		return nil, bizerr.InternalWrap("更新商品失败", err)
	}
	if !ok {
		return nil, ErrItemNotFound
	}
	updated, err := s.repo.FindItem(ctx, id)
	if err != nil || updated == nil {
		return nil, ErrItemNotFound
	}
	response := itemResponse(*updated)
	return &response, nil
}

func (s *Service) WithdrawItem(ctx context.Context, rootUserID int64, itemID string) error {
	id, err := parseMarketplaceID(itemID)
	if err != nil {
		return err
	}
	ok, err := s.repo.WithdrawItem(ctx, id, rootUserID, s.now())
	if err != nil {
		return bizerr.InternalWrap("下架商品失败", err)
	}
	if !ok {
		return ErrItemNotFound
	}
	return nil
}

func (s *Service) ItemDetail(ctx context.Context, rootUserID int64, itemID string) (*ItemResponse, error) {
	id, err := parseMarketplaceID(itemID)
	if err != nil {
		return nil, err
	}
	item, err := s.repo.FindItem(ctx, id)
	if err != nil {
		return nil, bizerr.InternalWrap("查询商品失败", err)
	}
	if item == nil || (item.SellerRootUserID != rootUserID && item.Status != ItemPublished && item.Status != ItemReserved && item.Status != ItemSold) {
		return nil, ErrItemNotFound
	}
	response := itemResponse(*item)
	return &response, nil
}

func (s *Service) SearchItems(ctx context.Context, rootUserID int64, mine bool, keyword, categoryID string, page, size int) (*pagination.PageResult[ItemResponse], error) {
	category := int64(0)
	var err error
	if strings.TrimSpace(categoryID) != "" {
		category, err = parseMarketplaceID(categoryID)
		if err != nil {
			return nil, err
		}
	}
	page, size = normalizeMarketplacePage(page, size)
	items, total, err := s.repo.SearchItems(ctx, rootUserID, mine, keyword, category, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("搜索商品失败", err)
	}
	responses := make([]ItemResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, itemResponse(item))
	}
	return pagination.NewPageResult(responses, total, page, size), nil
}

func (s *Service) itemFromRequest(ctx context.Context, req ItemReq) (*Item, error) {
	categoryID, err := parseMarketplaceID(req.CategoryID)
	if err != nil {
		return nil, err
	}
	category, err := s.repo.FindCategory(ctx, categoryID)
	if err != nil || category == nil || category.Status != StatusActive {
		return nil, ErrCategoryNotFound
	}
	if req.PriceCents <= 0 {
		return nil, bizerr.Param("商品价格必须大于0")
	}
	if err := validateImages(req.Images); err != nil {
		return nil, err
	}
	title, err := s.filterText(ctx, req.Title)
	if err != nil {
		return nil, err
	}
	description, err := s.filterText(ctx, req.Description)
	if err != nil {
		return nil, err
	}
	location, err := s.filterText(ctx, req.DeliveryLocation)
	if err != nil {
		return nil, err
	}
	condition := strings.TrimSpace(req.Condition)
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = ItemDraft
	}
	if strings.TrimSpace(title) == "" || strings.TrimSpace(description) == "" || condition == "" || strings.TrimSpace(location) == "" || (status != ItemDraft && status != ItemPublished) {
		return nil, bizerr.Param("商品参数错误")
	}
	return &Item{CategoryID: categoryID, Title: strings.TrimSpace(title), Description: strings.TrimSpace(description), Condition: condition, PriceCents: req.PriceCents, Images: append([]string(nil), req.Images...), DeliveryLocation: strings.TrimSpace(location), Status: status}, nil
}

func (s *Service) checkNewListing(ctx context.Context, userID, rootUserID int64) error {
	if s.verifier == nil {
		return bizerr.Internal("卖家认证能力未配置")
	}
	if err := s.verifier.VerifyMarketplaceSeller(ctx, rootUserID); err != nil {
		return err
	}
	if s.capabilities != nil {
		if err := s.capabilities.CheckCapability(ctx, userID, rootUserID, "content"); err != nil {
			return err
		}
		if err := s.capabilities.CheckCapability(ctx, userID, rootUserID, "trade"); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) checkTrade(ctx context.Context, userID, rootUserID int64) error {
	if s.capabilities == nil {
		return nil
	}
	return s.capabilities.CheckCapability(ctx, userID, rootUserID, "trade")
}

func (s *Service) filterText(ctx context.Context, value string) (string, error) {
	if s.filter == nil {
		return value, nil
	}
	return s.filter.FilterText(ctx, value)
}

func validateImages(images []string) error {
	if len(images) == 0 || len(images) > 9 {
		return bizerr.Param("商品图片数量必须为1到9张")
	}
	for _, image := range images {
		image = strings.TrimSpace(image)
		decoded, err := hex.DecodeString(image)
		if err != nil || len(decoded) != 16 {
			return bizerr.Param("商品图片引用不合法")
		}
	}
	return nil
}

func parseMarketplaceID(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, bizerr.Param("ID格式错误")
	}
	return value, nil
}

func normalizeMarketplacePage(page, size int) (int, int) {
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
