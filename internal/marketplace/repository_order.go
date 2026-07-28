package marketplace

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) CreateOrder(ctx context.Context, order *Order) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var item Item
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", order.ItemID).Take(&item).Error; err != nil {
			return errItemUnavailable
		}
		if item.Status != ItemPublished || item.ReservedOrderID != nil {
			return errItemUnavailable
		}
		if item.SellerRootUserID == order.BuyerRootUserID {
			return errOwnItem
		}
		var category Category
		if err := tx.Where("id = ? AND status = ?", item.CategoryID, StatusActive).Take(&category).Error; err != nil {
			return errItemUnavailable
		}
		order.SellerRootUserID = item.SellerRootUserID
		order.PriceCents = item.PriceCents
		order.CommissionRateBps = category.CommissionRateBps
		order.PlatformFeeCents = commissionFee(item.PriceCents, category.CommissionRateBps)
		order.SellerNetCents = item.PriceCents - order.PlatformFeeCents
		result := tx.Model(&Item{}).Where("id = ? AND status = ? AND reserved_order_id IS NULL", item.ID, ItemPublished).Updates(map[string]any{"status": ItemReserved, "reserved_order_id": order.ID, "updated_at": order.UpdatedAt})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errItemUnavailable
		}
		return tx.Create(order).Error
	})
}

func (r *Repository) FindOrder(ctx context.Context, id int64) (*Order, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var item Order
	err = db.Preload("Item.Category").Where("id = ?", id).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, err
}

