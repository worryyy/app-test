package marketplace

import (
	"strconv"
	"time"
)

type CategoryResponse struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	CommissionRateBps int    `json:"commissionRateBps"`
}

type ItemResponse struct {
	ID               string   `json:"id"`
	SellerRootUserID string   `json:"sellerRootUserId"`
	CategoryID       string   `json:"categoryId"`
	CategoryName     string   `json:"categoryName,omitempty"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Condition        string   `json:"condition"`
	PriceCents       int64    `json:"priceCents"`
	Images           []string `json:"images"`
	DeliveryLocation string   `json:"deliveryLocation"`
	Status           string   `json:"status"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
}

type OrderResponse struct {
	ID                string       `json:"id"`
	ItemID            string       `json:"itemId"`
	BuyerRootUserID   string       `json:"buyerRootUserId"`
	SellerRootUserID  string       `json:"sellerRootUserId"`
	Status            string       `json:"status"`
	PriceCents        int64        `json:"priceCents"`
	CommissionRateBps int          `json:"commissionRateBps"`
	PlatformFeeCents  int64        `json:"platformFeeCents"`
	SellerNetCents    int64        `json:"sellerNetCents"`
	ExpiresAt         string       `json:"expiresAt"`
	PaidAt            string       `json:"paidAt,omitempty"`
	DeliveredAt       string       `json:"deliveredAt,omitempty"`
	CompletedAt       string       `json:"completedAt,omitempty"`
	CreatedAt         string       `json:"createdAt"`
	Item              ItemResponse `json:"item"`
}

type PaymentResponse struct {
	OrderID              string `json:"orderId"`
	RequestID            string `json:"requestId"`
	Gateway              string `json:"gateway"`
	GatewayTransactionID string `json:"gatewayTransactionId"`
	AmountCents          int64  `json:"amountCents"`
	Status               string `json:"status"`
}

type RefundResponse struct {
	ID              string `json:"id"`
	OrderID         string `json:"orderId"`
	RequestID       string `json:"requestId"`
	AmountCents     int64  `json:"amountCents"`
	Reason          string `json:"reason"`
	Status          string `json:"status"`
	GatewayRefundID string `json:"gatewayRefundId,omitempty"`
	CreatedAt       string `json:"createdAt"`
}

type DisputeResponse struct {
	ID         string `json:"id"`
	OrderID    string `json:"orderId"`
	Reason     string `json:"reason"`
	Status     string `json:"status"`
	Resolution string `json:"resolution,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

type SettlementResponse struct {
	ID                  string `json:"id"`
	OrderID             string `json:"orderId"`
	RequestID           string `json:"requestId"`
	AmountCents         int64  `json:"amountCents"`
	Status              string `json:"status"`
	GatewaySettlementID string `json:"gatewaySettlementId,omitempty"`
	CreatedAt           string `json:"createdAt"`
}

func categoryResponse(value Category) CategoryResponse {
	return CategoryResponse{ID: marketplaceID(value.ID), Name: value.Name, CommissionRateBps: value.CommissionRateBps}
}

func itemResponse(value Item) ItemResponse {
	images := value.Images
	if images == nil {
		images = []string{}
	}
	return ItemResponse{
		ID: marketplaceID(value.ID), SellerRootUserID: marketplaceID(value.SellerRootUserID), CategoryID: marketplaceID(value.CategoryID),
		CategoryName: value.Category.Name, Title: value.Title, Description: value.Description, Condition: value.Condition,
		PriceCents: value.PriceCents, Images: images, DeliveryLocation: value.DeliveryLocation, Status: value.Status,
		CreatedAt: marketplaceTime(value.CreatedAt), UpdatedAt: marketplaceTime(value.UpdatedAt),
	}
}

func orderResponse(value Order) OrderResponse {
	return OrderResponse{
		ID: marketplaceID(value.ID), ItemID: marketplaceID(value.ItemID), BuyerRootUserID: marketplaceID(value.BuyerRootUserID),
		SellerRootUserID: marketplaceID(value.SellerRootUserID), Status: value.Status, PriceCents: value.PriceCents,
		CommissionRateBps: value.CommissionRateBps, PlatformFeeCents: value.PlatformFeeCents, SellerNetCents: value.SellerNetCents,
		ExpiresAt: marketplaceTime(value.ExpiresAt), PaidAt: optionalMarketplaceTime(value.PaidAt), DeliveredAt: optionalMarketplaceTime(value.DeliveredAt),
		CompletedAt: optionalMarketplaceTime(value.CompletedAt), CreatedAt: marketplaceTime(value.CreatedAt), Item: itemResponse(value.Item),
	}
}

func refundResponse(value Refund) RefundResponse {
	return RefundResponse{ID: marketplaceID(value.ID), OrderID: marketplaceID(value.OrderID), RequestID: value.RequestID, AmountCents: value.AmountCents, Reason: value.Reason, Status: value.Status, GatewayRefundID: stringValue(value.GatewayRefundID), CreatedAt: marketplaceTime(value.CreatedAt)}
}

func disputeResponse(value Dispute) DisputeResponse {
	return DisputeResponse{ID: marketplaceID(value.ID), OrderID: marketplaceID(value.OrderID), Reason: value.Reason, Status: value.Status, Resolution: value.Resolution, CreatedAt: marketplaceTime(value.CreatedAt)}
}

func settlementResponse(value Settlement) SettlementResponse {
	return SettlementResponse{ID: marketplaceID(value.ID), OrderID: marketplaceID(value.OrderID), RequestID: value.RequestID, AmountCents: value.AmountCents, Status: value.Status, GatewaySettlementID: stringValue(value.GatewaySettlementID), CreatedAt: marketplaceTime(value.CreatedAt)}
}

func marketplaceID(value int64) string { return strconv.FormatInt(value, 10) }
func marketplaceTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return value.In(location).Format(time.RFC3339)
}
func optionalMarketplaceTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return marketplaceTime(*value)
}
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
