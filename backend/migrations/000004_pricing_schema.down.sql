-- Pricing module schema - Down migration

DROP TRIGGER IF EXISTS update_price_list_updated_at ON pricing.price_list;
DROP TRIGGER IF EXISTS update_price_list_item_updated_at ON pricing.price_list_item;
DROP TRIGGER IF EXISTS update_quantity_tier_updated_at ON pricing.quantity_tier;
DROP TRIGGER IF EXISTS update_price_list_assignment_updated_at ON pricing.price_list_assignment;

DROP FUNCTION IF EXISTS pricing.update_updated_at_column();

DROP TABLE IF EXISTS pricing.price_list_assignment;
DROP TABLE IF EXISTS pricing.quantity_tier;
DROP TABLE IF EXISTS pricing.price_list_item;
DROP TABLE IF EXISTS pricing.price_list;

DROP TYPE IF EXISTS pricing.price_type;
DROP TYPE IF EXISTS pricing.price_list_status;

DROP SCHEMA IF EXISTS pricing;