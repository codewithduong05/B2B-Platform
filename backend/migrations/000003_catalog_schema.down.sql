-- Rollback catalog module schema

-- Drop triggers
DROP TRIGGER IF EXISTS update_product_attribute_updated_at ON catalog.product_attribute;
DROP TRIGGER IF EXISTS update_attribute_value_updated_at ON catalog.attribute_value;
DROP TRIGGER IF EXISTS update_attribute_updated_at ON catalog.attribute;
DROP TRIGGER IF EXISTS update_product_media_updated_at ON catalog.product_media;
DROP TRIGGER IF EXISTS update_product_unit_updated_at ON catalog.product_unit;
DROP TRIGGER IF EXISTS update_product_updated_at ON catalog.product;
CREATE TRIGGER update_supplier_updated_at BEFORE UPDATE ON catalog.supplier FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();
CREATE TRIGGER update_handling_class_updated_at BEFORE UPDATE ON catalog.handling_class FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();
CREATE TRIGGER update_unit_updated_at BEFORE UPDATE ON catalog.unit FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();
CREATE TRIGGER update_brand_updated_at BEFORE UPDATE ON catalog.brand FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();
CREATE TRIGGER update_category_updated_at BEFORE UPDATE ON catalog.category FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();

DROP FUNCTION IF EXISTS catalog.update_updated_at_column();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS catalog.product_attribute;
DROP TABLE IF EXISTS catalog.attribute_value;
DROP TABLE IF EXISTS catalog.attribute;
DROP TABLE IF EXISTS catalog.product_media;
DROP TABLE IF EXISTS catalog.product_unit;
DROP TABLE IF EXISTS catalog.product;
DROP TABLE IF EXISTS catalog.supplier;
DROP TABLE IF EXISTS catalog.handling_class;
DROP TABLE IF EXISTS catalog.unit;
DROP TABLE IF EXISTS catalog.brand;
DROP TABLE IF EXISTS catalog.category;

-- Drop enum types
DROP TYPE IF EXISTS catalog.unit_type;
DROP TYPE IF EXISTS catalog_handling_class_type;
DROP TYPE IF EXISTS catalog.product_status;