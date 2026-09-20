-- Drop inventory schema objects
DROP TRIGGER IF EXISTS update_quarantine_record_updated_at ON inventory.quarantine_record;
DROP TRIGGER IF EXISTS update_reservation_updated_at ON inventory.reservation;
DROP TRIGGER IF EXISTS update_lot_updated_at ON inventory.lot;
DROP TRIGGER IF EXISTS update_stock_level_updated_at ON inventory.stock_level;
DROP FUNCTION IF EXISTS inventory.update_updated_at_column();

DROP TABLE IF EXISTS inventory.stock_adjustment;
DROP TABLE IF EXISTS inventory.quarantine_record;
DROP TABLE IF EXISTS inventory.reservation;
DROP TABLE IF EXISTS inventory.lot;
DROP TABLE IF EXISTS inventory.stock_level;

DROP TYPE IF EXISTS inventory.quarantine_status;
DROP TYPE IF EXISTS inventory.reservation_status;
DROP TYPE IF EXISTS inventory.lot_status;

DROP SCHEMA IF EXISTS inventory;
