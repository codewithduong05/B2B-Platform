-- Catalog module schema
-- Creates tables for products, categories, brands, units, attributes, media

-- Ensure catalog schema exists
CREATE SCHEMA IF NOT EXISTS catalog;

-- Enum types
CREATE TYPE catalog.product_status AS ENUM ('draft', 'published', 'archived');
CREATE TYPE catalog_handling_class_type AS ENUM ('ambient', 'chilled', 'frozen');
CREATE TYPE catalog.unit_type AS ENUM ('base', 'derived');

-- Categories (hierarchical)
CREATE TABLE catalog.category (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,  -- opaque public identifier
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    parent_id BIGINT REFERENCES catalog.category (id) ON DELETE SET NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    seo_title VARCHAR(255),
    seo_description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_catalog_category_parent ON catalog.category (parent_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_catalog_category_slug ON catalog.category (slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_catalog_category_active ON catalog.category (is_active) WHERE deleted_at IS NULL;

-- Brands
CREATE TABLE catalog.brand (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    logo_url VARCHAR(500),
    website_url VARCHAR(500),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_catalog_brand_slug ON catalog.brand (slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_catalog_brand_active ON catalog.brand (is_active) WHERE deleted_at IS NULL;

-- Units of Measure
CREATE TABLE catalog.unit (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    unit_type catalog.unit_type NOT NULL DEFAULT 'base',
    base_unit_id BIGINT REFERENCES catalog.unit (id) ON DELETE SET NULL,
    conversion_factor NUMERIC(18, 6) NOT NULL DEFAULT 1,  -- factor to convert to base unit
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_catalog_unit_active ON catalog.unit (is_active) WHERE deleted_at IS NULL;

-- Handling Classes
CREATE TABLE catalog.handling_class (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    name catalog_handling_class_type NOT NULL,  -- 'ambient', 'chilled', 'frozen'
    description TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    temperature_min_c INT,
    temperature_max_c INT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_catalog_handling_class_active ON catalog.handling_class (is_active) WHERE deleted_at IS NULL;

-- Suppliers (as listed in catalog - reference to suppliers module)
CREATE TABLE catalog.supplier (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    supplier_id BIGINT NOT NULL,  -- references suppliers.supplier_profile.id
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    logo_url VARCHAR(500),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_catalog_supplier_slug ON catalog.supplier (slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_catalog_supplier_active ON catalog.supplier (is_active) WHERE deleted_at IS NULL;

-- Products
CREATE TABLE catalog.product (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,  -- opaque public identifier
    slug VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    short_description VARCHAR(500),
    category_id BIGINT NOT NULL REFERENCES catalog.category (id) ON DELETE RESTRICT,
    brand_id BIGINT REFERENCES catalog.brand (id) ON DELETE SET NULL,
    handling_class catalog_handling_class_type NOT NULL,
    base_unit_id BIGINT NOT NULL REFERENCES catalog.unit (id) ON DELETE RESTRICT,
    supplier_id BIGINT REFERENCES catalog.supplier (id) ON DELETE SET NULL,
    status catalog.product_status NOT NULL DEFAULT 'draft',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_featured BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    -- SEO
    seo_title VARCHAR(255),
    seo_description TEXT,
    seo_keywords TEXT,
    -- Pricing reference (actual pricing in pricing module)
    base_price_minor BIGINT,  -- reference price in minor units
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    -- Inventory reference
    track_inventory BOOLEAN NOT NULL DEFAULT TRUE,
    -- Dimensions
    weight_grams INT,
    length_mm INT,
    width_mm INT,
    height_mm INT,
    -- Metadata
    gtin VARCHAR(14),  -- Global Trade Item Number
    sku VARCHAR(100),  -- Supplier SKU
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_catalog_product_category ON catalog.product (category_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_catalog_product_brand ON catalog.product (brand_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_catalog_product_handling ON catalog.product (handling_class) WHERE deleted_at IS NULL;
CREATE INDEX idx_catalog_product_supplier ON catalog.product (supplier_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_catalog_product_slug ON catalog.product (slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_catalog_product_status ON catalog.product (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_catalog_product_active ON catalog.product (is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_catalog_product_featured ON catalog.product (is_featured) WHERE deleted_at IS NULL AND is_featured = TRUE;

-- Product Units (UoM conversions for a product)
CREATE TABLE catalog.product_unit (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES catalog.product (id) ON DELETE CASCADE,
    unit_id BIGINT NOT NULL REFERENCES catalog.unit (id) ON DELETE RESTRICT,
    conversion_factor NUMERIC(18, 6) NOT NULL,  -- how many base units in this unit
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    price_minor BIGINT,  -- price for this unit (optional, pricing module handles actual pricing)
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (product_id, unit_id)
);

CREATE INDEX idx_catalog_product_unit_product ON catalog.product_unit (product_id) WHERE deleted_at IS NULL;

-- Product Media
CREATE TABLE catalog.product_media (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    product_id BIGINT NOT NULL REFERENCES catalog.product (id) ON DELETE CASCADE,
    url VARCHAR(500) NOT NULL,
    alt_text VARCHAR(255),
    media_type VARCHAR(50) NOT NULL DEFAULT 'image',  -- 'image', 'video', 'document'
    sort_order INT NOT NULL DEFAULT 0,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    width_px INT,
    height_px INT,
    file_size_bytes BIGINT,
    mime_type VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_catalog_product_media_product ON catalog.product_media (product_id) WHERE deleted_at IS NULL;

-- Product Attributes (for faceted search)
CREATE TABLE catalog.attribute (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    attribute_type VARCHAR(50) NOT NULL,  -- 'text', 'number', 'boolean', 'select', 'multiselect'
    is_filterable BOOLEAN NOT NULL DEFAULT FALSE,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_catalog_attribute_filterable ON catalog.attribute (is_filterable) WHERE deleted_at IS NULL AND is_filterable = TRUE;

-- Attribute Values (for select/multiselect types)
CREATE TABLE catalog.attribute_value (
    id BIGSERIAL PRIMARY KEY,
    attribute_id BIGINT NOT NULL REFERENCES catalog.attribute (id) ON DELETE CASCADE,
    value VARCHAR(255) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_catalog_attribute_value_attribute ON catalog.attribute_value (attribute_id) WHERE deleted_at IS NULL;

-- Product Attribute Values (EAV model)
CREATE TABLE catalog.product_attribute (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES catalog.product (id) ON DELETE CASCADE,
    attribute_id BIGINT NOT NULL REFERENCES catalog.attribute (id) ON DELETE CASCADE,
    attribute_value_id BIGINT REFERENCES catalog.attribute_value (id) ON DELETE SET NULL,
    text_value TEXT,
    number_value NUMERIC(18, 6),
    boolean_value BOOLEAN,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (product_id, attribute_id, attribute_value_id)
);

CREATE INDEX idx_catalog_product_attribute_product ON catalog.product_attribute (product_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_catalog_product_attribute_attr ON catalog.product_attribute (attribute_id) WHERE deleted_at IS NULL;

-- Trigger to update updated_at timestamps
CREATE OR REPLACE FUNCTION catalog.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_category_updated_at BEFORE UPDATE ON catalog.category FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();
CREATE TRIGGER update_brand_updated_at BEFORE UPDATE ON catalog.brand FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();
CREATE TRIGGER update_unit_updated_at BEFORE UPDATE ON catalog.unit FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();
CREATE TRIGGER update_handling_class_updated_at BEFORE UPDATE ON catalog.handling_class FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();
CREATE TRIGGER update_supplier_updated_at BEFORE UPDATE ON catalog.supplier FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();
CREATE TRIGGER update_product_updated_at BEFORE UPDATE ON catalog.product FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();
CREATE TRIGGER update_product_unit_updated_at BEFORE UPDATE ON catalog.product_unit FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();
CREATE TRIGGER update_product_media_updated_at BEFORE UPDATE ON catalog.product_media FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();
CREATE TRIGGER update_attribute_updated_at BEFORE UPDATE ON catalog.attribute FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();
CREATE TRIGGER update_attribute_value_updated_at BEFORE UPDATE ON catalog.attribute_value FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();
CREATE TRIGGER update_product_attribute_updated_at BEFORE UPDATE ON catalog.product_attribute FOR EACH ROW EXECUTE FUNCTION catalog.update_updated_at_column();