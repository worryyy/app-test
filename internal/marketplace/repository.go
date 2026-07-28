package marketplace

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	errItemUnavailable = errors.New("item unavailable")
	errOwnItem         = errors.New("cannot buy own item")
	errOrderState      = errors.New("invalid order state")
	errRefundExists    = errors.New("refund exists")
	errDisputeExists   = errors.New("dispute exists")
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) gormDB(ctx context.Context) (*gorm.DB, error) {
	if r.db == nil {
		return nil, errors.New("marketplace database not initialized")
	}
	return r.db.WithContext(ctx), nil
}

func (r *Repository) ListCategories(ctx context.Context, includeDisabled bool) ([]Category, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	query := db.Order("name")
	if !includeDisabled {
		query = query.Where("status = ?", StatusActive)
	}
	items := []Category{}
	return items, query.Find(&items).Error
}

func (r *Repository) FindCategory(ctx context.Context, id int64) (*Category, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var item Category
	err = db.Where("id = ?", id).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, err
}

func (r *Repository) SaveCategory(ctx context.Context, item *Category) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	return db.Save(item).Error
}

func (r *Repository) CreateItem(ctx context.Context, item *Item) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	return db.Create(item).Error
}

func (r *Repository) FindItem(ctx context.Context, id int64) (*Item, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var item Item
	err = db.Preload("Category").Where("id = ?", id).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, err
}

func (r *Repository) SearchItems(ctx context.Context, rootUserID int64, mine bool, keyword string, categoryID int64, page, size int) ([]Item, int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, 0, err
	}
	base := db.Model(&Item{})
	if mine {
		base = base.Where("seller_root_user_id = ?", rootUserID)
	} else {
		base = base.Where("status = ?", ItemPublished)
	}
	if categoryID > 0 {
		base = base.Where("category_id = ?", categoryID)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where("title LIKE ? OR description LIKE ?", like, like)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := []Item{}
	err = base.Preload("Category").Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

func (r *Repository) UpdateItem(ctx context.Context, item *Item, ownerRootUserID int64) (bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}
	result := db.Model(&Item{}).
		Where("id = ? AND seller_root_user_id = ? AND status IN ?", item.ID, ownerRootUserID, []string{ItemDraft, ItemPublished}).
		Select("CategoryID", "Title", "Description", "Condition", "PriceCents", "Images", "DeliveryLocation", "Status", "UpdatedAt").
		Updates(item)
	return result.RowsAffected == 1, result.Error
}

func (r *Repository) WithdrawItem(ctx context.Context, id, ownerRootUserID int64, now time.Time) (bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}
	result := db.Model(&Item{}).Where("id = ? AND seller_root_user_id = ? AND status IN ?", id, ownerRootUserID, []string{ItemDraft, ItemPublished, ItemHidden}).Updates(map[string]any{"status": ItemWithdrawn, "updated_at": now})
	return result.RowsAffected == 1, result.Error
}

func (r *Repository) HideItem(ctx context.Context, id int64, now time.Time) (*Order, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var cancelled *Order
	err = db.Transaction(func(tx *gorm.DB) error {
		var item Item
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&item).Error; err != nil {
			return err
		}
		if item.Status == ItemReserved && item.ReservedOrderID != nil {
			var order Order
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ?", *item.ReservedOrderID, OrderPendingPayment).Take(&order).Error; err == nil {
				if err := tx.Model(&Order{}).Where("id = ? AND status = ?", order.ID, OrderPendingPayment).Updates(map[string]any{"status": OrderCancelled, "cancelled_at": now, "updated_at": now}).Error; err != nil {
					return err
				}
				cancelled = &order
			}
		}
		return tx.Model(&Item{}).Where("id = ?", id).Updates(map[string]any{"status": ItemHidden, "reserved_order_id": nil, "updated_at": now}).Error
	})
	return cancelled, err
}
