package marketplace

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type sellerVerifierStub struct{}

func (sellerVerifierStub) VerifyMarketplaceSeller(context.Context, int64) error { return nil }

func TestOrderPaymentAndCommissionSnapshot(t *testing.T) {
	svc, db, now := marketplaceTestService(t, NewGateway("test"))
	category := createMarketplaceCategory(t, svc, 500)
	item := createPublishedItem(t, svc, category.ID, 999)

	if _, err := svc.CreateOrder(context.Background(), 11, 10, CreateOrderReq{ItemID: item.ID}); !errors.Is(err, ErrOwnItem) {
		t.Fatalf("self purchase error = %v", err)
	}
	order, err := svc.CreateOrder(context.Background(), 101, 100, CreateOrderReq{ItemID: item.ID})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.PlatformFeeCents != 49 || order.SellerNetCents != 950 || order.CommissionRateBps != 500 {
		t.Fatalf("commission snapshot = %+v", order)
	}
	if _, err := svc.CreateOrder(context.Background(), 201, 200, CreateOrderReq{ItemID: item.ID}); !errors.Is(err, ErrItemUnavailable) {
		t.Fatalf("second order error = %v", err)
	}

	updatedRate := 1000
	if _, err := svc.SaveCategory(context.Background(), category.ID, CategoryReq{Name: category.Name, CommissionRateBps: &updatedRate}); err != nil {
		t.Fatal(err)
	}
	stored, err := svc.repo.FindOrder(context.Background(), mustMarketplaceID(t, order.ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored.CommissionRateBps != 500 || stored.PlatformFeeCents != 49 {
		t.Fatalf("historical commission changed: %+v", stored)
	}

	payment, err := svc.StartPayment(context.Background(), 100, order.ID, PaymentReq{RequestID: "pay-1"})
	if err != nil {
		t.Fatalf("start payment: %v", err)
	}
	duplicate, err := svc.StartPayment(context.Background(), 100, order.ID, PaymentReq{RequestID: "pay-1"})
	if err != nil || duplicate.GatewayTransactionID != payment.GatewayTransactionID {
		t.Fatalf("duplicate payment = %+v, err = %v", duplicate, err)
	}
	callback := TestPaymentCallbackReq{RequestID: payment.RequestID, GatewayTransactionID: payment.GatewayTransactionID, AmountCents: payment.AmountCents}
	if err := svc.TestPaymentCallback(context.Background(), callback); err != nil {
		t.Fatalf("payment callback: %v", err)
	}
	if err := svc.TestPaymentCallback(context.Background(), callback); err != nil {
		t.Fatalf("duplicate callback: %v", err)
	}
	var paymentCount int64
	if err := db.Model(&Payment{}).Count(&paymentCount).Error; err != nil || paymentCount != 1 {
		t.Fatalf("payment count = %d, err = %v", paymentCount, err)
	}
	if err := svc.MarkDelivered(context.Background(), 10, order.ID); err != nil {
		t.Fatalf("mark delivered: %v", err)
	}
	if err := svc.ConfirmReceived(context.Background(), 100, order.ID); err != nil {
		t.Fatalf("confirm received: %v", err)
	}
	if err := svc.ConfirmReceived(context.Background(), 100, order.ID); !errors.Is(err, ErrOrderState) {
		t.Fatalf("second confirm error = %v", err)
	}
	var settlement Settlement
	if err := db.Where("order_id = ?", mustMarketplaceID(t, order.ID)).Take(&settlement).Error; err != nil {
		t.Fatal(err)
	}
	if settlement.Status != RecordSucceeded || settlement.AmountCents != 950 || settlement.GatewaySettlementID == nil {
		t.Fatalf("settlement = %+v", settlement)
	}
	if now.IsZero() {
		t.Fatal("test time is zero")
	}
}

func TestDueJobsReleaseAndAutoCompleteIdempotently(t *testing.T) {
	svc, db, now := marketplaceTestService(t, NewGateway("test"))
	category := createMarketplaceCategory(t, svc, 500)
	expiringItem := createPublishedItem(t, svc, category.ID, 1000)
	expiringOrder, err := svc.CreateOrder(context.Background(), 101, 100, CreateOrderReq{ItemID: expiringItem.ID})
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return now.Add(orderPaymentWindow) }
	if err := svc.RunDueJobs(context.Background()); err != nil {
		t.Fatalf("expire jobs: %v", err)
	}
	storedOrder, _ := svc.repo.FindOrder(context.Background(), mustMarketplaceID(t, expiringOrder.ID))
	storedItem, _ := svc.repo.FindItem(context.Background(), mustMarketplaceID(t, expiringItem.ID))
	if storedOrder.Status != OrderCancelled || storedItem.Status != ItemPublished || storedItem.ReservedOrderID != nil {
		t.Fatalf("expired order=%+v item=%+v", storedOrder, storedItem)
	}

	svc.now = func() time.Time { return now }
	completeItem := createPublishedItem(t, svc, category.ID, 1000)
	completeOrder, err := svc.CreateOrder(context.Background(), 201, 200, CreateOrderReq{ItemID: completeItem.ID})
	if err != nil {
		t.Fatal(err)
	}
	payOrder(t, svc, completeOrder, 200, "pay-auto")
	if err := svc.MarkDelivered(context.Background(), 10, completeOrder.ID); err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return now.Add(autoCompleteDelay) }
	if err := svc.RunDueJobs(context.Background()); err != nil {
		t.Fatalf("auto complete jobs: %v", err)
	}
	if err := svc.RunDueJobs(context.Background()); err != nil {
		t.Fatalf("repeat jobs: %v", err)
	}
	storedOrder, _ = svc.repo.FindOrder(context.Background(), mustMarketplaceID(t, completeOrder.ID))
	if storedOrder.Status != OrderCompleted {
		t.Fatalf("auto completed status = %s", storedOrder.Status)
	}
	var count int64
	if err := db.Model(&Settlement{}).Where("order_id = ?", mustMarketplaceID(t, completeOrder.ID)).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("settlement count = %d, err = %v", count, err)
	}
}

