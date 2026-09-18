-- Catalog module queries
-- Products, Categories, Brands, Units, Handling Classes, Suppliers, Facets

-- name: CreateProduct :one
INSERT INTO catalog.product (
    code, slug, name, description, short_description,
    category_id, brand_id, handling_class, base_unit_id, supplier_id,
    status, is_active, is_featured, sort_order,
    base_price_minor, currency, track_inventory,
    weight_grams, length_mm, width_mm, height_mm,
    gtin, sku,
    seo_title, seo_description, seo_keywords
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21,
    $22, $23, $24, $25, $26
) RETURNING id, code, slug, name, status, created_at, updated_at;

-- name: GetProductByID :one
SELECT
    id, code, slug, name, description, short_description,
    category_id, brand_id, handling_class, base_unit_id, supplier_id,
    status, is_active, is_featured, sort_order,
    base_price_minor, currency, track_inventory,
    weight_grams, length_mm, width_mm, height_mm,
    gtin, sku,
    seo_title, seo_description, seo_keywords,
    created_at, updated_at, published_at, deleted_at
FROM catalog.product
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetProductByCode :one
SELECT
    id, code, slug, name, description, short_description,
    category_id, brand_id, handling_class, base_unit_id, supplier_id,
    status, is_active, is_featured, sort_order,
    base_price_minor, currency, track_inventory,
    weight_grams, length_mm, width_mm, height_mm,
    gtin, sku,
    seo_title, seo_description, seo_keywords,
    created_at, updated_at, published_at, deleted_at
FROM catalog.product
WHERE code = $1 AND deleted_at IS NULL;

-- name: GetProductBySlug :one
SELECT
    p.id, p.code, p.slug, p.name, p.description, p.short_description,
    p.category_id, p.brand_id, p.handling_class, p.base_unit_id, p.supplier_id,
    p.status, p.is_active, p.is_featured, p.sort_order,
    p.base_price_minor, p.currency, p.track_inventory,
    p.weight_grams, p.length_mm, p.width_mm, p.height_mm,
    p.gtin, p.sku,
    p.seo_title, p.seo_description, p.seo_keywords,
    p.created_at, p.updated_at, p.published_at,
    c.name AS category_name,
    c.slug AS category_slug,
    b.name AS brand_name,
    b.slug AS brand_slug,
    hc.name AS handling_class_name,
    hc.code AS handling_class_code,
    u.name AS base_unit_name,
    u.symbol AS base_unit_symbol,
    s.name AS supplier_name,
    s.slug AS supplier_slug
FROM catalog.product p
LEFT JOIN catalog.category c ON p.category_id = c.id AND c.deleted_at IS NULL
LEFT JOIN catalog.brand b ON p.brand_id = b.id AND b.deleted_at IS NULL
LEFT JOIN catalog.handling_class hc ON p.handling_class = hc.name
LEFT JOIN catalog.unit u ON p.base_unit_id = u.id AND u.deleted_at IS NULL
LEFT JOIN catalog.supplier s ON p.supplier_id = s.id AND s.deleted_at IS NULL
WHERE p.slug = $1 AND p.deleted_at IS NULL;

-- name: UpdateProduct :one
UPDATE catalog.product
SET
    slug = COALESCE($2, slug),
    name = COALESCE($3, name),
    description = COALESCE($4, description),
    short_description = COALESCE($5, short_description),
    category_id = COALESCE($6, category_id),
    brand_id = COALESCE($7, brand_id),
    handling_class = COALESCE($8, handling_class),
    base_unit_id = COALESCE($9, base_unit_id),
    supplier_id = COALESCE($10, supplier_id),
    status = COALESCE($11, status),
    is_active = COALESCE($12, is_active),
    is_featured = COALESCE($13, is_featured),
    sort_order = COALESCE($14, sort_order),
    base_price_minor = COALESCE($15, base_price_minor),
    currency = COALESCE($16, currency),
    track_inventory = COALESCE($17, track_inventory),
    weight_grams = COALESCE($18, weight_grams),
    length_mm = COALESCE($19, length_mm),
    width_mm = COALESCE($20, width_mm),
    height_mm = COALESCE($21, height_mm),
    gtin = COALESCE($22, gtin),
    sku = COALESCE($23, sku),
    seo_title = COALESCE($24, seo_title),
    seo_description = COALESCE($25, seo_description),
    seo_keywords = COALESCE($26, seo_keywords),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, code, slug, name, status, updated_at;

