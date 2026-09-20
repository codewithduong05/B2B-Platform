package repository

import (
	"context"
			"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/database/queries/inventory"
		"github.com/jackc/pgx/v5/pgtype"
)

// Local types for new query results
type ListLowStockLotsRow struct {
	ID                int64
	Code              string
	StockLevelID      int64
	LotNumber         string
	InitialQuantity   int32
	AvailableQuantity int32
	ReservedQuantity  int32
	Status            string
	IsQuarantined     bool
	ProductionDate    pgtype.Date
	ExpiresAt         pgtype.Timestamptz
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         pgtype.Timestamptz
	ProductID         int64
	SupplierID        int64
	SafetyStock       int32
	ProductCode       pgtype.Text
	ProductName       pgtype.Text
	SupplierCode      pgtype.Text
}

type ListExpiringLotsRow struct {
	ID                int64
	Code              string
	StockLevelID      int64
	LotNumber         string
	InitialQuantity   int32
	AvailableQuantity int32
	ReservedQuantity  int32
	Status            string
	IsQuarantined     bool
	ProductionDate    pgtype.Date
	ExpiresAt         pgtype.Timestamptz
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         pgtype.Timestamptz
	ProductID         int64
	SupplierID        int64
	ProductCode       pgtype.Text
	ProductName       pgtype.Text
	SupplierCode      pgtype.Text
}

type CreateStockAdjustmentRow struct {
	ID               int64
	Code             string
	LotID            int64
	QuantityDelta    int32
	PreviousQuantity int32
	NewQuantity      int32
	ReasonCode       string
	Reason           string
	AdjustedBy       pgtype.Int8
	CreatedAt        time.Time
}

