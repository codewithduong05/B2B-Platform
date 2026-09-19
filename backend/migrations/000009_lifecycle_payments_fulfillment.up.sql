-- TASK-006: order lifecycle, shipments, invoices, payments schema.
--
-- order_status grows by ALTER TYPE ... ADD VALUE (non-destructive, no table
-- rewrite). New values: confirmed, processing, shipped, delivered, cancelled.

ALTER TYPE commerce.order_status ADD VALUE IF NOT EXISTS 'confirmed';
ALTER TYPE commerce.order_status ADD VALUE IF NOT EXISTS 'processing';
ALTER TYPE commerce.order_status ADD VALUE IF NOT EXISTS 'shipped';
ALTER TYPE commerce.order_status ADD VALUE IF NOT EXISTS 'delivered';
ALTER TYPE commerce.order_status ADD VALUE IF NOT EXISTS 'cancelled';

-- Hold is orthogonal to fulfillment status (05: hold/release endpoints).
-- idem_key enables cancel-time reservation release for checkout-created
-- orders (NULL for rows predating this column).
ALTER TABLE commerce."order"
    ADD COLUMN on_hold BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN hold_reason TEXT,
    ADD COLUMN idem_key VARCHAR(100);

-- Shipments: subordinate tracking records. Partial fulfillment is allowed
-- per shipment line; the order-level shipped transition is coverage-gated
-- in the service layer.
CREATE TABLE commerce.shipment (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    order_id BIGINT NOT NULL REFERENCES commerce."order" (id) ON DELETE CASCADE,
    carrier VARCHAR(64),
    tracking_code VARCHAR(128),
    status VARCHAR(16) NOT NULL DEFAULT 'preparing' CHECK (status IN ('preparing', 'shipped', 'delivered', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_commerce_shipment_order ON commerce.shipment (order_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_commerce_shipment_code ON commerce.shipment (code) WHERE deleted_at IS NULL;

CREATE TABLE commerce.shipment_line (
    id BIGSERIAL PRIMARY KEY,
    shipment_id BIGINT NOT NULL REFERENCES commerce.shipment (id) ON DELETE CASCADE,
    order_line_id BIGINT NOT NULL REFERENCES commerce.order_line (id) ON DELETE RESTRICT,
    quantity INT NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (shipment_id, order_line_id)
);

CREATE INDEX idx_commerce_shipment_line_shipment ON commerce.shipment_line (shipment_id);
CREATE INDEX idx_commerce_shipment_line_order_line ON commerce.shipment_line (order_line_id);

-- Invoices: manual B2B issue/void. Rows are never deleted (R10).
CREATE TABLE commerce.invoice (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    order_id BIGINT NOT NULL REFERENCES commerce."order" (id) ON DELETE RESTRICT,
    subtotal_minor BIGINT NOT NULL DEFAULT 0 CHECK (subtotal_minor >= 0),
    total_minor BIGINT NOT NULL DEFAULT 0 CHECK (total_minor >= 0),
    balance_minor BIGINT NOT NULL DEFAULT 0 CHECK (balance_minor >= 0),
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    status VARCHAR(16) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'issued', 'void')),
    issued_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_commerce_invoice_order ON commerce.invoice (order_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_commerce_invoice_code ON commerce.invoice (code) WHERE deleted_at IS NULL;

-- Payments module schema (06: payments owns methods, intents, transactions).
-- B2B methods only; no gateway integration in this task.
CREATE SCHEMA IF NOT EXISTS payments;

CREATE TABLE payments.payment_method (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    buyer_id BIGINT NOT NULL REFERENCES identity.buyer_profile (id) ON DELETE CASCADE,
    method_type VARCHAR(32) NOT NULL CHECK (method_type IN ('credit_terms', 'bank_transfer', 'cod', 'card')),
    display_name VARCHAR(128) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_payments_method_buyer ON payments.payment_method (buyer_id) WHERE deleted_at IS NULL;

CREATE TABLE payments.payment_intent (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    buyer_id BIGINT NOT NULL REFERENCES identity.buyer_profile (id) ON DELETE RESTRICT,
    order_id BIGINT NOT NULL REFERENCES commerce."order" (id) ON DELETE RESTRICT,
    order_code VARCHAR(26) NOT NULL,
    payment_method_id BIGINT REFERENCES payments.payment_method (id) ON DELETE SET NULL,
    amount_minor BIGINT NOT NULL CHECK (amount_minor >= 0),
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    status VARCHAR(32) NOT NULL DEFAULT 'requires_action' CHECK (status IN ('requires_action', 'processing', 'succeeded', 'failed', 'cancelled')),
    idem_key VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (buyer_id, idem_key)
);

CREATE INDEX idx_payments_intent_buyer ON payments.payment_intent (buyer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payments_intent_order ON payments.payment_intent (order_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payments_intent_code ON payments.payment_intent (code) WHERE deleted_at IS NULL;

CREATE TABLE payments.payment_attempt (
    id BIGSERIAL PRIMARY KEY,
    intent_id BIGINT NOT NULL REFERENCES payments.payment_intent (id) ON DELETE CASCADE,
    result VARCHAR(16) NOT NULL CHECK (result IN ('succeeded', 'failed')),
    gateway_ref VARCHAR(128),
    note TEXT,
    recorded_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_attempt_intent ON payments.payment_attempt (intent_id);

CREATE TRIGGER update_shipment_updated_at BEFORE UPDATE ON commerce.shipment FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();
CREATE TRIGGER update_invoice_updated_at BEFORE UPDATE ON commerce.invoice FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();