func TestRefundAndDisputeFreeze(t *testing.T) {
	svc, _, now := marketplaceTestService(t, NewGateway("test"))
	category := createMarketplaceCategory(t, svc, 500)
	refundItem := createPublishedItem(t, svc, category.ID, 2000)
	refundOrder, _ := svc.CreateOrder(context.Background(), 101, 100, CreateOrderReq{ItemID: refundItem.ID})
	payOrder(t, svc, refundOrder, 100, "pay-refund")
	if err := svc.MarkDelivered(context.Background(), 10, refundOrder.ID); err != nil {
		t.Fatal(err)
	}
	refund, err := svc.RequestRefund(context.Background(), 100, refundOrder.ID, RefundReq{RequestID: "refund-1", Reason: "changed mind"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RequestRefund(context.Background(), 100, refundOrder.ID, RefundReq{RequestID: "refund-2", Reason: "again"}); !errors.Is(err, ErrRefundExists) {
		t.Fatalf("duplicate refund error = %v", err)
	}
	svc.now = func() time.Time { return now.Add(autoCompleteDelay) }
	if err := svc.RunDueJobs(context.Background()); err != nil {
		t.Fatal(err)
	}
	storedRefundOrder, _ := svc.repo.FindOrder(context.Background(), mustMarketplaceID(t, refundOrder.ID))
	if storedRefundOrder.Status != OrderDelivered {
		t.Fatalf("pending refund did not freeze completion: %s", storedRefundOrder.Status)
	}
	if err := svc.DecideRefund(context.Background(), refund.ID, 9, RefundDecisionReq{Approved: true}); err != nil {
		t.Fatalf("approve refund: %v", err)
	}
	if err := svc.DecideRefund(context.Background(), refund.ID, 9, RefundDecisionReq{Approved: true}); err != nil {
		t.Fatalf("repeat approve refund: %v", err)
	}
	storedRefundOrder, _ = svc.repo.FindOrder(context.Background(), mustMarketplaceID(t, refundOrder.ID))
	if storedRefundOrder.Status != OrderRefunded {
		t.Fatalf("refund status = %s", storedRefundOrder.Status)
	}

	svc.now = func() time.Time { return now }
	disputeItem := createPublishedItem(t, svc, category.ID, 3000)
	disputeOrder, _ := svc.CreateOrder(context.Background(), 201, 200, CreateOrderReq{ItemID: disputeItem.ID})
	payOrder(t, svc, disputeOrder, 200, "pay-dispute")
	if err := svc.MarkDelivered(context.Background(), 10, disputeOrder.ID); err != nil {
		t.Fatal(err)
	}
	dispute, err := svc.CreateDispute(context.Background(), 200, disputeOrder.ID, DisputeReq{Reason: "not as described"})
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return now.Add(autoCompleteDelay + time.Hour) }
	if err := svc.RunDueJobs(context.Background()); err != nil {
		t.Fatal(err)
	}
	storedDisputeOrder, _ := svc.repo.FindOrder(context.Background(), mustMarketplaceID(t, disputeOrder.ID))
	if storedDisputeOrder.Status != OrderDisputed {
		t.Fatalf("dispute did not freeze completion: %s", storedDisputeOrder.Status)
	}
	if err := svc.ResolveDispute(context.Background(), dispute.ID, 9, DisputeDecisionReq{Action: "resume", Resolution: "continue trade"}); err != nil {
		t.Fatal(err)
	}
	storedDisputeOrder, _ = svc.repo.FindOrder(context.Background(), mustMarketplaceID(t, disputeOrder.ID))
	if storedDisputeOrder.Status != OrderDelivered {
		t.Fatalf("resumed status = %s", storedDisputeOrder.Status)
	}
}

func TestProductionGatewayRejectsBeforeWritingPayment(t *testing.T) {
	svc, db, _ := marketplaceTestService(t, NewGateway("prod"))
	category := createMarketplaceCategory(t, svc, 500)
	item := createPublishedItem(t, svc, category.ID, 1000)
	order, _ := svc.CreateOrder(context.Background(), 101, 100, CreateOrderReq{ItemID: item.ID})
	if _, err := svc.StartPayment(context.Background(), 100, order.ID, PaymentReq{RequestID: "blocked"}); !errors.Is(err, ErrPaymentDisabled) {
		t.Fatalf("payment error = %v", err)
	}
	var count int64
	if err := db.Model(&Payment{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("payment count = %d, err = %v", count, err)
	}
}

func marketplaceTestService(t *testing.T, gateway Gateway) (*Service, *gorm.DB, time.Time) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_foreign_keys=on"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Category{}, &Item{}, &Order{}, &Payment{}, &Refund{}, &Dispute{}, &Settlement{}); err != nil {
		t.Fatal(err)
	}
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 7, 28, 10, 0, 0, 0, location)
	svc := NewService(db, gateway, nil)
	svc.SetSellerVerifier(sellerVerifierStub{})
	svc.now = func() time.Time { return now }
	return svc, db, now
}

