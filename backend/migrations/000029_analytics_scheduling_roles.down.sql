-- Rollback M5 Analytics: pg_cron scheduling and table grants for analytics_ro

-- Unschedule pg_cron jobs
SELECT cron.unschedule('daily-order-rollup');
SELECT cron.unschedule('daily-payment-rollup');
SELECT cron.unschedule('daily-inventory-rollup');
SELECT cron.unschedule('daily-promotion-rollup');
SELECT cron.unschedule('daily-financial-rollup');
SELECT cron.unschedule('daily-erp-sync-rollup');

-- Revoke grants
REVOKE USAGE ON SCHEMA analytics FROM analytics_ro;
REVOKE SELECT ON ALL TABLES IN SCHEMA analytics FROM analytics_ro;
ALTER DEFAULT PRIVILEGES IN SCHEMA analytics REVOKE SELECT ON TABLES FROM analytics_ro;

-- Drop pg_cron extension (CASCADE will drop dependent objects, but we've unscheduled jobs explicitly)
DROP EXTENSION IF EXISTS pg_cron;
