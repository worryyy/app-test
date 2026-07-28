package marketplace

type CategoryReq struct {
	Name              string `json:"name" binding:"required,max=64"`
	CommissionRateBps *int   `json:"commissionRateBps" binding:"omitempty,min=0,max=10000"`
}

type ItemReq struct {
	CategoryID       string   `json:"categoryId" binding:"required"`
	Title            string   `json:"title" binding:"required,max=128"`
	Description      string   `json:"description" binding:"required,max=5000"`
	Condition        string   `json:"condition" binding:"required,max=32"`
	PriceCents       int64    `json:"priceCents" binding:"required,min=1"`
	Images           []string `json:"images" binding:"required,min=1,max=9,dive,required"`
	DeliveryLocation string   `json:"deliveryLocation" binding:"required,max=255"`
	Status           string   `json:"status" binding:"omitempty,oneof=draft published"`
}

type CreateOrderReq struct {
	ItemID string `json:"itemId" binding:"required"`
}

type PaymentReq struct {
	RequestID string `json:"requestId" binding:"required,max=64"`
}

type TestPaymentCallbackReq struct {
	RequestID            string `json:"requestId" binding:"required,max=64"`
	GatewayTransactionID string `json:"gatewayTransactionId" binding:"required,max=128"`
	AmountCents          int64  `json:"amountCents" binding:"required,min=1"`
}

type RefundReq struct {
	RequestID string `json:"requestId" binding:"required,max=64"`
	Reason    string `json:"reason" binding:"required,max=255"`
}

type DisputeReq struct {
	Reason string `json:"reason" binding:"required,max=255"`
}

type RefundDecisionReq struct {
	Approved bool `json:"approved"`
}

type DisputeDecisionReq struct {
	Action     string `json:"action" binding:"required,oneof=resume refund"`
	Resolution string `json:"resolution" binding:"required,max=255"`
}
