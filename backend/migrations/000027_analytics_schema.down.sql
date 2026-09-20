-- Rollback M5 Analytics: Analytics schema and rollup tables

DROP TABLE IF EXISTS analytics.daily_erp_sync_rollup;
DROP TABLE IF EXISTS analytics.daily_financial_rollup;
DROP TABLE IF EXISTS analytics.daily_promotion_rollup;
DROP TABLE IF EXISTS analytics.daily_inventory_rollup;
DROP TABLE IF EXISTS analytics.daily_payment_rollup;
DROP TABLE IF EXISTS analytics.daily_order_rollup;
DROP SCHEMA IF EXISTS analytics CASCADE;