-- name: PublishProduct :exec
UPDATE catalog.product
SET status = 'published', published_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ArchiveProduct :exec
UPDATE catalog.product
SET status = 'archived', updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteProduct :exec
UPDATE catalog.product
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListProducts :many
SELECT
    id, code, slug, name, description, short_description,
    category_id, brand_id, handling_class, base_unit_id, supplier_id,
    status, is_active, is_featured, sort_order,
    base_price_minor, currency, track_inventory,
    weight_grams, length_mm, width_mm, height_mm,
    gtin, sku,
    seo_title, seo_description, seo_keywords,
    created_at, updated_at, published_at, deleted_at
FROM catalog.product,
    (SELECT
        $1::catalog.product_status AS p_status,
        $2::bigint AS p_category_id,
        $3::bigint AS p_brand_id,
        $4::catalog_handling_class_type AS p_handling_class,
        $5::bigint AS p_supplier_id,
        $6::text AS p_query
    ) params
WHERE deleted_at IS NULL
  AND (params.p_status IS NULL OR status = params.p_status)
  AND (params.p_category_id IS NULL OR category_id = params.p_category_id)
  AND (params.p_brand_id IS NULL OR brand_id = params.p_brand_id)
  AND (params.p_handling_class IS NULL OR handling_class = params.p_handling_class)
  AND (params.p_supplier_id IS NULL OR supplier_id = params.p_supplier_id)
  AND (params.p_query IS NULL OR name ILIKE '%' || params.p_query || '%' OR description ILIKE '%' || params.p_query || '%')
ORDER BY
    CASE WHEN $7 = 'name_asc' THEN name END ASC,
    CASE WHEN $7 = 'name_desc' THEN name END DESC,
    CASE WHEN $7 = 'price_asc' THEN base_price_minor END ASC,
    CASE WHEN $7 = 'price_desc' THEN base_price_minor END DESC,
    CASE WHEN $7 = 'created_desc' OR $7 IS NULL THEN created_at END DESC,
    CASE WHEN $7 = 'created_asc' THEN created_at END ASC,
    CASE WHEN $7 = 'featured' THEN is_featured END DESC
LIMIT $8 OFFSET $9;

-- name: CountProducts :one
SELECT COUNT(*)
FROM catalog.product,
    (SELECT
        $1::catalog.product_status AS p_status,
        $2::bigint AS p_category_id,
        $3::bigint AS p_brand_id,
        $4::catalog_handling_class_type AS p_handling_class,
        $5::bigint AS p_supplier_id,
        $6::text AS p_query
    ) params
WHERE deleted_at IS NULL
  AND (params.p_status IS NULL OR status = params.p_status)
  AND (params.p_category_id IS NULL OR category_id = params.p_category_id)
  AND (params.p_brand_id IS NULL OR brand_id = params.p_brand_id)
  AND (params.p_handling_class IS NULL OR handling_class = params.p_handling_class)
  AND (params.p_supplier_id IS NULL OR supplier_id = params.p_supplier_id)
  AND (params.p_query IS NULL OR name ILIKE '%' || params.p_query || '%' OR description ILIKE '%' || params.p_query || '%');

-- name: SearchProducts :many
SELECT
    p.id, p.code, p.slug, p.name, p.short_description,
    p.category_id, p.brand_id, p.handling_class, p.base_unit_id, p.supplier_id,
    p.status, p.is_active, p.is_featured, p.sort_order,
    p.base_price_minor, p.currency, p.track_inventory,
    p.created_at, p.updated_at, p.published_at,
    c.name AS category_name,
    c.slug AS category_slug,
    b.name AS brand_name,
    b.slug AS brand_slug,
    hc.name AS handling_class_name,
    hc.code AS handling_class_code,
    u.name AS base_unit_name,
    u.symbol AS base_unit_symbol,
    ts_rank_cd(
        setweight(to_tsvector('simple', p.name), 'A') ||
        setweight(to_tsvector('simple', p.description), 'B') ||
        setweight(to_tsvector('simple', p.short_description), 'B') ||
        setweight(to_tsvector('simple', c.name), 'C') ||
        setweight(to_tsvector('simple', b.name), 'C'),
        plainto_tsquery('simple', $1)
    ) AS rank
