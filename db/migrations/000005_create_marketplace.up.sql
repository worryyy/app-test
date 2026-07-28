CREATE TABLE IF NOT EXISTS marketplace_categories (
    id BIGINT NOT NULL,
    name VARCHAR(64) NOT NULL,
    commission_rate_bps INT NOT NULL DEFAULT 500,
    status VARCHAR(24) NOT NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_marketplace_category_name (name),
    KEY idx_marketplace_categories_status (status, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS marketplace_items (
    id BIGINT NOT NULL,
    seller_root_user_id BIGINT NOT NULL,
    seller_user_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,
    title VARCHAR(128) NOT NULL,
    description TEXT NOT NULL,
    item_condition VARCHAR(32) NOT NULL,
    price_cents BIGINT NOT NULL,
    images JSON NOT NULL,
    delivery_location VARCHAR(255) NOT NULL,
    status VARCHAR(24) NOT NULL,
    reserved_order_id BIGINT NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_marketplace_items_search (status, category_id, created_at DESC),
    KEY idx_marketplace_items_seller (seller_root_user_id, created_at DESC),
    CONSTRAINT fk_marketplace_item_category FOREIGN KEY (category_id) REFERENCES marketplace_categories(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS marketplace_orders (
    id BIGINT NOT NULL,
    item_id BIGINT NOT NULL,
    buyer_root_user_id BIGINT NOT NULL,
    buyer_user_id BIGINT NOT NULL,
    seller_root_user_id BIGINT NOT NULL,
    status VARCHAR(24) NOT NULL,
    price_cents BIGINT NOT NULL,
    commission_rate_bps INT NOT NULL,
    platform_fee_cents BIGINT NOT NULL,
    seller_net_cents BIGINT NOT NULL,
    expires_at DATETIME(3) NOT NULL,
    paid_at DATETIME(3) NULL,
    delivered_at DATETIME(3) NULL,
    completed_at DATETIME(3) NULL,
    cancelled_at DATETIME(3) NULL,
    refunded_at DATETIME(3) NULL,
    disputed_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_marketplace_orders_item_status (item_id, status),
    KEY idx_marketplace_orders_buyer (buyer_root_user_id, created_at DESC),
    KEY idx_marketplace_orders_seller (seller_root_user_id, created_at DESC),
    KEY idx_marketplace_orders_due (status, expires_at, delivered_at),
    CONSTRAINT fk_marketplace_order_item FOREIGN KEY (item_id) REFERENCES marketplace_items(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS marketplace_payments (
    id BIGINT NOT NULL,
    order_id BIGINT NOT NULL,
    gateway VARCHAR(32) NOT NULL,
    request_id VARCHAR(64) NOT NULL,
    gateway_transaction_id VARCHAR(128) NULL,
    amount_cents BIGINT NOT NULL,
    status VARCHAR(24) NOT NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_marketplace_payment_request (request_id),
    UNIQUE KEY uk_marketplace_payment_transaction (gateway_transaction_id),
    KEY idx_marketplace_payments_order (order_id, created_at DESC),
    CONSTRAINT fk_marketplace_payment_order FOREIGN KEY (order_id) REFERENCES marketplace_orders(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS marketplace_refunds (
    id BIGINT NOT NULL,
    order_id BIGINT NOT NULL,
    requested_by_root_user_id BIGINT NOT NULL,
    request_id VARCHAR(64) NOT NULL,
    gateway_refund_id VARCHAR(128) NULL,
    amount_cents BIGINT NOT NULL,
    reason VARCHAR(255) NOT NULL,
    status VARCHAR(24) NOT NULL,
    reviewed_by BIGINT NULL,
    reviewed_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_marketplace_refund_order (order_id),
    UNIQUE KEY uk_marketplace_refund_request (request_id),
    UNIQUE KEY uk_marketplace_refund_gateway (gateway_refund_id),
    KEY idx_marketplace_refunds_review (status, created_at ASC),
    CONSTRAINT fk_marketplace_refund_order FOREIGN KEY (order_id) REFERENCES marketplace_orders(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS marketplace_disputes (
    id BIGINT NOT NULL,
    order_id BIGINT NOT NULL,
    raised_by_root_user_id BIGINT NOT NULL,
    previous_order_status VARCHAR(24) NOT NULL,
    reason VARCHAR(255) NOT NULL,
    status VARCHAR(24) NOT NULL,
    resolution VARCHAR(255) NOT NULL DEFAULT '',
    resolved_by BIGINT NULL,
    resolved_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_marketplace_dispute_order (order_id),
    KEY idx_marketplace_disputes_review (status, created_at ASC),
    CONSTRAINT fk_marketplace_dispute_order FOREIGN KEY (order_id) REFERENCES marketplace_orders(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS marketplace_settlements (
    id BIGINT NOT NULL,
    order_id BIGINT NOT NULL,
    request_id VARCHAR(64) NOT NULL,
    gateway_settlement_id VARCHAR(128) NULL,
    amount_cents BIGINT NOT NULL,
    status VARCHAR(24) NOT NULL,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_marketplace_settlement_order (order_id),
    UNIQUE KEY uk_marketplace_settlement_request (request_id),
    UNIQUE KEY uk_marketplace_settlement_gateway (gateway_settlement_id),
    KEY idx_marketplace_settlements_status (status, created_at ASC),
    CONSTRAINT fk_marketplace_settlement_order FOREIGN KEY (order_id) REFERENCES marketplace_orders(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
