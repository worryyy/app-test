package marketplace

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/snowflake"
)

const (
	orderPaymentWindow = 15 * time.Minute
	autoCompleteDelay  = 48 * time.Hour
)

func (s *Service) CreateOrder(ctx context.Context, userID, rootUserID int64, req CreateOrderReq) (*OrderResponse, error) {
	if err := s.checkTrade(ctx, userID, rootUserID); err != nil {
		return nil, err
	}
	itemID, err := parseMarketplaceID(req.ItemID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	order := &Order{ID: snowflake.Generate().Int64(), ItemID: itemID, BuyerRootUserID: rootUserID, BuyerUserID: userID, Status: OrderPendingPayment, ExpiresAt: now.Add(orderPaymentWindow), CreatedAt: now, UpdatedAt: now}
	err = s.repo.CreateOrder(ctx, order)
	switch {
	case errors.Is(err, errOwnItem):
		return nil, ErrOwnItem
	case errors.Is(err, errItemUnavailable):
		return nil, ErrItemUnavailable
	case err != nil:
		return nil, bizerr.InternalWrap("创建订单失败", err)
	}
	stored, err := s.repo.FindOrder(ctx, order.ID)
	if err != nil || stored == nil {
		return nil, bizerr.InternalWrap("查询订单失败", err)
	}
	s.notify(ctx, stored.SellerRootUserID, "marketplace.order.created", "商品已被预订", stored.Item.Title, marketplaceID(stored.ID))
	response := orderResponse(*stored)
	return &response, nil
}

func (s *Service) ListOrders(ctx context.Context, rootUserID int64, admin bool, page, size int) (*pagination.PageResult[OrderResponse], error) {
	page, size = normalizeMarketplacePage(page, size)
	items, total, err := s.repo.ListOrders(ctx, rootUserID, admin, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询订单失败", err)
	}
	responses := make([]OrderResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, orderResponse(item))
	}
	return pagination.NewPageResult(responses, total, page, size), nil
}

func (s *Service) StartPayment(ctx context.Context, rootUserID int64, orderID string, req PaymentReq) (*PaymentResponse, error) {
	if s.gateway.Name() == "disabled" {
		return nil, ErrPaymentDisabled
	}
	id, err := parseMarketplaceID(orderID)
	if err != nil {
		return nil, err
	}
	order, err := s.repo.FindOrder(ctx, id)
	if err != nil || order == nil || order.BuyerRootUserID != rootUserID {
		return nil, ErrOrderNotFound
	}
	requestID := strings.TrimSpace(req.RequestID)
	if requestID == "" {
		return nil, bizerr.Param("支付请求号不能为空")
	}
	now := s.now()
	payment := &Payment{ID: snowflake.Generate().Int64(), OrderID: id, Gateway: s.gateway.Name(), RequestID: requestID, AmountCents: order.PriceCents, Status: RecordPending, CreatedAt: now, UpdatedAt: now}
	payment, _, err = s.repo.CreateOrFindPayment(ctx, payment)
	if errors.Is(err, errOrderState) {
		return nil, ErrOrderState
	}
	if err != nil {
		return nil, bizerr.InternalWrap("创建支付记录失败", err)
	}
	if payment.GatewayTransactionID == nil {
		transactionID, err := s.gateway.CreatePayment(ctx, payment.RequestID, payment.AmountCents)
		if err != nil {
			return nil, err
		}
		if err := s.repo.AttachPaymentTransaction(ctx, payment.ID, transactionID, s.now()); err != nil {
			return nil, bizerr.InternalWrap("保存支付流水号失败", err)
		}
		payment.GatewayTransactionID = &transactionID
	}
	return paymentResponse(*payment), nil
}

func (s *Service) TestPaymentCallback(ctx context.Context, req TestPaymentCallbackReq) error {
	if s.gateway.Name() != "test" {
		return ErrPaymentDisabled
	}
	order, changed, err := s.repo.ApplyPaymentCallback(ctx, strings.TrimSpace(req.RequestID), strings.TrimSpace(req.GatewayTransactionID), req.AmountCents, s.now())
	if errors.Is(err, errOrderState) {
		return ErrPaymentInvalid
	}
	if err != nil {
		return bizerr.InternalWrap("处理支付回调失败", err)
	}
	if changed {
		s.notify(ctx, order.BuyerRootUserID, "marketplace.payment.succeeded", "支付成功", order.Item.Title, marketplaceID(order.ID))
		s.notify(ctx, order.SellerRootUserID, "marketplace.payment.succeeded", "买家已付款", order.Item.Title, marketplaceID(order.ID))
	}
	return nil
}