FROM catalog.product p
LEFT JOIN catalog.category c ON p.category_id = c.id AND c.deleted_at IS NULL
LEFT JOIN catalog.brand b ON p.brand_id = b.id AND b.deleted_at IS NULL
LEFT JOIN catalog.handling_class hc ON p.handling_class = hc.name
LEFT JOIN catalog.unit u ON p.base_unit_id = u.id AND u.deleted_at IS NULL
WHERE p.deleted_at IS NULL
  AND p.status = 'published'
  AND p.is_active = TRUE
  AND ($2::bigint IS NULL OR p.category_id = $2)
  AND ($3::bigint IS NULL OR p.brand_id = $3)
  AND ($4::catalog_handling_class_type IS NULL OR p.handling_class = $4)
  AND ($5::bigint IS NULL OR p.supplier_id = $5)
  AND (
    $1 = ''
    OR p.name ILIKE '%' || $1 || '%'
    OR p.description ILIKE '%' || $1 || '%'
    OR p.short_description ILIKE '%' || $1 || '%'
    OR c.name ILIKE '%' || $1 || '%'
    OR b.name ILIKE '%' || $1 || '%'
    OR p.code ILIKE '%' || $1 || '%'
    OR p.sku ILIKE '%' || $1 || '%'
  )
ORDER BY rank DESC NULLS LAST, p.is_featured DESC, p.name ASC
LIMIT $6 OFFSET $7;

-- name: CountSearchProducts :one
SELECT COUNT(*)
FROM catalog.product p
LEFT JOIN catalog.category c ON p.category_id = c.id AND c.deleted_at IS NULL
LEFT JOIN catalog.brand b ON p.brand_id = b.id AND b.deleted_at IS NULL
WHERE p.deleted_at IS NULL
  AND p.status = 'published'
  AND p.is_active = TRUE
  AND ($1::bigint IS NULL OR p.category_id = $1)
  AND ($2::bigint IS NULL OR p.brand_id = $2)
  AND ($3::catalog_handling_class_type IS NULL OR p.handling_class = $3)
  AND ($4::bigint IS NULL OR p.supplier_id = $4)
  AND (
    $5 = ''
    OR p.name ILIKE '%' || $5 || '%'
    OR p.description ILIKE '%' || $5 || '%'
    OR p.short_description ILIKE '%' || $5 || '%'
    OR c.name ILIKE '%' || $5 || '%'
    OR b.name ILIKE '%' || $5 || '%'
    OR p.code ILIKE '%' || $5 || '%'
    OR p.sku ILIKE '%' || $5 || '%'
  );

