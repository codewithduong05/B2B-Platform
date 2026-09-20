-- M5 Analytics: Analytics schema, rollup tables, and analytics_ro role

CREATE SCHEMA IF NOT EXISTS analytics;

-- Create analytics_ro role for read-only analytics tool access
-- IMPORTANT: Replace 'YOUR_ANALYTICS_RO_PASSWORD' with a strong, secret password in production.
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'analytics_ro') THEN
        CREATE ROLE analytics_ro NOINHERIT LOGIN PASSWORD 'YOUR_ANALYTICS_RO_PASSWORD';
    END IF;
END
$$;
ALTER ROLE analytics_ro SET statement_timeout = '300s';
ALTER ROLE analytics_ro SET idle_in_transaction_session_timeout = '60s';

-- 1. Daily Order Rollup
CREATE TABLE analytics.daily_order_rollup (
    rollup_date        DATE NOT NULL,
    buyer_code         VARCHAR(32) NOT NULL,
    supplier_code      VARCHAR(32) NOT NULL,
    order_count        BIGINT NOT NULL DEFAULT 0,
    line_count         BIGINT NOT NULL DEFAULT 0,
    total_quantity     NUMERIC(18,4) NOT NULL DEFAULT 0,
    revenue_minor      BIGINT NOT NULL DEFAULT 0,
    currency           CHAR(3) NOT NULL,
    status             VARCHAR(32) NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rollup_date, buyer_code, supplier_code, currency, status)
);

CREATE INDEX idx_daily_order_rollup_date ON analytics.daily_order_rollup (rollup_date DESC);
CREATE INDEX idx_daily_order_rollup_buyer ON analytics.daily_order_rollup (buyer_code, rollup_date DESC);
CREATE INDEX idx_daily_order_rollup_supplier ON analytics.daily_order_rollup (supplier_code, rollup_date DESC);

-- 2. Daily Payment Rollup
CREATE TABLE analytics.daily_payment_rollup (
    rollup_date        DATE NOT NULL,
    buyer_code         VARCHAR(32) NOT NULL,
    payment_method     VARCHAR(64) NOT NULL,
    payment_status     VARCHAR(32) NOT NULL,
    transaction_count  BIGINT NOT NULL DEFAULT 0,
    amount_minor       BIGINT NOT NULL DEFAULT 0,
    currency           CHAR(3) NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rollup_date, buyer_code, payment_method, payment_status, currency)
);

CREATE INDEX idx_daily_payment_rollup_date ON analytics.daily_payment_rollup (rollup_date DESC);
CREATE INDEX idx_daily_payment_rollup_buyer ON analytics.daily_payment_rollup (buyer_code, rollup_date DESC);

-- 3. Daily Inventory Rollup
CREATE TABLE analytics.daily_inventory_rollup (
    rollup_date            DATE NOT NULL,
    product_code           VARCHAR(64) NOT NULL,
    supplier_code          VARCHAR(32) NOT NULL,
    location_code          VARCHAR(32),
    stock_on_hand          NUMERIC(18,4) NOT NULL DEFAULT 0,
    stock_reserved         NUMERIC(18,4) NOT NULL DEFAULT 0,
    stock_available        NUMERIC(18,4) NOT NULL DEFAULT 0,
    lots_count             INT NOT NULL DEFAULT 0,
    expiring_soon_count    INT NOT NULL DEFAULT 0,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rollup_date, product_code, supplier_code, location_code)
);

CREATE INDEX idx_daily_inventory_rollup_date ON analytics.daily_inventory_rollup (rollup_date DESC);
CREATE INDEX idx_daily_inventory_rollup_product ON analytics.daily_inventory_rollup (product_code, rollup_date DESC);
CREATE INDEX idx_daily_inventory_rollup_supplier ON analytics.daily_inventory_rollup (supplier_code, rollup_date DESC);

-- 4. Daily Promotion Rollup
CREATE TABLE analytics.daily_promotion_rollup (
    rollup_date        DATE NOT NULL,
    promotion_code     VARCHAR(64) NOT NULL,
    voucher_code       VARCHAR(64),
    redemptions        BIGINT NOT NULL DEFAULT 0,
    discount_minor     BIGINT NOT NULL DEFAULT 0,
    currency           CHAR(3) NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rollup_date, promotion_code, voucher_code, currency)
);

CREATE INDEX idx_daily_promotion_rollup_date ON analytics.daily_promotion_rollup (rollup_date DESC);
CREATE INDEX idx_daily_promotion_rollup_promo ON analytics.daily_promotion_rollup (promotion_code, rollup_date DESC);

-- 5. Daily Financial Rollup
CREATE TABLE analytics.daily_financial_rollup (
    rollup_date           DATE NOT NULL,
    buyer_code            VARCHAR(32) NOT NULL,
    currency              CHAR(3) NOT NULL,
    invoiced_minor        BIGINT NOT NULL DEFAULT 0,
    collected_minor       BIGINT NOT NULL DEFAULT 0,
    credited_minor        BIGINT NOT NULL DEFAULT 0,
    refunded_minor        BIGINT NOT NULL DEFAULT 0,
    outstanding_minor     BIGINT NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rollup_date, buyer_code, currency)
);

CREATE INDEX idx_daily_financial_rollup_date ON analytics.daily_financial_rollup (rollup_date DESC);
CREATE INDEX idx_daily_financial_rollup_buyer ON analytics.daily_financial_rollup (buyer_code, rollup_date DESC);

-- 6. Daily ERP Sync Rollup
CREATE TABLE analytics.daily_erp_sync_rollup (
    rollup_date        DATE NOT NULL,
    sync_type          VARCHAR(32) NOT NULL,
    job_status         VARCHAR(32) NOT NULL,
    total_records      BIGINT NOT NULL DEFAULT 0,
    matched_records    BIGINT NOT NULL DEFAULT 0,
    created_records    BIGINT NOT NULL DEFAULT 0,
    updated_records    BIGINT NOT NULL DEFAULT 0,
    drift_count        BIGINT NOT NULL DEFAULT 0,
    duration_ms        BIGINT NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (rollup_date, sync_type, job_status)
);

CREATE INDEX idx_daily_erp_sync_rollup_date ON analytics.daily_erp_sync_rollup (rollup_date DESC);

