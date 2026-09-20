-- TASK-034: Add inventory.stock_adjustment table

CREATE TABLE IF NOT EXISTS inventory.stock_adjustment (
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

CREATE INDEX IF NOT EXISTS idx_inventory_stock_adjustment_lot ON inventory.stock_adjustment (lot_id);