func (r *Repository) ListOrders(ctx context.Context, rootUserID int64, admin bool, page, size int) ([]Order, int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, 0, err
	}
	base := db.Model(&Order{})
	if !admin {
		base = base.Where("buyer_root_user_id = ? OR seller_root_user_id = ?", rootUserID, rootUserID)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := []Order{}
	err = base.Preload("Item.Category").Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

func (r *Repository) CreateOrFindPayment(ctx context.Context, payment *Payment) (*Payment, bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, false, err
	}
	created := false
	err = db.Transaction(func(tx *gorm.DB) error {
		var order Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", payment.OrderID).Take(&order).Error; err != nil {
			return errOrderState
		}
		if order.Status != OrderPendingPayment || !payment.CreatedAt.Before(order.ExpiresAt) {
			return errOrderState
		}
		var existing Payment
		err := tx.Where("request_id = ?", payment.RequestID).Take(&existing).Error
		if err == nil {
			if existing.OrderID != payment.OrderID || existing.AmountCents != payment.AmountCents {
				return errOrderState
			}
			*payment = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Create(payment).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	return payment, created, err
}

func (r *Repository) AttachPaymentTransaction(ctx context.Context, paymentID int64, transactionID string, now time.Time) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	result := db.Model(&Payment{}).Where("id = ? AND (gateway_transaction_id IS NULL OR gateway_transaction_id = ?)", paymentID, transactionID).Updates(map[string]any{"gateway_transaction_id": transactionID, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errOrderState
	}
	return nil
}

func (r *Repository) ApplyPaymentCallback(ctx context.Context, requestID, transactionID string, amountCents int64, now time.Time) (*Order, bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, false, err
	}
	var order Order
	changed := false
	err = db.Transaction(func(tx *gorm.DB) error {
		var payment Payment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("request_id = ?", requestID).Take(&payment).Error; err != nil {
			return errOrderState
		}
		if payment.AmountCents != amountCents || (payment.GatewayTransactionID != nil && *payment.GatewayTransactionID != transactionID) {
			return errOrderState
		}
		if payment.Status == RecordSucceeded {
			return tx.Preload("Item.Category").Where("id = ?", payment.OrderID).Take(&order).Error
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", payment.OrderID).Take(&order).Error; err != nil {
			return errOrderState
		}
		if order.Status != OrderPendingPayment || !now.Before(order.ExpiresAt) {
			return errOrderState
		}
		if err := tx.Model(&Payment{}).Where("id = ? AND status = ?", payment.ID, RecordPending).Updates(map[string]any{"gateway_transaction_id": transactionID, "status": RecordSucceeded, "updated_at": now}).Error; err != nil {
			if isUniqueError(err) {
				return errOrderState
			}
			return err
		}
		result := tx.Model(&Order{}).Where("id = ? AND status = ?", order.ID, OrderPendingPayment).Updates(map[string]any{"status": OrderPaid, "paid_at": now, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errOrderState
		}
		order.Status, order.PaidAt, order.UpdatedAt = OrderPaid, &now, now
		changed = true
		return tx.Preload("Item.Category").Where("id = ?", order.ID).Take(&order).Error
	})
	return &order, changed, err
}

func (r *Repository) CancelOrder(ctx context.Context, id, buyerRootUserID int64, now time.Time) (*Order, error) {
	return r.transitionOrder(ctx, id, func(tx *gorm.DB, order *Order) error {
		if order.BuyerRootUserID != buyerRootUserID || order.Status != OrderPendingPayment {
			return errOrderState
		}
		if err := tx.Model(&Order{}).Where("id = ? AND status = ?", id, OrderPendingPayment).Updates(map[string]any{"status": OrderCancelled, "cancelled_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&Item{}).Where("id = ? AND reserved_order_id = ?", order.ItemID, order.ID).Updates(map[string]any{"status": ItemPublished, "reserved_order_id": nil, "updated_at": now}).Error
	})
}

func (r *Repository) MarkDelivered(ctx context.Context, id, sellerRootUserID int64, now time.Time) (*Order, error) {
	return r.transitionOrder(ctx, id, func(tx *gorm.DB, order *Order) error {
		if order.SellerRootUserID != sellerRootUserID || order.Status != OrderPaid {
			return errOrderState
		}
		return tx.Model(&Order{}).Where("id = ? AND status = ?", id, OrderPaid).Updates(map[string]any{"status": OrderDelivered, "delivered_at": now, "updated_at": now}).Error
	})
}

func (r *Repository) ConfirmReceived(ctx context.Context, id, buyerRootUserID int64, settlement *Settlement, now time.Time) (*Order, error) {
	return r.transitionOrder(ctx, id, func(tx *gorm.DB, order *Order) error {
		if order.BuyerRootUserID != buyerRootUserID || order.Status != OrderDelivered {
			return errOrderState
		}
		settlement.OrderID, settlement.AmountCents = order.ID, order.SellerNetCents
		if err := completeOrderTx(tx, order, settlement, now); err != nil {
			return err
		}
		return nil
	})
}

func (r *Repository) transitionOrder(ctx context.Context, id int64, transition func(*gorm.DB, *Order) error) (*Order, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var order Order
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&order).Error; err != nil {
			return errOrderState
		}
		if err := transition(tx, &order); err != nil {
			return err
		}
		return tx.Preload("Item.Category").Where("id = ?", id).Take(&order).Error
	})
	return &order, err
}

func completeOrderTx(tx *gorm.DB, order *Order, settlement *Settlement, now time.Time) error {
	var pendingRefunds int64
	if err := tx.Model(&Refund{}).Where("order_id = ? AND status = ?", order.ID, RecordPending).Count(&pendingRefunds).Error; err != nil {
		return err
	}
	if pendingRefunds > 0 {
		return errOrderState
	}
	result := tx.Model(&Order{}).Where("id = ? AND status = ?", order.ID, OrderDelivered).Updates(map[string]any{"status": OrderCompleted, "completed_at": now, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errOrderState
	}
	if err := tx.Model(&Item{}).Where("id = ? AND reserved_order_id = ?", order.ItemID, order.ID).Updates(map[string]any{"status": ItemSold, "updated_at": now}).Error; err != nil {
		return err
	}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "order_id"}}, DoNothing: true}).Create(settlement).Error
}

func commissionFee(priceCents int64, rateBps int) int64 {
	rate := int64(rateBps)
	return (priceCents/10000)*rate + (priceCents%10000)*rate/10000
}
