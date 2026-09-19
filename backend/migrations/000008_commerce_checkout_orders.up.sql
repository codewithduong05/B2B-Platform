-- Commerce checkout + order core (TASK-005).
--
-- "order" is a reserved word and is always double-quoted.
-- Money is integer minor units (BIGINT), matching catalog.base_price_minor.
-- Cross-schema FKs follow the existing codebase convention (000005/000006):
-- integrity at the database level, reads via owning-module surfaces.

CREATE TYPE commerce.order_status AS ENUM ('placed');

-- Orders: exactly one per supplier per checkout (application-enforced split).
CREATE TABLE commerce."order" (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    buyer_id BIGINT NOT NULL REFERENCES identity.buyer_profile (id) ON DELETE RESTRICT,
    supplier_id BIGINT NOT NULL REFERENCES catalog.supplier (id) ON DELETE RESTRICT,
    cart_id BIGINT REFERENCES commerce.cart (id) ON DELETE SET NULL,
    cart_code VARCHAR(26),
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    subtotal_minor BIGINT NOT NULL DEFAULT 0 CHECK (subtotal_minor >= 0),
    discounts_minor BIGINT NOT NULL DEFAULT 0 CHECK (discounts_minor >= 0),
    total_minor BIGINT NOT NULL DEFAULT 0 CHECK (total_minor >= 0),
    status commerce.order_status NOT NULL DEFAULT 'placed',
    placed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_commerce_order_buyer ON commerce."order" (buyer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_commerce_order_supplier ON commerce."order" (supplier_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_commerce_order_code ON commerce."order" (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_commerce_order_placed ON commerce."order" (placed_at DESC) WHERE deleted_at IS NULL;

-- Order lines carry an authoritative price snapshot so history never
-- depends on the current mutable product price.
CREATE TABLE commerce.order_line (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    order_id BIGINT NOT NULL REFERENCES commerce."order" (id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES catalog.product (id) ON DELETE RESTRICT,
    supplier_id BIGINT NOT NULL REFERENCES catalog.supplier (id) ON DELETE RESTRICT,
    unit_id BIGINT NOT NULL REFERENCES catalog.unit (id) ON DELETE RESTRICT,
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_price_minor BIGINT NOT NULL CHECK (unit_price_minor >= 0),
    total_price_minor BIGINT NOT NULL CHECK (total_price_minor >= 0),
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    product_code VARCHAR(64) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    unit_code VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (order_id, product_id)
);

CREATE INDEX idx_commerce_order_line_order ON commerce.order_line (order_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_commerce_order_line_product ON commerce.order_line (product_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_commerce_order_line_code ON commerce.order_line (code) WHERE deleted_at IS NULL;

-- Checkout idempotency claims: one row per (buyer, key). Only successful
-- checkouts are recorded as completed; failed attempts delete the pending
-- row so the same key may be retried.
CREATE TABLE commerce.checkout_idempotency (
    id BIGSERIAL PRIMARY KEY,
    buyer_id BIGINT NOT NULL REFERENCES identity.buyer_profile (id) ON DELETE CASCADE,
    idem_key VARCHAR(100) NOT NULL,
    payload_hash CHAR(64) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed')),
    result JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (buyer_id, idem_key)
);

CREATE INDEX idx_commerce_idem_buyer_key ON commerce.checkout_idempotency (buyer_id, idem_key);

-- Order lifecycle audit (R8: every order state change is auditable).
CREATE TABLE commerce.order_history (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES commerce."order" (id) ON DELETE CASCADE,
    from_status commerce.order_status,
    to_status commerce.order_status NOT NULL,
    actor BIGINT,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_commerce_order_history_order ON commerce.order_history (order_id);

CREATE TRIGGER update_order_updated_at BEFORE UPDATE ON commerce."order" FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();
CREATE TRIGGER update_order_line_updated_at BEFORE UPDATE ON commerce.order_line FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();
CREATE TRIGGER update_checkout_idempotency_updated_at BEFORE UPDATE ON commerce.checkout_idempotency FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();
