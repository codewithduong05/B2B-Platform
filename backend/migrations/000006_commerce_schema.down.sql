-- Drop commerce schema objects
DROP TRIGGER IF EXISTS update_cart_line_updated_at ON commerce.cart_line;
DROP TRIGGER IF EXISTS update_cart_updated_at ON commerce.cart;
DROP FUNCTION IF EXISTS commerce.update_updated_at_column();

DROP TABLE IF EXISTS commerce.cart_line;
DROP TABLE IF EXISTS commerce.cart;

DROP SCHEMA IF EXISTS commerce;
