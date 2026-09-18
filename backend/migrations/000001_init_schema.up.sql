-- Initial schema setup for Atlas platform
-- Creates all module schemas and the base tables

-- Create schemas for each module
CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS catalog;
CREATE SCHEMA IF NOT EXISTS pricing;
CREATE SCHEMA IF NOT EXISTS inventory;
CREATE SCHEMA IF NOT EXISTS commerce;
CREATE SCHEMA IF NOT EXISTS payments;
CREATE SCHEMA IF NOT EXISTS promotions;
CREATE SCHEMA IF NOT EXISTS crm;
CREATE SCHEMA IF NOT EXISTS cms;
CREATE SCHEMA IF NOT EXISTS suppliers;
CREATE SCHEMA IF NOT EXISTS platform;
CREATE SCHEMA IF NOT EXISTS erp;
CREATE SCHEMA IF NOT EXISTS ai;

-- Create a simple health check table in platform schema
CREATE TABLE IF NOT EXISTS platform.health_check (
    id BIGSERIAL PRIMARY KEY,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'ok'
);

-- Create migration tracking table (golang-migrate creates its own, but this is for reference)
COMMENT ON SCHEMA identity IS 'Identity module: users, credentials, sessions, verification';
COMMENT ON SCHEMA catalog IS 'Catalog module: products, categories, brands, units, attributes';
COMMENT ON SCHEMA pricing IS 'Pricing module: price lists, entries, tiers, assignments';
COMMENT ON SCHEMA inventory IS 'Inventory module: stock levels, lots, reservations, quarantine';
COMMENT ON SCHEMA commerce IS 'Commerce module: carts, orders, shipments, invoices';
COMMENT ON SCHEMA payments IS 'Payments module: methods, intents, transactions, refunds';
COMMENT ON SCHEMA promotions IS 'Promotions module: promotions, vouchers, eligibility';
COMMENT ON SCHEMA crm IS 'CRM module: leads, referrals, partners';
COMMENT ON SCHEMA cms IS 'CMS module: articles, pages, menus, banners, settings';
COMMENT ON SCHEMA suppliers IS 'Suppliers module: supplier profiles, contracts, onboarding';
COMMENT ON SCHEMA platform IS 'Platform module: notifications, media, feature flags, audit log';
COMMENT ON SCHEMA erp IS 'ERP module: sync jobs, idempotency records, event log';
COMMENT ON SCHEMA ai IS 'AI module: prompts, proposals, review queue, usage, evaluations';