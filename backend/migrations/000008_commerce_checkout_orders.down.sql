DROP TRIGGER IF EXISTS update_checkout_idempotency_updated_at ON commerce.checkout_idempotency;
DROP TRIGGER IF EXISTS update_order_line_updated_at ON commerce.order_line;
DROP TRIGGER IF EXISTS update_order_updated_at ON commerce."order";

DROP TABLE IF EXISTS commerce.order_history;
DROP TABLE IF EXISTS commerce.checkout_idempotency;
DROP TABLE IF EXISTS commerce.order_line;
DROP TABLE IF EXISTS commerce."order";

DROP TYPE IF EXISTS commerce.order_status;
