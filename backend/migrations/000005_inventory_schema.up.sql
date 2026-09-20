-- Inventory module schema
-- Creates tables for stock levels, lots, reservations, and quarantine records

CREATE SCHEMA IF NOT EXISTS inventory;

-- Enums
CREATE TYPE inventory.lot_status AS ENUM ('active', 'depleted', 'expired', 'quarantined');
CREATE TYPE inventory.reservation_status AS ENUM ('reserved', 'allocated', 'released', 'expired');
CREATE TYPE inventory.quarantine_status AS ENUM ('pending', 'quarantined', 'released', 'disposed');

-- Stock Levels (per product and supplier)
CREATE TABLE inventory.stock_level (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,  -- opaque public identifier
    product_id BIGINT NOT NULL REFERENCES catalog.product (id) ON DELETE RESTRICT,
    supplier_id BIGINT NOT NULL REFERENCES catalog.supplier (id) ON DELETE RESTRICT,
    available_quantity INT NOT NULL DEFAULT 0 CHECK (available_quantity >= 0),
    reserved_quantity INT NOT NULL DEFAULT 0 CHECK (reserved_quantity >= 0),
    total_quantity INT NOT NULL DEFAULT 0 CHECK (total_quantity >= 0),
    safety_stock INT NOT NULL DEFAULT 0 CHECK (safety_stock >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (product_id, supplier_id)
);

CREATE INDEX idx_inventory_stock_level_product ON inventory.stock_level (product_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_inventory_stock_level_supplier ON inventory.stock_level (supplier_id) WHERE deleted_at IS NULL;

-- Lots (composed of stock levels, tracked by expiry and lot number)
CREATE TABLE inventory.lot (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    stock_level_id BIGINT NOT NULL REFERENCES inventory.stock_level (id) ON DELETE CASCADE,
    lot_number VARCHAR(100) NOT NULL,
    initial_quantity INT NOT NULL CHECK (initial_quantity > 0),
    available_quantity INT NOT NULL CHECK (available_quantity >= 0),
    reserved_quantity INT NOT NULL DEFAULT 0 CHECK (reserved_quantity >= 0),
    status inventory.lot_status NOT NULL DEFAULT 'active',
    is_quarantined BOOLEAN NOT NULL DEFAULT FALSE,
    production_date DATE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_inventory_lot_stock_level ON inventory.lot (stock_level_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_inventory_lot_expiry ON inventory.lot (expires_at ASC, id ASC) WHERE deleted_at IS NULL AND is_quarantined = FALSE AND available_quantity > 0 AND status = 'active';
CREATE INDEX idx_inventory_lot_status ON inventory.lot (status, is_quarantined) WHERE deleted_at IS NULL;

-- Reservations (allocated through FEFO, referencing order lines)
CREATE TABLE inventory.reservation (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    lot_id BIGINT NOT NULL REFERENCES inventory.lot (id) ON DELETE CASCADE,
    order_line_id BIGINT,  -- references commerce.order_line (id) when commerce is implemented
    request_id VARCHAR(100) NOT NULL,  -- for idempotency
    quantity INT NOT NULL CHECK (quantity > 0),
    status inventory.reservation_status NOT NULL DEFAULT 'reserved',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_inventory_reservation_lot ON inventory.reservation (lot_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_inventory_reservation_request_lot ON inventory.reservation (request_id, lot_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_inventory_reservation_expiry ON inventory.reservation (expires_at ASC) WHERE status = 'reserved' AND deleted_at IS NULL;

-- Quarantine Records / Adjustments
CREATE TABLE inventory.quarantine_record (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    lot_id BIGINT NOT NULL REFERENCES inventory.lot (id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    status inventory.quarantine_status NOT NULL DEFAULT 'quarantined',
    adjusted_quantity INT NOT NULL DEFAULT 0,
    adjusted_by BIGINT REFERENCES identity.user (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_inventory_quarantine_lot ON inventory.quarantine_record (lot_id) WHERE deleted_at IS NULL;

-- Stock Adjustments
CREATE TABLE inventory.stock_adjustment (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    lot_id BIGINT NOT NULL REFERENCES inventory.lot (id) ON DELETE CASCADE,
    quantity_delta INT NOT NULL,
    previous_quantity INT NOT NULL,
    new_quantity INT NOT NULL,
    reason_code VARCHAR(100) NOT NULL,
    reason TEXT NOT NULL,
    adjusted_by BIGINT REFERENCES identity.user (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inventory_stock_adjustment_lot ON inventory.stock_adjustment (lot_id);

-- Trigger to update updated_at timestamps
CREATE OR REPLACE FUNCTION inventory.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_stock_level_updated_at BEFORE UPDATE ON inventory.stock_level FOR EACH ROW EXECUTE FUNCTION inventory.update_updated_at_column();
CREATE TRIGGER update_lot_updated_at BEFORE UPDATE ON inventory.lot FOR EACH ROW EXECUTE FUNCTION inventory.update_updated_at_column();
CREATE TRIGGER update_reservation_updated_at BEFORE UPDATE ON inventory.reservation FOR EACH ROW EXECUTE FUNCTION inventory.update_updated_at_column();
CREATE TRIGGER update_quarantine_record_updated_at BEFORE UPDATE ON inventory.quarantine_record FOR EACH ROW EXECUTE FUNCTION inventory.update_updated_at_column();