func createMarketplaceCategory(t *testing.T, svc *Service, rate int) *CategoryResponse {
	t.Helper()
	value, err := svc.SaveCategory(context.Background(), "", CategoryReq{Name: "数码", CommissionRateBps: &rate})
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func createPublishedItem(t *testing.T, svc *Service, categoryID string, price int64) *ItemResponse {
	t.Helper()
	value, err := svc.CreateItem(context.Background(), 11, 10, ItemReq{
		CategoryID: categoryID, Title: "键盘", Description: "九成新机械键盘", Condition: "likeNew", PriceCents: price,
		Images: []string{"0123456789abcdef0123456789abcdef"}, DeliveryLocation: "图书馆门口", Status: ItemPublished,
	})
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func payOrder(t *testing.T, svc *Service, order *OrderResponse, buyerRootUserID int64, requestID string) {
	t.Helper()
	payment, err := svc.StartPayment(context.Background(), buyerRootUserID, order.ID, PaymentReq{RequestID: requestID})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.TestPaymentCallback(context.Background(), TestPaymentCallbackReq{RequestID: payment.RequestID, GatewayTransactionID: payment.GatewayTransactionID, AmountCents: payment.AmountCents}); err != nil {
		t.Fatal(err)
	}
}

func mustMarketplaceID(t *testing.T, raw string) int64 {
	t.Helper()
	value, err := parseMarketplaceID(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
