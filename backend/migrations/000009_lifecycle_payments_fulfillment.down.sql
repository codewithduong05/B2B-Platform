DROP TRIGGER IF EXISTS update_invoice_updated_at ON commerce.invoice;
DROP TRIGGER IF EXISTS update_shipment_updated_at ON commerce.shipment;

DROP TABLE IF EXISTS payments.payment_attempt;
DROP TABLE IF EXISTS payments.payment_intent;
DROP TABLE IF EXISTS payments.payment_method;
DROP SCHEMA IF EXISTS payments;

DROP TABLE IF EXISTS commerce.invoice;
DROP TABLE IF EXISTS commerce.shipment_line;
DROP TABLE IF EXISTS commerce.shipment;

ALTER TABLE commerce."order"
    DROP COLUMN IF EXISTS idem_key,
    DROP COLUMN IF EXISTS hold_reason,
    DROP COLUMN IF EXISTS on_hold;

-- NOTE: Postgres cannot drop enum values. The values added by the up
-- migration (confirmed, processing, shipped, delivered, cancelled) remain
-- in the type after rollback; only tables/columns are removed.
