-- M5 Analytics: pg_cron scheduling and table grants for analytics_ro
-- Assumes analytics schema, tables, functions, and analytics_ro role from 000027 and 000028 are already applied.

-- Enable pg_cron extension
CREATE EXTENSION IF NOT EXISTS pg_cron;

-- Schedule each rollup at 02:00 UTC for the previous day
-- Note: CURRENT_DATE - 1 is used to process data for the day that just completed.
SELECT cron.schedule('daily-order-rollup', '0 2 * * *', $$
    SELECT analytics.run_daily_order_rollup(CURRENT_DATE - 1);
$$);

SELECT cron.schedule('daily-payment-rollup', '0 2 * * *', $$
    SELECT analytics.run_daily_payment_rollup(CURRENT_DATE - 1);
$$);

SELECT cron.schedule('daily-inventory-rollup', '0 2 * * *', $$
    SELECT analytics.run_daily_inventory_rollup(CURRENT_DATE - 1);
$$);

SELECT cron.schedule('daily-promotion-rollup', '0 2 * * *', $$
    SELECT analytics.run_daily_promotion_rollup(CURRENT_DATE - 1);
$$);

SELECT cron.schedule('daily-financial-rollup', '0 2 * * *', $$
    SELECT analytics.run_daily_financial_rollup(CURRENT_DATE - 1);
$$);

SELECT cron.schedule('daily-erp-sync-rollup', '0 2 * * *', $$
    SELECT analytics.run_daily_erp_sync_rollup(CURRENT_DATE - 1);
$$);

-- Grant usage on schema and select on tables to analytics_ro
GRANT USAGE ON SCHEMA analytics TO analytics_ro;
GRANT SELECT ON ALL TABLES IN SCHEMA analytics TO analytics_ro;
ALTER DEFAULT PRIVILEGES IN SCHEMA analytics GRANT SELECT ON TABLES TO analytics_ro;