-- name: CreateCategory :one
INSERT INTO catalog.category (
    code, name, slug, description, parent_id, sort_order,
    seo_title, seo_description
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING id, code, slug, name, created_at, updated_at;

-- name: GetCategoryByID :one
SELECT id, code, name, slug, description, parent_id, sort_order,
       is_active, seo_title, seo_description, created_at, updated_at, deleted_at
FROM catalog.category
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetCategoryByCode :one
SELECT id, code, name, slug, description, parent_id, sort_order,
       is_active, seo_title, seo_description, created_at, updated_at, deleted_at
FROM catalog.category
WHERE code = $1 AND deleted_at IS NULL;

-- name: GetCategoryBySlug :one
SELECT id, code, name, slug, description, parent_id, sort_order,
       is_active, seo_title, seo_description, created_at, updated_at, deleted_at
FROM catalog.category
WHERE slug = $1 AND deleted_at IS NULL;

-- name: GetCategoryTree :many
WITH RECURSIVE category_tree AS (
    SELECT id, code, name, slug, description, parent_id, sort_order,
           is_active, seo_title, seo_description, created_at, updated_at, deleted_at, 0 as level
    FROM catalog.category
    WHERE parent_id IS NULL AND deleted_at IS NULL
    UNION ALL
    SELECT c.id, c.code, c.name, c.slug, c.description, c.parent_id, c.sort_order,
           c.is_active, c.seo_title, c.seo_description, c.created_at, c.updated_at, c.deleted_at, ct.level + 1
    FROM catalog.category c
    JOIN category_tree ct ON c.parent_id = ct.id
    WHERE c.deleted_at IS NULL
)
SELECT * FROM category_tree ORDER BY level, sort_order, name;

-- name: GetCategoryChildren :many
SELECT id, code, name, slug, description, parent_id, sort_order,
       is_active, seo_title, seo_description, created_at, updated_at, deleted_at
FROM catalog.category
WHERE parent_id = $1 AND deleted_at IS NULL
ORDER BY sort_order, name;

-- name: GetCategoryBreadcrumbs :many
WITH RECURSIVE breadcrumbs AS (
    SELECT c.id, c.code, c.name, c.slug, c.description, c.parent_id, c.sort_order,
           c.is_active, c.seo_title, c.seo_description, c.created_at, c.updated_at, c.deleted_at, 0 as level
    FROM catalog.category c
    WHERE c.id = $1 AND c.deleted_at IS NULL
    UNION ALL
    SELECT c.id, c.code, c.name, c.slug, c.description, c.parent_id, c.sort_order,
           c.is_active, c.seo_title, c.seo_description, c.created_at, c.updated_at, c.deleted_at, b.level + 1
    FROM catalog.category c
    JOIN breadcrumbs b ON c.id = b.parent_id
    WHERE c.deleted_at IS NULL
)
SELECT id, code, name, slug, description, parent_id, sort_order,
       is_active, seo_title, seo_description, created_at, updated_at, deleted_at
FROM breadcrumbs ORDER BY level DESC;

-- name: UpdateCategory :one
UPDATE catalog.category
SET
    name = COALESCE($2, name),
    slug = COALESCE($3, slug),
    description = COALESCE($4, description),
    parent_id = COALESCE($5, parent_id),
    sort_order = COALESCE($6, sort_order),
    is_active = COALESCE($7, is_active),
    seo_title = COALESCE($8, seo_title),
    seo_description = COALESCE($9, seo_description),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, code, slug, name, updated_at;

-- name: SoftDeleteCategory :exec
UPDATE catalog.category
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListCategories :many
SELECT id, code, name, slug, description, parent_id, sort_order,
       is_active, seo_title, seo_description, created_at, updated_at, deleted_at
FROM catalog.category
WHERE deleted_at IS NULL
  AND ($1::int8 IS NULL OR parent_id = $1)
ORDER BY sort_order, name
LIMIT $2 OFFSET $3;

-- name: CountCategories :one
SELECT COUNT(*)
FROM catalog.category
WHERE deleted_at IS NULL
  AND ($1::int8 IS NULL OR parent_id = $1);

-- name: CreateBrand :one
INSERT INTO catalog.brand (
    code, name, slug, description, logo_url, website_url
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING id, code, slug, name, created_at, updated_at;

-- name: GetBrandByID :one
SELECT id, code, name, slug, description, logo_url, website_url,
       is_active, created_at, updated_at, deleted_at
FROM catalog.brand
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetBrandByCode :one
SELECT id, code, name, slug, description, logo_url, website_url,
       is_active, created_at, updated_at, deleted_at
FROM catalog.brand
WHERE code = $1 AND deleted_at IS NULL;

-- name: GetBrandBySlug :one
SELECT id, code, name, slug, description, logo_url, website_url,
       is_active, created_at, updated_at, deleted_at
FROM catalog.brand
WHERE slug = $1 AND deleted_at IS NULL;

-- name: UpdateBrand :one
UPDATE catalog.brand
SET
    name = COALESCE($2, name),
    slug = COALESCE($3, slug),
    description = COALESCE($4, description),
    logo_url = COALESCE($5, logo_url),
    website_url = COALESCE($6, website_url),
    is_active = COALESCE($7, is_active),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, code, slug, name, updated_at;

-- name: SoftDeleteBrand :exec
UPDATE catalog.brand
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListBrands :many
SELECT id, code, name, slug, description, logo_url, website_url,
       is_active, created_at, updated_at, deleted_at
FROM catalog.brand
WHERE deleted_at IS NULL AND is_active = TRUE
ORDER BY name
LIMIT $1 OFFSET $2;

-- name: ListBrandsAdmin :many
SELECT id, code, name, slug, description, logo_url, website_url,
       is_active, created_at, updated_at, deleted_at
FROM catalog.brand
WHERE deleted_at IS NULL
ORDER BY name
LIMIT $1 OFFSET $2;

-- name: CountBrands :one
SELECT COUNT(*)
FROM catalog.brand
WHERE deleted_at IS NULL AND is_active = TRUE;

-- name: CreateUnit :one
INSERT INTO catalog.unit (
    code, name, symbol, unit_type, base_unit_id, conversion_factor
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING id, code, name, symbol, created_at, updated_at;

-- name: GetUnitByID :one
SELECT id, code, name, symbol, unit_type, base_unit_id, conversion_factor,
       is_active, created_at, updated_at, deleted_at
FROM catalog.unit
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUnitByCode :one
SELECT id, code, name, symbol, unit_type, base_unit_id, conversion_factor,
       is_active, created_at, updated_at, deleted_at
FROM catalog.unit
WHERE code = $1 AND deleted_at IS NULL;

-- name: UpdateUnit :one
UPDATE catalog.unit
SET
    name = COALESCE($2, name),
    symbol = COALESCE($3, symbol),
    unit_type = COALESCE($4, unit_type),
    base_unit_id = COALESCE($5, base_unit_id),
    conversion_factor = COALESCE($6, conversion_factor),
    is_active = COALESCE($7, is_active),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, code, name, symbol, updated_at;

-- name: SoftDeleteUnit :exec
UPDATE catalog.unit
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListUnits :many
SELECT id, code, name, symbol, unit_type, base_unit_id, conversion_factor,
       is_active, created_at, updated_at, deleted_at
FROM catalog.unit
WHERE deleted_at IS NULL AND is_active = TRUE
ORDER BY unit_type, name
LIMIT $1 OFFSET $2;

-- name: CountUnits :one
SELECT COUNT(*)
FROM catalog.unit
WHERE deleted_at IS NULL AND is_active = TRUE;

-- name: ListHandlingClasses :many
SELECT id, code, name, description, sort_order,
       temperature_min_c, temperature_max_c,
       created_at, updated_at, deleted_at
FROM catalog.handling_class
WHERE deleted_at IS NULL
ORDER BY sort_order, name;

-- name: GetHandlingClassByID :one
SELECT id, code, name, description, sort_order,
       temperature_min_c, temperature_max_c,
       created_at, updated_at, deleted_at
FROM catalog.handling_class
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetHandlingClassByCode :one
SELECT id, code, name, description, sort_order,
       temperature_min_c, temperature_max_c,
       created_at, updated_at, deleted_at
FROM catalog.handling_class
WHERE code = $1 AND deleted_at IS NULL;

-- name: CreateSupplier :one
INSERT INTO catalog.supplier (
    code, supplier_id, name, slug, description, logo_url
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING id, code, slug, name, created_at, updated_at;

-- name: GetSupplierByID :one
SELECT id, code, supplier_id, name, slug, description, logo_url,
       is_active, created_at, updated_at, deleted_at
FROM catalog.supplier
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetSupplierByCode :one
SELECT id, code, supplier_id, name, slug, description, logo_url,
       is_active, created_at, updated_at, deleted_at
FROM catalog.supplier
WHERE code = $1 AND deleted_at IS NULL;

-- name: GetSupplierBySlug :one
SELECT id, code, supplier_id, name, slug, description, logo_url,
       is_active, created_at, updated_at, deleted_at
FROM catalog.supplier
WHERE slug = $1 AND deleted_at IS NULL;

-- name: UpdateSupplier :one
UPDATE catalog.supplier
SET
    name = COALESCE($2, name),
    slug = COALESCE($3, slug),
    description = COALESCE($4, description),
    logo_url = COALESCE($5, logo_url),
    is_active = COALESCE($6, is_active),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, code, slug, name, updated_at;

-- name: SoftDeleteSupplier :exec
UPDATE catalog.supplier
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListSuppliers :many
SELECT id, code, supplier_id, name, slug, description, logo_url,
       is_active, created_at, updated_at, deleted_at
FROM catalog.supplier
WHERE deleted_at IS NULL AND is_active = TRUE
ORDER BY name
LIMIT $1 OFFSET $2;

-- name: ListSuppliersAdmin :many
SELECT id, code, supplier_id, name, slug, description, logo_url,
       is_active, created_at, updated_at, deleted_at
FROM catalog.supplier
WHERE deleted_at IS NULL
ORDER BY name
LIMIT $1 OFFSET $2;

-- name: CountSuppliers :one
SELECT COUNT(*)
FROM catalog.supplier
WHERE deleted_at IS NULL;

-- name: CreateProductUnit :one
INSERT INTO catalog.product_unit (
    product_id, unit_id, conversion_factor, is_default, price_minor
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING id, product_id, unit_id, conversion_factor, is_default, price_minor, is_active, created_at;

-- name: ListProductUnits :many
SELECT pu.id, pu.product_id, pu.unit_id, pu.conversion_factor, pu.is_default,
       pu.price_minor, pu.is_active, pu.created_at, pu.updated_at, pu.deleted_at,
       u.code, u.name, u.symbol
FROM catalog.product_unit pu
JOIN catalog.unit u ON pu.unit_id = u.id AND u.deleted_at IS NULL
WHERE pu.product_id = $1 AND pu.deleted_at IS NULL
ORDER BY pu.is_default DESC, u.name;

-- name: UpdateProductUnit :exec
UPDATE catalog.product_unit
SET conversion_factor = COALESCE($2, conversion_factor),
    is_default = COALESCE($3, is_default),
    price_minor = COALESCE($4, price_minor),
    is_active = COALESCE($5, is_active),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteProductUnit :exec
UPDATE catalog.product_unit
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: CreateProductMedia :one
INSERT INTO catalog.product_media (
    code, product_id, url, alt_text, media_type, sort_order,
    is_primary, width_px, height_px, file_size_bytes, mime_type
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING id, code, product_id, created_at;

-- name: ListProductMedia :many
SELECT id, code, product_id, url, alt_text, media_type, sort_order,
       is_primary, width_px, height_px, file_size_bytes, mime_type,
       created_at, updated_at, deleted_at
FROM catalog.product_media
WHERE product_id = $1 AND deleted_at IS NULL
ORDER BY is_primary DESC, sort_order;

-- name: UpdateProductMedia :exec
UPDATE catalog.product_media
SET alt_text = COALESCE($2, alt_text),
    media_type = COALESCE($3, media_type),
    sort_order = COALESCE($4, sort_order),
    is_primary = COALESCE($5, is_primary),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteProductMedia :exec
UPDATE catalog.product_media
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListProductAttributes :many
SELECT pa.id, pa.product_id, pa.attribute_id, pa.attribute_value_id,
       pa.text_value, pa.number_value, pa.boolean_value,
       pa.created_at, pa.updated_at, pa.deleted_at,
       a.code AS attribute_code, a.name AS attribute_name, a.attribute_type,
       av.value AS attribute_value_name, av.id AS attribute_value_id
FROM catalog.product_attribute pa
JOIN catalog.attribute a ON pa.attribute_id = a.id AND a.deleted_at IS NULL
LEFT JOIN catalog.attribute_value av ON pa.attribute_value_id = av.id AND av.deleted_at IS NULL
WHERE pa.product_id = $1 AND pa.deleted_at IS NULL
ORDER BY a.sort_order, a.name;

-- name: UpsertProductAttribute :one
INSERT INTO catalog.product_attribute (
    product_id, attribute_id, attribute_value_id, text_value, number_value, boolean_value
) VALUES (
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT (product_id, attribute_id, attribute_value_id) DO UPDATE SET
    text_value = EXCLUDED.text_value,
    number_value = EXCLUDED.number_value,
    boolean_value = EXCLUDED.boolean_value,
    updated_at = NOW()
RETURNING id, product_id, attribute_id, attribute_value_id, text_value, number_value, boolean_value, created_at;

-- name: DeleteProductAttribute :exec
UPDATE catalog.product_attribute
SET deleted_at = NOW(), updated_at = NOW()
WHERE product_id = $1 AND attribute_id = $2;

-- name: GetFacets :many
SELECT 'category' AS facet_type, c.code AS value_id, c.name AS value_name, COUNT(DISTINCT p.id) AS count
FROM catalog.product p
JOIN catalog.category c ON p.category_id = c.id
WHERE p.deleted_at IS NULL AND p.status = 'published' AND p.is_active = TRUE
  AND c.deleted_at IS NULL AND c.is_active = TRUE
GROUP BY c.code, c.name
UNION ALL
SELECT 'brand' AS facet_type, b.code AS value_id, b.name AS value_name, COUNT(DISTINCT p.id) AS count
FROM catalog.product p
JOIN catalog.brand b ON p.brand_id = b.id
WHERE p.deleted_at IS NULL AND p.status = 'published' AND p.is_active = TRUE
  AND b.deleted_at IS NULL AND b.is_active = TRUE
GROUP BY b.code, b.name
UNION ALL
SELECT 'handling_class' AS facet_type, p.handling_class AS value_id, hc.name AS value_name, COUNT(*) AS count
FROM catalog.product p
JOIN catalog.handling_class hc ON p.handling_class = hc.name
WHERE p.deleted_at IS NULL AND p.status = 'published' AND p.is_active = TRUE
  AND hc.deleted_at IS NULL
GROUP BY p.handling_class, hc.name
UNION ALL
SELECT 'supplier' AS facet_type, s.code AS value_id, s.name AS value_name, COUNT(DISTINCT p.id) AS count
FROM catalog.product p
JOIN catalog.supplier s ON p.supplier_id = s.id
WHERE p.deleted_at IS NULL AND p.status = 'published' AND p.is_active = TRUE
  AND s.deleted_at IS NULL AND s.is_active = TRUE
GROUP BY s.code, s.name
ORDER BY facet_type, value_name;

-- name: GetFilteredFacets :many
WITH filtered_products AS (
    SELECT p.id
    FROM catalog.product p,
        (SELECT
            $1::bigint AS p_category_id,
            $2::bigint AS p_brand_id,
            $3::catalog_handling_class_type AS p_handling_class,
            $4::bigint AS p_supplier_id,
            $5::text AS p_query
        ) params
    WHERE p.deleted_at IS NULL AND p.status = 'published' AND p.is_active = TRUE
      AND (params.p_category_id IS NULL OR p.category_id = params.p_category_id)
      AND (params.p_brand_id IS NULL OR p.brand_id = params.p_brand_id)
      AND (params.p_handling_class IS NULL OR p.handling_class = params.p_handling_class)
      AND (params.p_supplier_id IS NULL OR p.supplier_id = params.p_supplier_id)
      AND (params.p_query IS NULL OR p.name ILIKE '%' || params.p_query || '%' OR p.description ILIKE '%' || params.p_query || '%')
)
SELECT 'category' AS facet_type, c.code AS value_id, c.name AS value_name, COUNT(DISTINCT fp.id) AS count
FROM filtered_products fp
JOIN catalog.product prod ON fp.id = prod.id
JOIN catalog.category c ON prod.category_id = c.id
WHERE c.deleted_at IS NULL AND c.is_active = TRUE
GROUP BY c.code, c.name
UNION ALL
SELECT 'brand' AS facet_type, b.code AS value_id, b.name AS value_name, COUNT(DISTINCT fp.id) AS count
FROM filtered_products fp
JOIN catalog.product prod ON fp.id = prod.id
JOIN catalog.brand b ON prod.brand_id = b.id
WHERE b.deleted_at IS NULL AND b.is_active = TRUE
GROUP BY b.code, b.name
UNION ALL
SELECT 'handling_class' AS facet_type, prod.handling_class AS value_id, hc.name AS value_name, COUNT(DISTINCT fp.id) AS count
FROM filtered_products fp
JOIN catalog.product prod ON fp.id = prod.id
JOIN catalog.handling_class hc ON prod.handling_class = hc.name
WHERE hc.deleted_at IS NULL
GROUP BY prod.handling_class, hc.name
UNION ALL
SELECT 'supplier' AS facet_type, s.code AS value_id, s.name AS value_name, COUNT(DISTINCT fp.id) AS count
FROM filtered_products fp
JOIN catalog.product prod ON fp.id = prod.id
JOIN catalog.supplier s ON prod.supplier_id = s.id
WHERE s.deleted_at IS NULL AND s.is_active = TRUE
GROUP BY s.code, s.name
ORDER BY facet_type, value_name;

-- name: GetAttributeFacets :many
SELECT
    a.id AS attribute_id,
    a.code AS attribute_code,
    a.name AS attribute_name,
    a.attribute_type,
    av.id AS value_id,
    av.value AS value_name,
    COUNT(DISTINCT pa.product_id) AS count
FROM catalog.product_attribute pa
JOIN catalog.attribute a ON pa.attribute_id = a.id AND a.deleted_at IS NULL
JOIN catalog.attribute_value av ON pa.attribute_value_id = av.id AND av.deleted_at IS NULL
JOIN catalog.product p ON pa.product_id = p.id AND p.deleted_at IS NULL AND p.status = 'published' AND p.is_active = TRUE
WHERE a.is_filterable = TRUE
  AND ($1::int8 IS NULL OR a.id = $1)
GROUP BY a.id, a.code, a.name, a.attribute_type, av.id, av.value
ORDER BY a.sort_order, a.name, av.sort_order, av.value;