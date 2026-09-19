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
