package repository

import (
	"context"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/database/queries/inventory"
)

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