type GetQuarantineRecordByLotRow struct {
	ID               int64
	Code             string
	LotID            int64
	Status           string
	Reason           string
	AdjustedQuantity int32
	AdjustedBy       int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type InventoryRepository struct {
	db *database.DB
	q  *inventory.Queries
}

func NewInventoryRepository(db *database.DB) *InventoryRepository {
	return &InventoryRepository{
		db: db,
		q:  inventory.New(db.Pool),
	}
}

func NewInventoryRepositoryWithTx(tx *database.Tx) *InventoryRepository {
	return &InventoryRepository{
		q: inventory.New(tx),
	}
}

func (r *InventoryRepository) CreateStockLevel(ctx context.Context, params inventory.CreateStockLevelParams) (inventory.InventoryStockLevel, error) {
	return r.q.CreateStockLevel(ctx, params)
}

func (r *InventoryRepository) GetStockLevelByID(ctx context.Context, id int64) (inventory.InventoryStockLevel, error) {
	return r.q.GetStockLevelByID(ctx, id)
}

func (r *InventoryRepository) GetStockLevelByIDForUpdate(ctx context.Context, id int64) (inventory.InventoryStockLevel, error) {
	return r.q.GetStockLevelByIDForUpdate(ctx, id)
}

func (r *InventoryRepository) GetStockLevelByProductAndSupplier(ctx context.Context, productID, supplierID int64) (inventory.InventoryStockLevel, error) {
	return r.q.GetStockLevelByProductAndSupplier(ctx, inventory.GetStockLevelByProductAndSupplierParams{
		ProductID:  productID,
		SupplierID: supplierID,
	})
}

func (r *InventoryRepository) ListStockLevels(ctx context.Context, productID int64) ([]inventory.InventoryStockLevel, error) {
	return r.q.ListStockLevels(ctx, productID)
}

func (r *InventoryRepository) UpdateStockLevelQuantities(ctx context.Context, id int64, available, reserved, total int32) (inventory.InventoryStockLevel, error) {
	return r.q.UpdateStockLevelQuantities(ctx, inventory.UpdateStockLevelQuantitiesParams{
		ID:                id,
		AvailableQuantity: available,
		ReservedQuantity:  reserved,
		TotalQuantity:     total,
	})
}

func (r *InventoryRepository) CreateLot(ctx context.Context, params inventory.CreateLotParams) (inventory.InventoryLot, error) {
	return r.q.CreateLot(ctx, params)
}

func (r *InventoryRepository) GetLotByID(ctx context.Context, id int64) (inventory.InventoryLot, error) {
	return r.q.GetLotByID(ctx, id)
}

func (r *InventoryRepository) GetLotByIDForUpdate(ctx context.Context, id int64) (inventory.InventoryLot, error) {
	return r.q.GetLotByIDForUpdate(ctx, id)
}

func (r *InventoryRepository) GetLotByStockAndNumberForUpdate(ctx context.Context, stockLevelID int64, lotNumber string) (inventory.InventoryLot, error) {
	return r.q.GetLotByStockAndNumberForUpdate(ctx, inventory.GetLotByStockAndNumberForUpdateParams{
		StockLevelID: stockLevelID,
		LotNumber:    lotNumber,
	})
}

func (r *InventoryRepository) GetFEFOLotsForUpdate(ctx context.Context, stockLevelID int64) ([]inventory.InventoryLot, error) {
	return r.q.GetFEFOLotsForUpdate(ctx, stockLevelID)
}

func (r *InventoryRepository) UpdateLotQuantities(ctx context.Context, id int64, available, reserved int32, status inventory.InventoryLotStatus) (inventory.InventoryLot, error) {
	return r.q.UpdateLotQuantities(ctx, inventory.UpdateLotQuantitiesParams{
		ID:                id,
		AvailableQuantity: available,
		ReservedQuantity:  reserved,
		Status:            status,
	})
}

func (r *InventoryRepository) CreateReservation(ctx context.Context, params inventory.CreateReservationParams) (inventory.InventoryReservation, error) {
	return r.q.CreateReservation(ctx, params)
}

func (r *InventoryRepository) GetReservationByID(ctx context.Context, id int64) (inventory.InventoryReservation, error) {
	return r.q.GetReservationByID(ctx, id)
}

func (r *InventoryRepository) GetReservationsByRequestID(ctx context.Context, requestID string) ([]inventory.InventoryReservation, error) {
	return r.q.GetReservationsByRequestID(ctx, requestID)
}

func (r *InventoryRepository) GetReservationsByRequestIDForUpdate(ctx context.Context, requestID string) ([]inventory.InventoryReservation, error) {
	return r.q.GetReservationsByRequestIDForUpdate(ctx, requestID)
}

func (r *InventoryRepository) FindExpiredReservations(ctx context.Context) ([]inventory.InventoryReservation, error) {
	return r.q.FindExpiredReservations(ctx)
}

func (r *InventoryRepository) UpdateReservationStatus(ctx context.Context, id int64, status inventory.InventoryReservationStatus) (inventory.InventoryReservation, error) {
	return r.q.UpdateReservationStatus(ctx, inventory.UpdateReservationStatusParams{
		ID:     id,
		Status: status,
	})
}

func (r *InventoryRepository) QuarantineLot(ctx context.Context, id int64) (inventory.InventoryLot, error) {
	return r.q.QuarantineLot(ctx, id)
}

func (r *InventoryRepository) CreateQuarantineRecord(ctx context.Context, params inventory.CreateQuarantineRecordParams) (inventory.InventoryQuarantineRecord, error) {
	return r.q.CreateQuarantineRecord(ctx, params)
}

func (r *InventoryRepository) ReleaseLot(ctx context.Context, id int64, availableQuantity int32) (inventory.InventoryLot, error) {
	var lot inventory.InventoryLot
	err := r.db.Pool.QueryRow(ctx, `
		UPDATE inventory.lot
		SET is_quarantined = FALSE,
		    status = 'active',
		    available_quantity = $2,
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, stock_level_id, lot_number, initial_quantity, available_quantity, reserved_quantity, status, is_quarantined, production_date, expires_at, created_at, updated_at, deleted_at
	`, id, availableQuantity).Scan(
		&lot.ID, &lot.Code, &lot.StockLevelID, &lot.LotNumber, &lot.InitialQuantity,
		&lot.AvailableQuantity, &lot.ReservedQuantity, &lot.Status, &lot.IsQuarantined,
		&lot.ProductionDate, &lot.ExpiresAt, &lot.CreatedAt, &lot.UpdatedAt, &lot.DeletedAt,
	)
	return lot, err
}

func (r *InventoryRepository) ListLowStockLots(ctx context.Context) ([]ListLowStockLotsRow, error) {
	rows, err := r.db.Pool.Query(ctx, `
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
		ORDER BY (sl.safety_stock - l.available_quantity) DESC, l.expires_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lots []ListLowStockLotsRow
	for rows.Next() {
		var l ListLowStockLotsRow
		var pc, pn, sc pgtype.Text
		err := rows.Scan(&l.ID, &l.Code, &l.StockLevelID, &l.LotNumber, &l.InitialQuantity,
			&l.AvailableQuantity, &l.ReservedQuantity, &l.Status, &l.IsQuarantined,
			&l.ProductionDate, &l.ExpiresAt, &l.CreatedAt, &l.UpdatedAt, &l.DeletedAt,
			&l.ProductID, &l.SupplierID, &l.SafetyStock, &pc, &pn, &sc)
		if err != nil {
			return nil, err
		}
		if pc.Valid {
			l.ProductCode = pc
		}
		if pn.Valid {
			l.ProductName = pn
		}
		if sc.Valid {
			l.SupplierCode = sc
		}
		lots = append(lots, l)
	}
	return lots, rows.Err()
}

func (r *InventoryRepository) ListExpiringLots(ctx context.Context, days int32) ([]ListExpiringLotsRow, error) {
	rows, err := r.db.Pool.Query(ctx, `
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
		ORDER BY l.expires_at ASC
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lots []ListExpiringLotsRow
	for rows.Next() {
		var l ListExpiringLotsRow
		var pc, pn, sc pgtype.Text
		var exp pgtype.Timestamptz
		err := rows.Scan(&l.ID, &l.Code, &l.StockLevelID, &l.LotNumber, &l.InitialQuantity,
			&l.AvailableQuantity, &l.ReservedQuantity, &l.Status, &l.IsQuarantined,
			&l.ProductionDate, &exp, &l.CreatedAt, &l.UpdatedAt, &l.DeletedAt,
			&l.ProductID, &l.SupplierID, &pc, &pn, &sc)
		if err != nil {
			return nil, err
		}
		if pc.Valid {
			l.ProductCode = pc
		}
		if pn.Valid {
			l.ProductName = pn
		}
		if sc.Valid {
			l.SupplierCode = sc
		}
		if exp.Valid {
			l.ExpiresAt = pgtype.Timestamptz{Time: exp.Time, Valid: true}
		}
		lots = append(lots, l)
	}
	return lots, rows.Err()
}

func (r *InventoryRepository) AdjustLotQuantities(ctx context.Context, id int64, quantityDelta int32) (inventory.InventoryLot, error) {
	var lot inventory.InventoryLot
	err := r.db.Pool.QueryRow(ctx, `
		UPDATE inventory.lot
		SET available_quantity = available_quantity + $2,
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, stock_level_id, lot_number, initial_quantity, available_quantity, reserved_quantity, status, is_quarantined, production_date, expires_at, created_at, updated_at, deleted_at
	`, id, quantityDelta).Scan(
		&lot.ID, &lot.Code, &lot.StockLevelID, &lot.LotNumber, &lot.InitialQuantity,
		&lot.AvailableQuantity, &lot.ReservedQuantity, &lot.Status, &lot.IsQuarantined,
		&lot.ProductionDate, &lot.ExpiresAt, &lot.CreatedAt, &lot.UpdatedAt, &lot.DeletedAt,
	)
	return lot, err
}

func (r *InventoryRepository) CreateStockAdjustment(ctx context.Context, code string, lotID int64, quantityDelta, previousQuantity, newQuantity int32, reasonCode, reason string, adjustedBy pgtype.Int8) (CreateStockAdjustmentRow, error) {
	var adj CreateStockAdjustmentRow
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO inventory.stock_adjustment (
			code, lot_id, quantity_delta, previous_quantity, new_quantity,
			reason_code, reason, adjusted_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, code, lot_id, quantity_delta, previous_quantity, new_quantity, reason_code, reason, adjusted_by, created_at
	`, code, lotID, quantityDelta, previousQuantity, newQuantity, reasonCode, reason, adjustedBy).Scan(
		&adj.ID, &adj.Code, &adj.LotID, &adj.QuantityDelta, &adj.PreviousQuantity,
		&adj.NewQuantity, &adj.ReasonCode, &adj.Reason, &adj.AdjustedBy, &adj.CreatedAt,
	)
	return adj, err
}

func (r *InventoryRepository) UpdateStockLevelQuantitiesOnAdjustment(ctx context.Context, id int64, quantityDelta int32) (inventory.InventoryStockLevel, error) {
	var sl inventory.InventoryStockLevel
	err := r.db.Pool.QueryRow(ctx, `
		UPDATE inventory.stock_level
		SET available_quantity = available_quantity + $2,
		    total_quantity = total_quantity + $2,
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, product_id, supplier_id, available_quantity, reserved_quantity, total_quantity, safety_stock, created_at, updated_at, deleted_at
	`, id, quantityDelta).Scan(
		&sl.ID, &sl.Code, &sl.ProductID, &sl.SupplierID, &sl.AvailableQuantity,
		&sl.ReservedQuantity, &sl.TotalQuantity, &sl.SafetyStock, &sl.CreatedAt, &sl.UpdatedAt, &sl.DeletedAt,
	)
	return sl, err
}

func (r *InventoryRepository) GetQuarantineRecordByLot(ctx context.Context, lotID int64) (GetQuarantineRecordByLotRow, error) {
	var qr GetQuarantineRecordByLotRow
	var qrStatus, qrReasonText, qrCode pgtype.Text
	var qrAdjustedQty pgtype.Int4
	var qrCreatedAt, qrUpdatedAt pgtype.Timestamptz
	var qrAdjustedBy pgtype.Int8
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, code, lot_id, status, reason, adjusted_quantity, adjusted_by, created_at, updated_at
		FROM inventory.quarantine_record
		WHERE lot_id = $1 AND status = 'quarantined' AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`, lotID).Scan(&qr.ID, &qrCode, &qr.LotID, &qrStatus, &qrReasonText, &qrAdjustedQty, &qrAdjustedBy, &qrCreatedAt, &qrUpdatedAt)
	if qrCode.Valid {
		qr.Code = qrCode.String
	}
	if qrReasonText.Valid {
		qr.Reason = qrReasonText.String
	}
	if qrAdjustedQty.Valid {
		qr.AdjustedQuantity = int32(qrAdjustedQty.Int32)
	}
	if qrAdjustedBy.Valid {
		qr.AdjustedBy = qrAdjustedBy.Int64
	}
	if qrCreatedAt.Valid {
		qr.CreatedAt = qrCreatedAt.Time
	}
	if qrUpdatedAt.Valid {
		qr.UpdatedAt = qrUpdatedAt.Time
	}
	return qr, err
}

func (r *InventoryRepository) UpdateQuarantineRecordStatus(ctx context.Context, id int64, adjustedQuantity int32) (inventory.InventoryQuarantineRecord, error) {
	var qr inventory.InventoryQuarantineRecord
	var qrStatus, qrReasonText, qrCode pgtype.Text
	var qrAdjustedQty pgtype.Int4
	var qrCreatedAt, qrUpdatedAt pgtype.Timestamptz
	var qrAdjustedBy pgtype.Int8
	err := r.db.Pool.QueryRow(ctx, `
		UPDATE inventory.quarantine_record
		SET status = 'released',
		    adjusted_quantity = $2,
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, lot_id, status, reason, adjusted_quantity, adjusted_by, created_at, updated_at, deleted_at
	`, id, adjustedQuantity).Scan(&qr.ID, &qrCode, &qr.LotID, &qrStatus, &qrReasonText, &qrAdjustedQty, &qrAdjustedBy, &qrCreatedAt, &qrUpdatedAt, &qr.DeletedAt)
	if qrCode.Valid {
		qr.Code = qrCode.String
	}
	if qrStatus.Valid {
		qr.Status = inventory.InventoryQuarantineStatus(qrStatus.String)
	}
	if qrReasonText.Valid {
		qr.Reason = qrReasonText.String
	}
	if qrAdjustedQty.Valid {
		qr.AdjustedQuantity = int32(qrAdjustedQty.Int32)
	}
	if qrAdjustedBy.Valid {
		qr.AdjustedBy = pgtype.Int8{Int64: qrAdjustedBy.Int64, Valid: qrAdjustedBy.Valid}
	}
	if qrCreatedAt.Valid {
		qr.CreatedAt = qrCreatedAt.Time
	}
	if qrUpdatedAt.Valid {
		qr.UpdatedAt = qrUpdatedAt.Time
	}
	return qr, err
}
