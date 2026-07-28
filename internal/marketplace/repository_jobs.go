package marketplace

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) ListExpiredOrderIDs(ctx context.Context, now time.Time, limit int) ([]int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	ids := []int64{}
	err = db.Model(&Order{}).Where("status = ? AND expires_at <= ?", OrderPendingPayment, now).Order("id").Limit(limit).Pluck("id", &ids).Error
	return ids, err
}

func (r *Repository) ExpireOrder(ctx context.Context, id int64, now time.Time) (*Order, bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, false, err
	}
	var order Order
	changed := false
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if order.Status != OrderPendingPayment || order.ExpiresAt.After(now) {
			return nil
		}
		result := tx.Model(&Order{}).Where("id = ? AND status = ?", id, OrderPendingPayment).Updates(map[string]any{"status": OrderCancelled, "cancelled_at": now, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		if err := tx.Model(&Item{}).Where("id = ? AND reserved_order_id = ?", order.ItemID, order.ID).Updates(map[string]any{"status": ItemPublished, "reserved_order_id": nil, "updated_at": now}).Error; err != nil {
			return err
		}
		changed = true
		return tx.Preload("Item.Category").Where("id = ?", id).Take(&order).Error
	})
	return &order, changed, err
}

func (r *Repository) ListAutoCompleteOrderIDs(ctx context.Context, cutoff time.Time, limit int) ([]int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	ids := []int64{}
	err = db.Model(&Order{}).
		Where("status = ? AND delivered_at <= ?", OrderDelivered, cutoff).
		Where("NOT EXISTS (SELECT 1 FROM marketplace_refunds r WHERE r.order_id = marketplace_orders.id AND r.status = ?)", RecordPending).
		Order("id").Limit(limit).Pluck("id", &ids).Error
	return ids, err
}

func (r *Repository) AutoCompleteOrder(ctx context.Context, id int64, cutoff time.Time, settlement *Settlement, now time.Time) (*Order, bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, false, err
	}
	var order Order
	changed := false
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if order.Status != OrderDelivered || order.DeliveredAt == nil || order.DeliveredAt.After(cutoff) {
			return nil
		}
		settlement.OrderID, settlement.AmountCents = order.ID, order.SellerNetCents
		if err := completeOrderTx(tx, &order, settlement, now); err != nil {
			return err
		}
		changed = true
		return tx.Preload("Item.Category").Where("id = ?", id).Take(&order).Error
	})
	return &order, changed, err
}

func (r *Repository) FindSettlement(ctx context.Context, id int64) (*Settlement, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var item Settlement
	err = db.Where("id = ?", id).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, err
}

func (r *Repository) UpdateSettlement(ctx context.Context, id int64, status, gatewayID string, now time.Time) (bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}
	updates := map[string]any{"status": status, "updated_at": now}
	if gatewayID != "" {
		updates["gateway_settlement_id"] = gatewayID
	}
	result := db.Model(&Settlement{}).Where("id = ? AND status IN ?", id, []string{RecordPending, RecordFailed}).Updates(updates)
	return result.RowsAffected == 1, result.Error
}

func (r *Repository) ListRetrySettlementIDs(ctx context.Context, limit int) ([]int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	ids := []int64{}
	err = db.Model(&Settlement{}).Where("status IN ?", []string{RecordPending, RecordFailed}).Order("created_at").Limit(limit).Pluck("id", &ids).Error
	return ids, err
}
