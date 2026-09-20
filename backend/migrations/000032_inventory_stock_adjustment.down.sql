-- TASK-034: Drop inventory.stock_adjustment table

DROP INDEX IF EXISTS inventory.idx_inventory_stock_adjustment_lot;
DROP TABLE IF EXISTS inventory.stock_adjustment;
