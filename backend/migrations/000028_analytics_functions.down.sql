-- Rollback M5 Analytics: PL/pgSQL rollup functions

REVOKE EXECUTE ON FUNCTION analytics.date_window(DATE) FROM analytics_ro;
REVOKE EXECUTE ON FUNCTION analytics.backfill_rollups(DATE, DATE) FROM analytics_ro;
REVOKE EXECUTE ON FUNCTION analytics.run_daily_erp_sync_rollup(DATE) FROM analytics_ro;
REVOKE EXECUTE ON FUNCTION analytics.run_daily_financial_rollup(DATE) FROM analytics_ro;
REVOKE EXECUTE ON FUNCTION analytics.run_daily_promotion_rollup(DATE) FROM analytics_ro;
REVOKE EXECUTE ON FUNCTION analytics.run_daily_inventory_rollup(DATE) FROM analytics_ro;
REVOKE EXECUTE ON FUNCTION analytics.run_daily_payment_rollup(DATE) FROM analytics_ro;
REVOKE EXECUTE ON FUNCTION analytics.run_daily_order_rollup(DATE) FROM analytics_ro;

DROP FUNCTION IF EXISTS analytics.backfill_rollups(DATE, DATE);
DROP FUNCTION IF EXISTS analytics.run_daily_erp_sync_rollup(DATE);
DROP FUNCTION IF EXISTS analytics.run_daily_financial_rollup(DATE);
DROP FUNCTION IF EXISTS analytics.run_daily_promotion_rollup(DATE);
DROP FUNCTION IF EXISTS analytics.run_daily_inventory_rollup(DATE);
DROP FUNCTION IF EXISTS analytics.run_daily_payment_rollup(DATE);
DROP FUNCTION IF EXISTS analytics.run_daily_order_rollup(DATE);
DROP FUNCTION IF EXISTS analytics.date_window(DATE);