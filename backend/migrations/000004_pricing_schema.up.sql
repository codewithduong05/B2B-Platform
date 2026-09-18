-- Pricing module schema
-- Creates tables for price lists, price items, quantity tiers, and buyer assignments

-- Ensure pricing schema exists
CREATE SCHEMA IF NOT EXISTS pricing;

-- Enum types
CREATE TYPE pricing.price_list_status AS ENUM ('draft', 'active', 'inactive', 'archived');
CREATE TYPE pricing.price_type AS ENUM ('standard', 'contract', 'promotional');

-- Price Lists
CREATE TABLE pricing.price_list (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,  -- opaque public identifier
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status pricing.price_list_status NOT NULL DEFAULT 'draft',
    price_type pricing.price_type NOT NULL DEFAULT 'standard',
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_to TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_pricing_price_list_code ON pricing.price_list (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_pricing_price_list_status ON pricing.price_list (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_pricing_price_list_active ON pricing.price_list (is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_pricing_price_list_effective ON pricing.price_list (effective_from, effective_to) WHERE deleted_at IS NULL;

-- Price List Items (prices per product/unit)
CREATE TABLE pricing.price_list_item (
    id BIGSERIAL PRIMARY KEY,
    price_list_id BIGINT NOT NULL REFERENCES pricing.price_list (id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL,  -- references catalog.product(id)
    unit_id BIGINT NOT NULL,     -- references catalog.unit(id)
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    price_minor BIGINT NOT NULL, -- price in minor units (cents)
    min_quantity INT NOT NULL DEFAULT 1,
    max_quantity INT,            -- NULL means no upper limit
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_to TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (price_list_id, product_id, unit_id, min_quantity, effective_from)
);

CREATE INDEX idx_pricing_price_list_item_list ON pricing.price_list_item (price_list_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_pricing_price_list_item_product ON pricing.price_list_item (product_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_pricing_price_list_item_unit ON pricing.price_list_item (unit_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_pricing_price_list_item_effective ON pricing.price_list_item (effective_from, effective_to) WHERE deleted_at IS NULL;
CREATE INDEX idx_pricing_price_list_item_quantity ON pricing.price_list_item (min_quantity, max_quantity) WHERE deleted_at IS NULL;

-- Quantity Tiers (volume pricing within a price list item)
CREATE TABLE pricing.quantity_tier (
    id BIGSERIAL PRIMARY KEY,
    price_list_item_id BIGINT NOT NULL REFERENCES pricing.price_list_item (id) ON DELETE CASCADE,
    min_quantity INT NOT NULL,
    max_quantity INT,            -- NULL means no upper limit
    price_minor BIGINT NOT NULL, -- price in minor units for this tier
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (price_list_item_id, min_quantity)
);

CREATE INDEX idx_pricing_quantity_tier_item ON pricing.quantity_tier (price_list_item_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_pricing_quantity_tier_quantity ON pricing.quantity_tier (min_quantity, max_quantity) WHERE deleted_at IS NULL;

-- Price List Assignments (buyer-specific price lists)
CREATE TABLE pricing.price_list_assignment (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    price_list_id BIGINT NOT NULL REFERENCES pricing.price_list (id) ON DELETE CASCADE,
    buyer_profile_id BIGINT NOT NULL,  -- references identity.buyer_profile(id)
    assigned_by BIGINT,                -- references identity.user(id)
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_to TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (price_list_id, buyer_profile_id, effective_from)
);

CREATE INDEX idx_pricing_price_list_assignment_list ON pricing.price_list_assignment (price_list_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_pricing_price_list_assignment_buyer ON pricing.price_list_assignment (buyer_profile_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_pricing_price_list_assignment_effective ON pricing.price_list_assignment (effective_from, effective_to) WHERE deleted_at IS NULL;

-- Trigger to update updated_at timestamps
CREATE OR REPLACE FUNCTION pricing.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_price_list_updated_at BEFORE UPDATE ON pricing.price_list FOR EACH ROW EXECUTE FUNCTION pricing.update_updated_at_column();
CREATE TRIGGER update_price_list_item_updated_at BEFORE UPDATE ON pricing.price_list_item FOR EACH ROW EXECUTE FUNCTION pricing.update_updated_at_column();
CREATE TRIGGER update_quantity_tier_updated_at BEFORE UPDATE ON pricing.quantity_tier FOR EACH ROW EXECUTE FUNCTION pricing.update_updated_at_column();
CREATE TRIGGER update_price_list_assignment_updated_at BEFORE UPDATE ON pricing.price_list_assignment FOR EACH ROW EXECUTE FUNCTION pricing.update_updated_at_column();