func (s *Service) CancelOrder(ctx context.Context, rootUserID int64, orderID string) error {
	id, err := parseMarketplaceID(orderID)
	if err != nil {
		return err
	}
	order, err := s.repo.CancelOrder(ctx, id, rootUserID, s.now())
	if errors.Is(err, errOrderState) {
		return ErrOrderState
	}
	if err != nil {
		return bizerr.InternalWrap("取消订单失败", err)
	}
	s.notify(ctx, order.SellerRootUserID, "marketplace.order.cancelled", "订单已取消", order.Item.Title, marketplaceID(order.ID))
	return nil
}

func (s *Service) MarkDelivered(ctx context.Context, rootUserID int64, orderID string) error {
	id, err := parseMarketplaceID(orderID)
	if err != nil {
		return err
	}
	order, err := s.repo.MarkDelivered(ctx, id, rootUserID, s.now())
	if errors.Is(err, errOrderState) {
		return ErrOrderState
	}
	if err != nil {
		return bizerr.InternalWrap("确认交付失败", err)
	}
	s.notify(ctx, order.BuyerRootUserID, "marketplace.order.delivered", "卖家已确认交付", order.Item.Title, marketplaceID(order.ID))
	return nil
}

func (s *Service) ConfirmReceived(ctx context.Context, rootUserID int64, orderID string) error {
	id, err := parseMarketplaceID(orderID)
	if err != nil {
		return err
	}
	now := s.now()
	settlement := newSettlement(id, 0, now)
	order, err := s.repo.ConfirmReceived(ctx, id, rootUserID, settlement, now)
	if errors.Is(err, errOrderState) {
		return ErrOrderState
	}
	if err != nil {
		return bizerr.InternalWrap("确认收货失败", err)
	}
	_ = s.processSettlement(ctx, settlement.ID)
	s.notify(ctx, order.SellerRootUserID, "marketplace.order.completed", "订单已完成", order.Item.Title, marketplaceID(order.ID))
	return nil
}

func (s *Service) RequestRefund(ctx context.Context, rootUserID int64, orderID string, req RefundReq) (*RefundResponse, error) {
	id, err := parseMarketplaceID(orderID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	refund := &Refund{ID: snowflake.Generate().Int64(), OrderID: id, RequestedByRootUserID: rootUserID, RequestID: strings.TrimSpace(req.RequestID), Reason: strings.TrimSpace(req.Reason), Status: RecordPending, CreatedAt: now, UpdatedAt: now}
	if refund.RequestID == "" || refund.Reason == "" {
		return nil, bizerr.Param("退款参数错误")
	}
	err = s.repo.CreateRefund(ctx, refund)
	if errors.Is(err, errRefundExists) {
		return nil, ErrRefundExists
	}
	if errors.Is(err, errOrderState) {
		return nil, ErrOrderState
	}
	if err != nil {
		return nil, bizerr.InternalWrap("提交退款失败", err)
	}
	response := refundResponse(*refund)
	return &response, nil
}

func (s *Service) CreateDispute(ctx context.Context, rootUserID int64, orderID string, req DisputeReq) (*DisputeResponse, error) {
	id, err := parseMarketplaceID(orderID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	dispute := &Dispute{ID: snowflake.Generate().Int64(), OrderID: id, RaisedByRootUserID: rootUserID, Reason: strings.TrimSpace(req.Reason), Status: DisputePending, CreatedAt: now, UpdatedAt: now}
	if dispute.Reason == "" {
		return nil, bizerr.Param("纠纷原因不能为空")
	}
	err = s.repo.CreateDispute(ctx, dispute, now)
	if errors.Is(err, errDisputeExists) {
		return nil, ErrDisputeExists
	}
	if errors.Is(err, errOrderState) {
		return nil, ErrOrderState
	}
	if err != nil {
		return nil, bizerr.InternalWrap("提交纠纷失败", err)
	}
	response := disputeResponse(*dispute)
	return &response, nil
}

func paymentResponse(value Payment) *PaymentResponse {
	return &PaymentResponse{OrderID: marketplaceID(value.OrderID), RequestID: value.RequestID, Gateway: value.Gateway, GatewayTransactionID: stringValue(value.GatewayTransactionID), AmountCents: value.AmountCents, Status: value.Status}
}

func newSettlement(orderID, amountCents int64, now time.Time) *Settlement {
	return &Settlement{ID: snowflake.Generate().Int64(), OrderID: orderID, RequestID: "settlement:" + marketplaceID(orderID), AmountCents: amountCents, Status: RecordPending, CreatedAt: now, UpdatedAt: now}
}

func (s *Service) notify(ctx context.Context, rootUserID int64, eventType, title, content, resourceID string) {
	if s.notifier != nil {
		_ = s.notifier.NotifyMarketplace(ctx, rootUserID, eventType, title, content, resourceID)
	}
}
