-- name: CreateStockLevel :one
INSERT INTO inventory.stock_level (
    code, product_id, supplier_id, available_quantity, reserved_quantity, total_quantity, safety_stock
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetStockLevelByID :one
SELECT * FROM inventory.stock_level
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetStockLevelByProductAndSupplier :one
SELECT * FROM inventory.stock_level
WHERE product_id = $1 AND supplier_id = $2 AND deleted_at IS NULL;

-- name: ListStockLevels :many
SELECT * FROM inventory.stock_level
WHERE product_id = $1 AND deleted_at IS NULL;

-- name: UpdateStockLevelQuantities :one
UPDATE inventory.stock_level
SET available_quantity = $2,
    reserved_quantity = $3,
    total_quantity = $4,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: CreateLot :one
INSERT INTO inventory.lot (
    code, stock_level_id, lot_number, initial_quantity, available_quantity, reserved_quantity, status, is_quarantined, production_date, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetLotByID :one
SELECT * FROM inventory.lot
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetLotByStockAndNumberForUpdate :one
SELECT * FROM inventory.lot
WHERE stock_level_id = $1 AND lot_number = $2 AND deleted_at IS NULL
FOR UPDATE;

-- name: GetLotByCode :one
SELECT * FROM inventory.lot
WHERE code = $1 AND deleted_at IS NULL;

-- name: GetFEFOLotsForUpdate :many
SELECT * FROM inventory.lot
WHERE stock_level_id = $1
  AND deleted_at IS NULL
  AND is_quarantined = FALSE
  AND available_quantity > 0
  AND status = 'active'
ORDER BY expires_at ASC, id ASC
FOR UPDATE;

-- name: UpdateLotQuantities :one
UPDATE inventory.lot
SET available_quantity = $2,
    reserved_quantity = $3,
    status = $4,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: CreateReservation :one
INSERT INTO inventory.reservation (
    code, lot_id, order_line_id, request_id, quantity, status, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetReservationByID :one
SELECT * FROM inventory.reservation
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetReservationByCode :one
SELECT * FROM inventory.reservation
WHERE code = $1 AND deleted_at IS NULL;

-- name: GetReservationsByRequestID :many
SELECT * FROM inventory.reservation
WHERE request_id = $1 AND deleted_at IS NULL;

-- name: FindExpiredReservations :many
SELECT * FROM inventory.reservation
WHERE status = 'reserved'
  AND expires_at < NOW()
  AND deleted_at IS NULL
FOR UPDATE;

-- name: UpdateReservationStatus :one
UPDATE inventory.reservation
SET status = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: QuarantineLot :one
UPDATE inventory.lot
SET is_quarantined = TRUE,
    status = 'quarantined',
    available_quantity = 0,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: CreateQuarantineRecord :one
INSERT INTO inventory.quarantine_record (
    code, lot_id, reason, status, adjusted_quantity, adjusted_by
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetStockLevelByIDForUpdate :one
SELECT * FROM inventory.stock_level
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE;

-- name: GetLotByIDForUpdate :one
SELECT * FROM inventory.lot
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE;

-- name: GetReservationsByRequestIDForUpdate :many
SELECT * FROM inventory.reservation
WHERE request_id = $1 AND deleted_at IS NULL
FOR UPDATE;

-- name: ReleaseLot :one
UPDATE inventory.lot
SET is_quarantined = FALSE,
    status = 'active',
    available_quantity = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ListLowStockLots :many
SELECT l.*, sl.product_id, sl.supplier_id, sl.safety_stock,
       p.code as product_code, p.name as product_name,
       s.code as supplier_code
FROM inventory.lot l
JOIN inventory.stock_level sl ON sl.id = l.stock_level_id
JOIN catalog.product p ON p.id = sl.product_id
JOIN catalog.supplier s ON s.id = sl.supplier_id
WHERE l.deleted_at IS NULL
  AND l.is_quarantined = FALSE
  AND l.status = 'active'
  AND l.available_quantity < sl.safety_stock
ORDER BY (sl.safety_stock - l.available_quantity) DESC, l.expires_at ASC;

-- name: ListExpiringLots :many
SELECT l.*, sl.product_id, sl.supplier_id,
       p.code as product_code, p.name as product_name,
       s.code as supplier_code
FROM inventory.lot l
JOIN inventory.stock_level sl ON sl.id = l.stock_level_id
JOIN catalog.product p ON p.id = sl.product_id
JOIN catalog.supplier s ON s.id = sl.supplier_id
WHERE l.deleted_at IS NULL
  AND l.is_quarantined = FALSE
  AND l.status = 'active'
  AND l.expires_at IS NOT NULL
  AND l.expires_at <= NOW() + INTERVAL '$1 days'
ORDER BY l.expires_at ASC;

-- name: AdjustLotQuantities :one
UPDATE inventory.lot
SET available_quantity = available_quantity + $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: CreateStockAdjustment :one
INSERT INTO inventory.stock_adjustment (
    code, lot_id, quantity_delta, previous_quantity, new_quantity,
    reason_code, reason, adjusted_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: UpdateStockLevelQuantitiesOnAdjustment :one
UPDATE inventory.stock_level
SET available_quantity = available_quantity + $2,
    total_quantity = total_quantity + $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetQuarantineRecordByLot :one
SELECT * FROM inventory.quarantine_record
WHERE lot_id = $1 AND status = 'quarantined' AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdateQuarantineRecordStatus :one
UPDATE inventory.quarantine_record
SET status = 'released',
    adjusted_quantity = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetQuarantineRecordByLotAndStatus :one
SELECT * FROM inventory.quarantine_record
WHERE lot_id = $1 AND status = $2 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 1;
