-- Pricing module queries
-- Price Lists, Price Items, Quantity Tiers, Buyer Assignments

-- name: CreatePriceList :one
INSERT INTO pricing.price_list (
    code, name, description, status, price_type, currency,
    effective_from, effective_to, is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING id, code, name, status, created_at, updated_at;

-- name: GetPriceListByID :one
SELECT
    id, code, name, description, status, price_type, currency,
    effective_from, effective_to, is_active,
    created_at, updated_at, deleted_at
FROM pricing.price_list
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetPriceListByCode :one
SELECT
    id, code, name, description, status, price_type, currency,
    effective_from, effective_to, is_active,
    created_at, updated_at, deleted_at
FROM pricing.price_list
WHERE code = $1 AND deleted_at IS NULL;

-- name: UpdatePriceList :one
UPDATE pricing.price_list
SET
    name = COALESCE($2, name),
    description = COALESCE($3, description),
    status = COALESCE($4, status),
    price_type = COALESCE($5, price_type),
    currency = COALESCE($6, currency),
    effective_from = COALESCE($7, effective_from),
    effective_to = COALESCE($8, effective_to),
    is_active = COALESCE($9, is_active),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, code, name, status, updated_at;

-- name: SoftDeletePriceList :exec
UPDATE pricing.price_list
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListPriceLists :many
SELECT
    id, code, name, description, status, price_type, currency,
    effective_from, effective_to, is_active,
    created_at, updated_at, deleted_at
FROM pricing.price_list
WHERE deleted_at IS NULL
  AND ($1::pricing.price_list_status IS NULL OR status = $1)
  AND ($2::boolean IS NULL OR is_active = $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountPriceLists :one
SELECT COUNT(*)
FROM pricing.price_list
WHERE deleted_at IS NULL
  AND ($1::pricing.price_list_status IS NULL OR status = $1)
  AND ($2::boolean IS NULL OR is_active = $2);

-- name: GetActivePriceLists :many
SELECT
    id, code, name, description, status, price_type, currency,
    effective_from, effective_to, is_active,
    created_at, updated_at, deleted_at
FROM pricing.price_list
WHERE deleted_at IS NULL
  AND is_active = TRUE
  AND status = 'active'
  AND effective_from <= NOW()
  AND (effective_to IS NULL OR effective_to > NOW())
ORDER BY created_at DESC;

-- name: CreatePriceListItem :one
INSERT INTO pricing.price_list_item (
    price_list_id, product_id, unit_id, currency, price_minor,
    min_quantity, max_quantity, effective_from, effective_to, is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING id, price_list_id, product_id, unit_id, currency, price_minor,
    min_quantity, max_quantity, effective_from, effective_to, is_active,
    created_at, updated_at;

-- name: GetPriceListItemByID :one
SELECT
    id, price_list_id, product_id, unit_id, currency, price_minor,
    min_quantity, max_quantity, effective_from, effective_to, is_active,
    created_at, updated_at, deleted_at
FROM pricing.price_list_item
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdatePriceListItem :one
UPDATE pricing.price_list_item
SET
    currency = COALESCE($2, currency),
    price_minor = COALESCE($3, price_minor),
    min_quantity = COALESCE($4, min_quantity),
    max_quantity = COALESCE($5, max_quantity),
    effective_from = COALESCE($6, effective_from),
    effective_to = COALESCE($7, effective_to),
    is_active = COALESCE($8, is_active),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, price_list_id, product_id, unit_id, currency, price_minor,
    min_quantity, max_quantity, effective_from, effective_to, is_active,
    updated_at;

-- name: SoftDeletePriceListItem :exec
UPDATE pricing.price_list_item
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListPriceListItems :many
SELECT
    id, price_list_id, product_id, unit_id, currency, price_minor,
    min_quantity, max_quantity, effective_from, effective_to, is_active,
    created_at, updated_at, deleted_at
FROM pricing.price_list_item
WHERE deleted_at IS NULL
  AND ($1::bigint IS NULL OR price_list_id = $1)
  AND ($2::bigint IS NULL OR product_id = $2)
  AND ($3::bigint IS NULL OR unit_id = $3)
  AND ($4::boolean IS NULL OR is_active = $4)
ORDER BY product_id, min_quantity
LIMIT $5 OFFSET $6;

-- name: CountPriceListItems :one
SELECT COUNT(*)
FROM pricing.price_list_item
WHERE deleted_at IS NULL
  AND ($1::bigint IS NULL OR price_list_id = $1)
  AND ($2::bigint IS NULL OR product_id = $2)
  AND ($3::bigint IS NULL OR unit_id = $3)
  AND ($4::boolean IS NULL OR is_active = $4);

-- name: GetPriceListItemsByList :many
SELECT
    id, price_list_id, product_id, unit_id, currency, price_minor,
    min_quantity, max_quantity, effective_from, effective_to, is_active,
    created_at, updated_at, deleted_at
FROM pricing.price_list_item
WHERE deleted_at IS NULL
  AND price_list_id = $1
  AND is_active = TRUE
  AND effective_from <= NOW()
  AND (effective_to IS NULL OR effective_to > NOW())
ORDER BY product_id, min_quantity;

-- name: CreateQuantityTier :one
INSERT INTO pricing.quantity_tier (
    price_list_item_id, min_quantity, max_quantity, price_minor, is_active
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING id, price_list_item_id, min_quantity, max_quantity, price_minor,
    is_active, created_at, updated_at;

-- name: GetQuantityTierByID :one
SELECT
    id, price_list_item_id, min_quantity, max_quantity, price_minor,
    is_active, created_at, updated_at, deleted_at
FROM pricing.quantity_tier
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateQuantityTier :one
UPDATE pricing.quantity_tier
SET
    min_quantity = COALESCE($2, min_quantity),
    max_quantity = COALESCE($3, max_quantity),
    price_minor = COALESCE($4, price_minor),
    is_active = COALESCE($5, is_active),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, price_list_item_id, min_quantity, max_quantity, price_minor,
    is_active, updated_at;

-- name: SoftDeleteQuantityTier :exec
UPDATE pricing.quantity_tier
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListQuantityTiers :many
SELECT
    id, price_list_item_id, min_quantity, max_quantity, price_minor,
    is_active, created_at, updated_at, deleted_at
FROM pricing.quantity_tier
WHERE deleted_at IS NULL
  AND ($1::bigint IS NULL OR price_list_item_id = $1)
  AND ($2::boolean IS NULL OR is_active = $2)
ORDER BY min_quantity;

-- name: GetQuantityTiersByItem :many
SELECT
    id, price_list_item_id, min_quantity, max_quantity, price_minor,
    is_active, created_at, updated_at, deleted_at
FROM pricing.quantity_tier
WHERE deleted_at IS NULL
  AND price_list_item_id = $1
  AND is_active = TRUE
ORDER BY min_quantity;

-- name: CreatePriceListAssignment :one
INSERT INTO pricing.price_list_assignment (
    code, price_list_id, buyer_profile_id, assigned_by,
    effective_from, effective_to, is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING id, code, price_list_id, buyer_profile_id, assigned_by,
    effective_from, effective_to, is_active, created_at, updated_at;

-- name: GetPriceListAssignmentByID :one
SELECT
    id, code, price_list_id, buyer_profile_id, assigned_by,
    effective_from, effective_to, is_active,
    created_at, updated_at, deleted_at
FROM pricing.price_list_assignment
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetPriceListAssignmentByCode :one
SELECT
    id, code, price_list_id, buyer_profile_id, assigned_by,
    effective_from, effective_to, is_active,
    created_at, updated_at, deleted_at
FROM pricing.price_list_assignment
WHERE code = $1 AND deleted_at IS NULL;

-- name: UpdatePriceListAssignment :one
UPDATE pricing.price_list_assignment
SET
    price_list_id = COALESCE($2, price_list_id),
    effective_from = COALESCE($3, effective_from),
    effective_to = COALESCE($4, effective_to),
    is_active = COALESCE($5, is_active),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, code, price_list_id, buyer_profile_id, assigned_by,
    effective_from, effective_to, is_active, updated_at;

-- name: SoftDeletePriceListAssignment :exec
UPDATE pricing.price_list_assignment
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: ListPriceListAssignments :many
SELECT
    id, code, price_list_id, buyer_profile_id, assigned_by,
    effective_from, effective_to, is_active,
    created_at, updated_at, deleted_at
FROM pricing.price_list_assignment
WHERE deleted_at IS NULL
  AND ($1::bigint IS NULL OR price_list_id = $1)
  AND ($2::bigint IS NULL OR buyer_profile_id = $2)
  AND ($3::boolean IS NULL OR is_active = $3)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: CountPriceListAssignments :one
SELECT COUNT(*)
FROM pricing.price_list_assignment
WHERE deleted_at IS NULL
  AND ($1::bigint IS NULL OR price_list_id = $1)
  AND ($2::bigint IS NULL OR buyer_profile_id = $2)
  AND ($3::boolean IS NULL OR is_active = $3);

-- name: GetActiveAssignmentForBuyer :one
SELECT
    id, code, price_list_id, buyer_profile_id, assigned_by,
    effective_from, effective_to, is_active,
    created_at, updated_at, deleted_at
FROM pricing.price_list_assignment
WHERE deleted_at IS NULL
  AND buyer_profile_id = $1
  AND is_active = TRUE
  AND effective_from <= NOW()
  AND (effective_to IS NULL OR effective_to > NOW())
ORDER BY effective_from DESC
LIMIT 1;

-- name: GetPriceListWithItems :one
SELECT
    pl.id, pl.code, pl.name, pl.description, pl.status, pl.price_type, pl.currency,
    pl.effective_from, pl.effective_to, pl.is_active,
    pl.created_at, pl.updated_at, pl.deleted_at
FROM pricing.price_list pl
WHERE pl.id = $1 AND pl.deleted_at IS NULL;

-- name: GetPriceForProduct :many
SELECT
    pli.id, pli.price_list_id, pli.product_id, pli.unit_id, pli.currency,
    pli.price_minor, pli.min_quantity, pli.max_quantity,
    pli.effective_from, pli.effective_to, pli.is_active,
    pli.created_at, pli.updated_at, pli.deleted_at,
    qt.id as tier_id, qt.min_quantity as tier_min_quantity,
    qt.max_quantity as tier_max_quantity, qt.price_minor as tier_price_minor
FROM pricing.price_list_item pli
LEFT JOIN pricing.quantity_tier qt ON qt.price_list_item_id = pli.id AND qt.deleted_at IS NULL AND qt.is_active = TRUE
WHERE pli.deleted_at IS NULL
  AND pli.price_list_id = $1
  AND pli.product_id = $2
  AND pli.unit_id = $3
  AND pli.is_active = TRUE
  AND pli.effective_from <= NOW()
  AND (pli.effective_to IS NULL OR pli.effective_to > NOW())
ORDER BY pli.min_quantity, qt.min_quantity;

-- name: GetPriceForQuote :many
SELECT
    pli.id, pli.price_list_id, pli.product_id, pli.unit_id, pli.currency,
    pli.price_minor, pli.min_quantity, pli.max_quantity,
    pli.effective_from, pli.effective_to, pli.is_active,
    qt.id as tier_id, qt.min_quantity as tier_min_quantity,
    qt.max_quantity as tier_max_quantity, qt.price_minor as tier_price_minor
FROM pricing.price_list_item pli
LEFT JOIN pricing.quantity_tier qt ON qt.price_list_item_id = pli.id AND qt.deleted_at IS NULL AND qt.is_active = TRUE
WHERE pli.deleted_at IS NULL
  AND pli.price_list_id = $1
  AND pli.product_id = $2
  AND pli.unit_id = $3
  AND pli.is_active = TRUE
  AND pli.effective_from <= NOW()
  AND (pli.effective_to IS NULL OR pli.effective_to > NOW())
ORDER BY pli.min_quantity, qt.min_quantity;