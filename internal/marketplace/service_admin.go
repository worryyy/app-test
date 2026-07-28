package marketplace

import (
	"context"
	"errors"
	"strings"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/snowflake"
)

func (s *Service) ListRefunds(ctx context.Context, page, size int) (*pagination.PageResult[RefundResponse], error) {
	page, size = normalizeMarketplacePage(page, size)
	items, total, err := s.repo.ListRefunds(ctx, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询退款失败", err)
	}
	responses := make([]RefundResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, refundResponse(item))
	}
	return pagination.NewPageResult(responses, total, page, size), nil
}

func (s *Service) DecideRefund(ctx context.Context, refundID string, adminID int64, req RefundDecisionReq) error {
	id, err := parseMarketplaceID(refundID)
	if err != nil {
		return err
	}
	refund, err := s.repo.FindRefund(ctx, id)
	if err != nil || refund == nil {
		return ErrOrderState
	}
	if refund.Status == RecordSucceeded && req.Approved {
		return nil
	}
	gatewayID := ""
	if req.Approved {
		gatewayID, err = s.gateway.Refund(ctx, refund.RequestID, refund.AmountCents)
		if err != nil {
			return err
		}
	}
	order, err := s.repo.DecideRefund(ctx, id, adminID, req.Approved, gatewayID, s.now())
	if errors.Is(err, errOrderState) {
		return ErrOrderState
	}
	if err != nil {
		return bizerr.InternalWrap("处理退款失败", err)
	}
	event, title := "marketplace.refund.rejected", "退款申请已驳回"
	if req.Approved {
		event, title = "marketplace.refund.succeeded", "退款已完成"
	}
	s.notify(ctx, order.BuyerRootUserID, event, title, order.Item.Title, marketplaceID(order.ID))
	return nil
}

func (s *Service) ListDisputes(ctx context.Context, page, size int) (*pagination.PageResult[DisputeResponse], error) {
	page, size = normalizeMarketplacePage(page, size)
	items, total, err := s.repo.ListDisputes(ctx, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询纠纷失败", err)
	}
	responses := make([]DisputeResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, disputeResponse(item))
	}
	return pagination.NewPageResult(responses, total, page, size), nil
}

func (s *Service) ResolveDispute(ctx context.Context, disputeID string, adminID int64, req DisputeDecisionReq) error {
	id, err := parseMarketplaceID(disputeID)
	if err != nil {
		return err
	}
	dispute, err := s.repo.FindDispute(ctx, id)
	if err != nil || dispute == nil || dispute.Status != DisputePending {
		return ErrOrderState
	}
	order, err := s.repo.FindOrder(ctx, dispute.OrderID)
	if err != nil || order == nil {
		return ErrOrderNotFound
	}
	now := s.now()
	refund := &Refund{ID: snowflake.Generate().Int64(), RequestID: "dispute-refund:" + marketplaceID(id), Reason: strings.TrimSpace(req.Resolution), CreatedAt: now, UpdatedAt: now}
	gatewayID := ""
	if req.Action == "refund" {
		gatewayID, err = s.gateway.Refund(ctx, refund.RequestID, order.PriceCents)
		if err != nil {
			return err
		}
	}
	order, err = s.repo.ResolveDispute(ctx, id, adminID, req.Action, strings.TrimSpace(req.Resolution), gatewayID, refund, now)
	if errors.Is(err, errOrderState) {
		return ErrOrderState
	}
	if err != nil {
		return bizerr.InternalWrap("处理纠纷失败", err)
	}
	s.notify(ctx, order.BuyerRootUserID, "marketplace.dispute.resolved", "纠纷已处理", req.Resolution, marketplaceID(order.ID))
	s.notify(ctx, order.SellerRootUserID, "marketplace.dispute.resolved", "纠纷已处理", req.Resolution, marketplaceID(order.ID))
	return nil
}

func (s *Service) ListSettlements(ctx context.Context, page, size int) (*pagination.PageResult[SettlementResponse], error) {
	page, size = normalizeMarketplacePage(page, size)
	items, total, err := s.repo.ListSettlements(ctx, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询结算失败", err)
	}
	responses := make([]SettlementResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, settlementResponse(item))
	}
	return pagination.NewPageResult(responses, total, page, size), nil
}

func (s *Service) RetrySettlement(ctx context.Context, settlementID string) error {
	id, err := parseMarketplaceID(settlementID)
	if err != nil {
		return err
	}
	if err := s.processSettlement(ctx, id); errors.Is(err, errOrderState) {
		return ErrSettlementState
	} else if err != nil {
		return err
	}
	return nil
}

func (s *Service) ReportTarget(ctx context.Context, targetID string) (int64, any, error) {
	id, err := parseMarketplaceID(targetID)
	if err != nil {
		return 0, nil, err
	}
	item, err := s.repo.FindItem(ctx, id)
	if err != nil || item == nil {
		return 0, nil, ErrItemNotFound
	}
	response := itemResponse(*item)
	return item.SellerRootUserID, response, nil
}

func (s *Service) HideForModeration(ctx context.Context, targetID string) error {
	id, err := parseMarketplaceID(targetID)
	if err != nil {
		return err
	}
	order, err := s.repo.HideItem(ctx, id, s.now())
	if err != nil {
		return bizerr.InternalWrap("隐藏商品失败", err)
	}
	if order != nil {
		s.notify(ctx, order.BuyerRootUserID, "marketplace.order.cancelled", "订单因商品下架取消", "商品已由管理员下架", marketplaceID(order.ID))
	}
	return nil
}
