package marketplace

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) CreateRefund(ctx context.Context, refund *Refund) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var order Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", refund.OrderID).Take(&order).Error; err != nil {
			return errOrderState
		}
		if order.BuyerRootUserID != refund.RequestedByRootUserID || (order.Status != OrderPaid && order.Status != OrderDelivered) {
			return errOrderState
		}
		refund.AmountCents = order.PriceCents
		if err := tx.Create(refund).Error; err != nil {
			if isUniqueError(err) {
				return errRefundExists
			}
			return err
		}
		return nil
	})
}

func (r *Repository) FindRefund(ctx context.Context, id int64) (*Refund, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var item Refund
	err = db.Where("id = ?", id).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, err
}

func (r *Repository) DecideRefund(ctx context.Context, id, adminID int64, approved bool, gatewayRefundID string, now time.Time) (*Order, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var order Order
	err = db.Transaction(func(tx *gorm.DB) error {
		var refund Refund
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&refund).Error; err != nil {
			return errOrderState
		}
		if refund.Status != RecordPending {
			return errOrderState
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", refund.OrderID).Take(&order).Error; err != nil {
			return errOrderState
		}
		status := RecordRejected
		updates := map[string]any{"status": status, "reviewed_by": adminID, "reviewed_at": now, "updated_at": now}
		if approved {
			status = RecordSucceeded
			updates["status"], updates["gateway_refund_id"] = status, gatewayRefundID
			if err := tx.Model(&Order{}).Where("id = ? AND status IN ?", order.ID, []string{OrderPaid, OrderDelivered, OrderDisputed}).Updates(map[string]any{"status": OrderRefunded, "refunded_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
			if err := tx.Model(&Item{}).Where("id = ?", order.ItemID).Updates(map[string]any{"status": ItemWithdrawn, "reserved_order_id": nil, "updated_at": now}).Error; err != nil {
				return err
			}
			order.Status, order.RefundedAt = OrderRefunded, &now
		}
		if err := tx.Model(&Refund{}).Where("id = ? AND status = ?", refund.ID, RecordPending).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Preload("Item.Category").Where("id = ?", order.ID).Take(&order).Error
	})
	return &order, err
}

func (r *Repository) CreateDispute(ctx context.Context, dispute *Dispute, now time.Time) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var order Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", dispute.OrderID).Take(&order).Error; err != nil {
			return errOrderState
		}
		if (order.BuyerRootUserID != dispute.RaisedByRootUserID && order.SellerRootUserID != dispute.RaisedByRootUserID) || (order.Status != OrderPaid && order.Status != OrderDelivered) {
			return errOrderState
		}
		var pendingRefunds int64
		if err := tx.Model(&Refund{}).Where("order_id = ? AND status = ?", order.ID, RecordPending).Count(&pendingRefunds).Error; err != nil {
			return err
		}
		if pendingRefunds > 0 {
			return errOrderState
		}
		dispute.PreviousOrderStatus = order.Status
		if err := tx.Create(dispute).Error; err != nil {
			if isUniqueError(err) {
				return errDisputeExists
			}
			return err
		}
		return tx.Model(&Order{}).Where("id = ? AND status = ?", order.ID, order.Status).Updates(map[string]any{"status": OrderDisputed, "disputed_at": now, "updated_at": now}).Error
	})
}

func (r *Repository) FindDispute(ctx context.Context, id int64) (*Dispute, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var item Dispute
	err = db.Where("id = ?", id).Take(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, err
}

func (r *Repository) ResolveDispute(ctx context.Context, id, adminID int64, action, resolution, gatewayRefundID string, refund *Refund, now time.Time) (*Order, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}
	var order Order
	err = db.Transaction(func(tx *gorm.DB) error {
		var dispute Dispute
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ?", id, DisputePending).Take(&dispute).Error; err != nil {
			return errOrderState
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ?", dispute.OrderID, OrderDisputed).Take(&order).Error; err != nil {
			return errOrderState
		}
		nextStatus := dispute.PreviousOrderStatus
		if action == "refund" {
			nextStatus = OrderRefunded
			refund.OrderID, refund.RequestedByRootUserID, refund.AmountCents = order.ID, dispute.RaisedByRootUserID, order.PriceCents
			refund.GatewayRefundID, refund.Status = &gatewayRefundID, RecordSucceeded
			admin := adminID
			refund.ReviewedBy, refund.ReviewedAt = &admin, &now
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "order_id"}}, DoNothing: true}).Create(refund).Error; err != nil {
				return err
			}
			if err := tx.Model(&Item{}).Where("id = ?", order.ItemID).Updates(map[string]any{"status": ItemWithdrawn, "reserved_order_id": nil, "updated_at": now}).Error; err != nil {
				return err
			}
		}
		orderUpdates := map[string]any{"status": nextStatus, "updated_at": now}
		if nextStatus == OrderRefunded {
			orderUpdates["refunded_at"] = now
		}
		if err := tx.Model(&Order{}).Where("id = ? AND status = ?", order.ID, OrderDisputed).Updates(orderUpdates).Error; err != nil {
			return err
		}
		status := DisputeRejected
		if action == "refund" {
			status = DisputeResolved
		}
		if err := tx.Model(&Dispute{}).Where("id = ? AND status = ?", dispute.ID, DisputePending).Updates(map[string]any{"status": status, "resolution": resolution, "resolved_by": adminID, "resolved_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		order.Status, order.UpdatedAt = nextStatus, now
		return tx.Preload("Item.Category").Where("id = ?", order.ID).Take(&order).Error
	})
	return &order, err
}

func (r *Repository) ListRefunds(ctx context.Context, page, size int) ([]Refund, int64, error) {
	return listRecords[Refund](ctx, r, page, size)
}
func (r *Repository) ListDisputes(ctx context.Context, page, size int) ([]Dispute, int64, error) {
	return listRecords[Dispute](ctx, r, page, size)
}
func (r *Repository) ListSettlements(ctx context.Context, page, size int) ([]Settlement, int64, error) {
	return listRecords[Settlement](ctx, r, page, size)
}

func listRecords[T any](ctx context.Context, r *Repository, page, size int) ([]T, int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if err := db.Model(new(T)).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := []T{}
	err = db.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

func isUniqueError(err error) bool {
	if err == nil {
		return false
	}
	value := strings.ToLower(err.Error())
	return strings.Contains(value, "unique") || strings.Contains(value, "duplicate")
}
