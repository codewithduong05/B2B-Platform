-- Rollback M5 ERP Slice 1

DROP TABLE IF EXISTS erp.order_dispatch;
DROP TABLE IF EXISTS erp.sync_drift;
DROP TABLE IF EXISTS erp.sync_job;
DROP TABLE IF EXISTS erp.erp_webhook_event;
DROP SCHEMA IF EXISTS erp CASCADE;
