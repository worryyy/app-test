package marketplace

import "time"

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"

	ItemDraft     = "draft"
	ItemPublished = "published"
	ItemReserved  = "reserved"
	ItemSold      = "sold"
	ItemWithdrawn = "withdrawn"
	ItemHidden    = "hidden"

	OrderPendingPayment = "pendingPayment"
	OrderPaid           = "paid"
	OrderDelivered      = "delivered"
	OrderCompleted      = "completed"
	OrderCancelled      = "cancelled"
	OrderRefunded       = "refunded"
	OrderDisputed       = "disputed"

	RecordPending   = "pending"
	RecordSucceeded = "succeeded"
	RecordRejected  = "rejected"
	RecordFailed    = "failed"

	DisputePending  = "pending"
	DisputeResolved = "resolved"
	DisputeRejected = "rejected"

	DefaultCommissionRateBps = 500
)

type Category struct {
	ID                int64     `gorm:"column:id;primaryKey"`
	Name              string    `gorm:"column:name;uniqueIndex"`
	CommissionRateBps int       `gorm:"column:commission_rate_bps"`
	Status            string    `gorm:"column:status"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

func (Category) TableName() string { return "marketplace_categories" }

type Item struct {
	ID               int64     `gorm:"column:id;primaryKey"`
	SellerRootUserID int64     `gorm:"column:seller_root_user_id"`
	SellerUserID     int64     `gorm:"column:seller_user_id"`
	CategoryID       int64     `gorm:"column:category_id"`
	Title            string    `gorm:"column:title"`
	Description      string    `gorm:"column:description"`
	Condition        string    `gorm:"column:item_condition"`
	PriceCents       int64     `gorm:"column:price_cents"`
	Images           []string  `gorm:"column:images;serializer:json"`
	DeliveryLocation string    `gorm:"column:delivery_location"`
	Status           string    `gorm:"column:status"`
	ReservedOrderID  *int64    `gorm:"column:reserved_order_id"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
	Category         Category  `gorm:"foreignKey:CategoryID"`
}

func (Item) TableName() string { return "marketplace_items" }

type Order struct {
	ID                int64      `gorm:"column:id;primaryKey"`
	ItemID            int64      `gorm:"column:item_id"`
	BuyerRootUserID   int64      `gorm:"column:buyer_root_user_id"`
	BuyerUserID       int64      `gorm:"column:buyer_user_id"`
	SellerRootUserID  int64      `gorm:"column:seller_root_user_id"`
	Status            string     `gorm:"column:status"`
	PriceCents        int64      `gorm:"column:price_cents"`
	CommissionRateBps int        `gorm:"column:commission_rate_bps"`
	PlatformFeeCents  int64      `gorm:"column:platform_fee_cents"`
	SellerNetCents    int64      `gorm:"column:seller_net_cents"`
	ExpiresAt         time.Time  `gorm:"column:expires_at"`
	PaidAt            *time.Time `gorm:"column:paid_at"`
	DeliveredAt       *time.Time `gorm:"column:delivered_at"`
	CompletedAt       *time.Time `gorm:"column:completed_at"`
	CancelledAt       *time.Time `gorm:"column:cancelled_at"`
	RefundedAt        *time.Time `gorm:"column:refunded_at"`
	DisputedAt        *time.Time `gorm:"column:disputed_at"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
	Item              Item       `gorm:"foreignKey:ItemID"`
}

func (Order) TableName() string { return "marketplace_orders" }

type Payment struct {
	ID                   int64     `gorm:"column:id;primaryKey"`
	OrderID              int64     `gorm:"column:order_id"`
	Gateway              string    `gorm:"column:gateway"`
	RequestID            string    `gorm:"column:request_id;uniqueIndex"`
	GatewayTransactionID *string   `gorm:"column:gateway_transaction_id;uniqueIndex"`
	AmountCents          int64     `gorm:"column:amount_cents"`
	Status               string    `gorm:"column:status"`
	CreatedAt            time.Time `gorm:"column:created_at"`
	UpdatedAt            time.Time `gorm:"column:updated_at"`
}

func (Payment) TableName() string { return "marketplace_payments" }

type Refund struct {
	ID                    int64      `gorm:"column:id;primaryKey"`
	OrderID               int64      `gorm:"column:order_id;uniqueIndex"`
	RequestedByRootUserID int64      `gorm:"column:requested_by_root_user_id"`
	RequestID             string     `gorm:"column:request_id;uniqueIndex"`
	GatewayRefundID       *string    `gorm:"column:gateway_refund_id;uniqueIndex"`
	AmountCents           int64      `gorm:"column:amount_cents"`
	Reason                string     `gorm:"column:reason"`
	Status                string     `gorm:"column:status"`
	ReviewedBy            *int64     `gorm:"column:reviewed_by"`
	ReviewedAt            *time.Time `gorm:"column:reviewed_at"`
	CreatedAt             time.Time  `gorm:"column:created_at"`
	UpdatedAt             time.Time  `gorm:"column:updated_at"`
}

func (Refund) TableName() string { return "marketplace_refunds" }

type Dispute struct {
	ID                  int64      `gorm:"column:id;primaryKey"`
	OrderID             int64      `gorm:"column:order_id;uniqueIndex"`
	RaisedByRootUserID  int64      `gorm:"column:raised_by_root_user_id"`
	PreviousOrderStatus string     `gorm:"column:previous_order_status"`
	Reason              string     `gorm:"column:reason"`
	Status              string     `gorm:"column:status"`
	Resolution          string     `gorm:"column:resolution"`
	ResolvedBy          *int64     `gorm:"column:resolved_by"`
	ResolvedAt          *time.Time `gorm:"column:resolved_at"`
	CreatedAt           time.Time  `gorm:"column:created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at"`
}

func (Dispute) TableName() string { return "marketplace_disputes" }

type Settlement struct {
	ID                  int64     `gorm:"column:id;primaryKey"`
	OrderID             int64     `gorm:"column:order_id;uniqueIndex"`
	RequestID           string    `gorm:"column:request_id;uniqueIndex"`
	GatewaySettlementID *string   `gorm:"column:gateway_settlement_id;uniqueIndex"`
	AmountCents         int64     `gorm:"column:amount_cents"`
	Status              string    `gorm:"column:status"`
	CreatedAt           time.Time `gorm:"column:created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at"`
}

func (Settlement) TableName() string { return "marketplace_settlements" }
