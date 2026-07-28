package marketplace

import (
	"context"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

func (s *Service) RunDueJobs(ctx context.Context) error {
	now := s.now()
	expiredIDs, err := s.repo.ListExpiredOrderIDs(ctx, now, 100)
	if err != nil {
		return bizerr.InternalWrap("查询超时订单失败", err)
	}
	for _, id := range expiredIDs {
		order, changed, err := s.repo.ExpireOrder(ctx, id, now)
		if err != nil {
			return bizerr.InternalWrap("释放超时订单失败", err)
		}
		if changed {
			s.notify(ctx, order.BuyerRootUserID, "marketplace.order.expired", "订单已超时取消", order.Item.Title, marketplaceID(order.ID))
			s.notify(ctx, order.SellerRootUserID, "marketplace.order.expired", "商品预订已释放", order.Item.Title, marketplaceID(order.ID))
		}
	}

	cutoff := now.Add(-autoCompleteDelay)
	completeIDs, err := s.repo.ListAutoCompleteOrderIDs(ctx, cutoff, 100)
	if err != nil {
		return bizerr.InternalWrap("查询待完成订单失败", err)
	}
	for _, id := range completeIDs {
		settlement := newSettlement(id, 0, now)
		order, changed, err := s.repo.AutoCompleteOrder(ctx, id, cutoff, settlement, now)
		if err != nil {
			return bizerr.InternalWrap("自动完成订单失败", err)
		}
		if changed {
			if err := s.processSettlement(ctx, settlement.ID); err != nil {
				s.logger.Warn("marketplace settlement failed", zap.Int64("settlementId", settlement.ID), zap.Error(err))
			}
			s.notify(ctx, order.BuyerRootUserID, "marketplace.order.completed", "订单已自动完成", order.Item.Title, marketplaceID(order.ID))
			s.notify(ctx, order.SellerRootUserID, "marketplace.order.completed", "订单已自动完成", order.Item.Title, marketplaceID(order.ID))
		}
	}
	return nil
}

func (s *Service) RetryPendingSettlements(ctx context.Context) {
	ids, err := s.repo.ListRetrySettlementIDs(ctx, 100)
	if err != nil {
		s.logger.Warn("list marketplace settlements failed", zap.Error(err))
		return
	}
	for _, id := range ids {
		if err := s.processSettlement(ctx, id); err != nil {
			s.logger.Warn("retry marketplace settlement failed", zap.Int64("settlementId", id), zap.Error(err))
		}
	}
}

func (s *Service) processSettlement(ctx context.Context, id int64) error {
	settlement, err := s.repo.FindSettlement(ctx, id)
	if err != nil || settlement == nil {
		return errOrderState
	}
	if settlement.Status == RecordSucceeded {
		return nil
	}
	gatewayID, err := s.gateway.Settle(ctx, settlement.RequestID, settlement.AmountCents)
	if err != nil {
		_, _ = s.repo.UpdateSettlement(ctx, id, RecordFailed, "", s.now())
		return err
	}
	ok, err := s.repo.UpdateSettlement(ctx, id, RecordSucceeded, gatewayID, s.now())
	if err != nil {
		return bizerr.InternalWrap("更新结算状态失败", err)
	}
	if !ok {
		current, findErr := s.repo.FindSettlement(ctx, id)
		if findErr == nil && current != nil && current.Status == RecordSucceeded {
			return nil
		}
		return errOrderState
	}
	return nil
}
