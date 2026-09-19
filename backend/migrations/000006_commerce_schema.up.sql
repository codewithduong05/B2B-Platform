-- Commerce module schema
-- Creates tables for carts and cart lines

CREATE SCHEMA IF NOT EXISTS commerce;

-- Update timestamp function for commerce
CREATE OR REPLACE FUNCTION commerce.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Carts (one per buyer)
CREATE TABLE commerce.cart (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    buyer_id BIGINT NOT NULL REFERENCES identity.buyer_profile (id) ON DELETE CASCADE,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (buyer_id)
);

CREATE INDEX idx_commerce_cart_buyer ON commerce.cart (buyer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_commerce_cart_code ON commerce.cart (code) WHERE deleted_at IS NULL;

-- Cart Lines
CREATE TABLE commerce.cart_line (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    cart_id BIGINT NOT NULL REFERENCES commerce.cart (id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES catalog.product (id) ON DELETE RESTRICT,
    supplier_id BIGINT NOT NULL REFERENCES catalog.supplier (id) ON DELETE RESTRICT,
    unit_id BIGINT NOT NULL REFERENCES catalog.unit (id) ON DELETE RESTRICT,
    quantity INT NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (cart_id, product_id)
);

CREATE INDEX idx_commerce_cart_line_cart ON commerce.cart_line (cart_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_commerce_cart_line_product ON commerce.cart_line (product_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_commerce_cart_line_code ON commerce.cart_line (code) WHERE deleted_at IS NULL;

CREATE TRIGGER update_cart_updated_at BEFORE UPDATE ON commerce.cart FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();
CREATE TRIGGER update_cart_line_updated_at BEFORE UPDATE ON commerce.cart_line FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();
