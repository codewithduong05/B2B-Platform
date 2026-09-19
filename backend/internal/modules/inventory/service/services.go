package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/database/queries/inventory"
	repo "github.com/atlas-platform/backend/internal/modules/inventory/repository"
	"github.com/atlas-platform/backend/internal/modules/inventory/schema"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrStockLevelNotFound  = errors.New("stock level not found")
	ErrLotNotFound         = errors.New("lot not found")
	ErrReservationNotFound = errors.New("reservation not found")
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrLotQuarantined      = errors.New("lot is quarantined")
	ErrInvalidQuantity     = errors.New("invalid quantity")
	ErrDuplicateRequest    = errors.New("duplicate request key")
)

type EventPublisher interface {
	Publish(ctx context.Context, routingKey string, payload interface{}) error
}

type ReservationResult struct {
	Reservations []schema.ReservationSummary
	Replayed     bool
}

type InventoryService struct {
	db        *database.DB
	repo      *repo.InventoryRepository
	publisher EventPublisher
}

func NewInventoryService(db *database.DB, publisher EventPublisher) *InventoryService {
	return &InventoryService{
		db:        db,
		repo:      repo.NewInventoryRepository(db),
		publisher: publisher,
	}
}

func (s *InventoryService) ReserveStock(ctx context.Context, req schema.ReserveStockRequest) ([]schema.ReservationSummary, error) {
	if req.Quantity <= 0 {
		return nil, ErrInvalidQuantity
	}

	if req.RequestID != "" {
		existing, err := s.repo.GetReservationsByRequestID(ctx, req.RequestID)
		if err == nil && len(existing) > 0 {
			var summaries []schema.ReservationSummary
			for _, r := range existing {
				summaries = append(summaries, toReservationSummary(r))
			}
			return summaries, nil
		}
	}

	var reservations []schema.ReservationSummary

	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repo.NewInventoryRepositoryWithTx(tx)

		if req.RequestID != "" {
			existing, err := txRepo.GetReservationsByRequestID(ctx, req.RequestID)
			if err == nil && len(existing) > 0 {
				var summaries []schema.ReservationSummary
				for _, r := range existing {
					summaries = append(summaries, toReservationSummary(r))
				}
				reservations = summaries
				return nil
			}
		}

		sl, err := txRepo.GetStockLevelByProductAndSupplier(ctx, req.ProductID, req.SupplierID)
		if err != nil {
			return ErrStockLevelNotFound
		}

		lots, err := txRepo.GetFEFOLotsForUpdate(ctx, sl.ID)
		if err != nil {
			return fmt.Errorf("get fefo lots: %w", err)
		}

		var totalAvailable int32
		for _, lot := range lots {
			totalAvailable += lot.AvailableQuantity
		}

		if totalAvailable < req.Quantity {
			if s.publisher != nil {
				_ = s.publisher.Publish(ctx, "inventory.stock.shortfall", map[string]interface{}{
					"product_id":  req.ProductID,
					"supplier_id": req.SupplierID,
					"requested":   req.Quantity,
					"available":   totalAvailable,
					"request_id":  req.RequestID,
				})
			}
			return ErrInsufficientStock
		}

		remainingToAllocate := req.Quantity
		expiryTime := time.Now().Add(30 * time.Minute)
		if req.ExpiresIn > 0 {
			expiryTime = time.Now().Add(time.Duration(req.ExpiresIn) * time.Minute)
		}

		var allocatedLots []struct {
			lot    inventory.InventoryLot
			toTake int32
		}

		for _, lot := range lots {
			if remainingToAllocate <= 0 {
				break
			}
			take := lot.AvailableQuantity
			if take > remainingToAllocate {
				take = remainingToAllocate
			}
			allocatedLots = append(allocatedLots, struct {
				lot    inventory.InventoryLot
				toTake int32
			}{lot: lot, toTake: take})
			remainingToAllocate -= take
		}

		if remainingToAllocate > 0 {
			return ErrInsufficientStock
		}

		var totalReservedDelta int32
		for _, alloc := range allocatedLots {
			newAvail := alloc.lot.AvailableQuantity - alloc.toTake
			newRes := alloc.lot.ReservedQuantity + alloc.toTake
			newStatus := alloc.lot.Status

			_, err = txRepo.UpdateLotQuantities(ctx, alloc.lot.ID, newAvail, newRes, newStatus)
			if err != nil {
				return fmt.Errorf("update lot quantities: %w", err)
			}

			resCode := fmt.Sprintf("res_%d_%d", time.Now().UnixNano(), alloc.lot.ID)

			var orderLineID pgtype.Int8
			if req.OrderLineID != nil {
				orderLineID = pgtype.Int8{Int64: *req.OrderLineID, Valid: true}
			}

			res, err := txRepo.CreateReservation(ctx, inventory.CreateReservationParams{
				Code:        resCode,
				LotID:       alloc.lot.ID,
				OrderLineID: orderLineID,
				RequestID:   req.RequestID,
				Quantity:    alloc.toTake,
				Status:      inventory.InventoryReservationStatusReserved,
				ExpiresAt:   expiryTime,
			})
			if err != nil {
				if database.IsUniqueViolation(err) {
					return ErrDuplicateRequest
				}
				return fmt.Errorf("create reservation: %w", err)
			}

			totalReservedDelta += alloc.toTake
			reservations = append(reservations, toReservationSummary(res))
		}

		// Re-read the stock level under a row lock: the snapshot above
		// predates the FEFO lot locks, so concurrent reserves would
		// otherwise lost-update these counters (overstated availability).
		lockedSL, err := txRepo.GetStockLevelByIDForUpdate(ctx, sl.ID)
		if err != nil {
			return fmt.Errorf("lock stock level: %w", err)
		}
		newStockAvail := lockedSL.AvailableQuantity - req.Quantity
		newStockRes := lockedSL.ReservedQuantity + req.Quantity
		_, err = txRepo.UpdateStockLevelQuantities(ctx, sl.ID, newStockAvail, newStockRes, sl.TotalQuantity)
		if err != nil {
			return fmt.Errorf("update stock level: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if s.publisher != nil {
		for _, res := range reservations {
			_ = s.publisher.Publish(ctx, "inventory.stock.reserved", res)
		}
	}

	return reservations, nil
}

func (s *InventoryService) ReserveStockResult(ctx context.Context, req schema.ReserveStockRequest) (*ReservationResult, error) {
	replayed := false
	if req.RequestID != "" {
		existing, err := s.repo.GetReservationsByRequestID(ctx, req.RequestID)
		if err == nil && len(existing) > 0 {
			replayed = true
		}
	}

	reservations, err := s.ReserveStock(ctx, req)
	if err != nil {
		return nil, err
	}

	return &ReservationResult{Reservations: reservations, Replayed: replayed}, nil
}

func (s *InventoryService) GetAvailability(ctx context.Context, productIDs []int64, supplierID *int64) ([]schema.StockLevelSummary, error) {
	unique := make([]int64, 0, len(productIDs))
	seen := make(map[int64]struct{}, len(productIDs))
	for _, id := range productIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}

	var summaries []schema.StockLevelSummary
	for _, productID := range unique {
		if supplierID != nil {
			sl, err := s.repo.GetStockLevelByProductAndSupplier(ctx, productID, *supplierID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}
				return nil, fmt.Errorf("get stock level: %w", err)
			}
			summaries = append(summaries, toStockLevelSummary(sl))
			continue
		}

		levels, err := s.repo.ListStockLevels(ctx, productID)
		if err != nil {
			return nil, fmt.Errorf("list stock levels: %w", err)
		}
		for _, sl := range levels {
			summaries = append(summaries, toStockLevelSummary(sl))
		}
	}

	return summaries, nil
}

func (s *InventoryService) GetReservation(ctx context.Context, id int64) (schema.ReservationSummary, error) {
	res, err := s.repo.GetReservationByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return schema.ReservationSummary{}, ErrReservationNotFound
		}
		return schema.ReservationSummary{}, fmt.Errorf("get reservation: %w", err)
	}

	return toReservationSummary(res), nil
}

func (s *InventoryService) GetReservationsByRequestID(ctx context.Context, requestID string) ([]schema.ReservationSummary, error) {
	rows, err := s.repo.GetReservationsByRequestID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("get reservations by request id: %w", err)
	}

	if len(rows) == 0 {
		return nil, ErrReservationNotFound
	}

	var summaries []schema.ReservationSummary
	for _, r := range rows {
		summaries = append(summaries, toReservationSummary(r))
	}

	return summaries, nil
}

func (s *InventoryService) ReleaseExpiredReservations(ctx context.Context) (int, error) {
	expired, err := s.repo.FindExpiredReservations(ctx)
	if err != nil {
		return 0, fmt.Errorf("find expired reservations: %w", err)
	}

	releasedCount := 0

	for _, res := range expired {
		err := s.db.WithTx(ctx, func(tx *database.Tx) error {
			txRepo := repo.NewInventoryRepositoryWithTx(tx)

			updatedRes, err := txRepo.UpdateReservationStatus(ctx, res.ID, inventory.InventoryReservationStatusExpired)
			if err != nil {
				return err
			}

			lot, err := txRepo.GetLotByID(ctx, updatedRes.LotID)
			if err != nil {
				return err
			}

			newAvail := lot.AvailableQuantity + updatedRes.Quantity
			newRes := lot.ReservedQuantity - updatedRes.Quantity
			if newRes < 0 {
				newRes = 0
			}

			_, err = txRepo.UpdateLotQuantities(ctx, lot.ID, newAvail, newRes, lot.Status)
			if err != nil {
				return err
			}

			sl, err := txRepo.GetStockLevelByID(ctx, lot.StockLevelID)
			if err == nil && sl.ID != 0 {
				newSlAvail := sl.AvailableQuantity + updatedRes.Quantity
				newSlRes := sl.ReservedQuantity - updatedRes.Quantity
				if newSlRes < 0 {
					newSlRes = 0
				}
				_, _ = txRepo.UpdateStockLevelQuantities(ctx, sl.ID, newSlAvail, newSlRes, sl.TotalQuantity)
			}

			return nil
		})

		if err == nil {
			releasedCount++
			if s.publisher != nil {
				_ = s.publisher.Publish(ctx, "inventory.stock.updated", map[string]interface{}{
					"reservation_id": res.ID,
					"lot_id":         res.LotID,
					"quantity":       res.Quantity,
					"status":         "expired",
				})
			}
		}
	}

	return releasedCount, nil
}

func (s *InventoryService) QuarantineLot(ctx context.Context, req schema.QuarantineLotRequest, userID int64) (*schema.LotSummary, error) {
	var updatedLot inventory.InventoryLot

	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repo.NewInventoryRepositoryWithTx(tx)

		lot, err := txRepo.GetLotByID(ctx, req.LotID)
		if err != nil {
			return ErrLotNotFound
		}

		if lot.IsQuarantined {
			updatedLot = lot
			return nil
		}

		quarantinedQty := lot.AvailableQuantity

		updatedLot, err = txRepo.QuarantineLot(ctx, lot.ID)
		if err != nil {
			return fmt.Errorf("quarantine lot: %w", err)
		}

		qCode := fmt.Sprintf("q_%d_%d", time.Now().UnixNano(), lot.ID)
		var userRef pgtype.Int8
		if userID > 0 {
			userRef = pgtype.Int8{Int64: userID, Valid: true}
		}

		_, err = txRepo.CreateQuarantineRecord(ctx, inventory.CreateQuarantineRecordParams{
			Code:             qCode,
			LotID:            lot.ID,
			Reason:           req.Reason,
			Status:           inventory.InventoryQuarantineStatusQuarantined,
			AdjustedQuantity: quarantinedQty,
			AdjustedBy:       userRef,
		})
		if err != nil {
			return fmt.Errorf("create quarantine record: %w", err)
		}

		sl, err := txRepo.GetStockLevelByID(ctx, lot.StockLevelID)
		if err == nil && sl.ID != 0 {
			newSlAvail := sl.AvailableQuantity - quarantinedQty
			if newSlAvail < 0 {
				newSlAvail = 0
			}
			newSlTotal := sl.TotalQuantity - quarantinedQty
			if newSlTotal < 0 {
				newSlTotal = 0
			}
			_, _ = txRepo.UpdateStockLevelQuantities(ctx, sl.ID, newSlAvail, sl.ReservedQuantity, newSlTotal)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, "inventory.lot.quarantined", map[string]interface{}{
			"lot_id":      req.LotID,
			"reason":      req.Reason,
			"adjusted_by": userID,
		})
	}

	sum := toLotSummary(updatedLot)
	return &sum, nil
}

func toReservationSummary(r inventory.InventoryReservation) schema.ReservationSummary {
	var orderLineID *int64
	if r.OrderLineID.Valid {
		v := r.OrderLineID.Int64
		orderLineID = &v
	}

	return schema.ReservationSummary{
		ID:          r.ID,
		Code:        r.Code,
		LotID:       r.LotID,
		OrderLineID: orderLineID,
		RequestID:   r.RequestID,
		Quantity:    r.Quantity,
		Status:      string(r.Status),
		ExpiresAt:   r.ExpiresAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func toStockLevelSummary(sl inventory.InventoryStockLevel) schema.StockLevelSummary {
	return schema.StockLevelSummary{
		ID:                sl.ID,
		Code:              sl.Code,
		ProductID:         sl.ProductID,
		SupplierID:        sl.SupplierID,
		AvailableQuantity: sl.AvailableQuantity,
		ReservedQuantity:  sl.ReservedQuantity,
		TotalQuantity:     sl.TotalQuantity,
		SafetyStock:       sl.SafetyStock,
		CreatedAt:         sl.CreatedAt,
		UpdatedAt:         sl.UpdatedAt,
	}
}

func toLotSummary(l inventory.InventoryLot) schema.LotSummary {
	var prodDate *time.Time
	if l.ProductionDate.Valid {
		prodDate = &l.ProductionDate.Time
	}

	var expDate *time.Time
	if l.ExpiresAt.Valid {
		expDate = &l.ExpiresAt.Time
	}

	return schema.LotSummary{
		ID:                l.ID,
		Code:              l.Code,
		StockLevelID:      l.StockLevelID,
		LotNumber:         l.LotNumber,
		InitialQuantity:   l.InitialQuantity,
		AvailableQuantity: l.AvailableQuantity,
		ReservedQuantity:  l.ReservedQuantity,
		Status:            string(l.Status),
		IsQuarantined:     l.IsQuarantined,
		ProductionDate:    prodDate,
		ExpiresAt:         expDate,
		CreatedAt:         l.CreatedAt,
		UpdatedAt:         l.UpdatedAt,
	}
}

// ReleaseReservationAttempt rolls back reservations created under the given
// request IDs (e.g. a failed checkout attempt that reserved some lines
// before hitting insufficient stock). Only rows still in 'reserved' status
// are released and restored to their lot and stock level; any other status
// is skipped, making the call idempotent. It must only be used for
// same-attempt rollback, never against fulfilled orders.
func (s *InventoryService) ReleaseReservationAttempt(ctx context.Context, requestIDs []string) error {
	if len(requestIDs) == 0 {
		return nil
	}
	return s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repo.NewInventoryRepositoryWithTx(tx)
		for _, requestID := range requestIDs {
			if requestID == "" {
				continue
			}
			// Row-locked reads: concurrent releases/reserves of the same
			// rows must serialize, otherwise restores lost-update the
			// counters (and can drive them negative).
			rows, err := txRepo.GetReservationsByRequestIDForUpdate(ctx, requestID)
			if err != nil {
				return fmt.Errorf("get reservations for release: %w", err)
			}
			for _, res := range rows {
				if res.Status != inventory.InventoryReservationStatusReserved {
					continue
				}
				lot, err := txRepo.GetLotByIDForUpdate(ctx, res.LotID)
				if err != nil {
					return fmt.Errorf("get lot for release: %w", err)
				}
				if _, err := txRepo.UpdateLotQuantities(ctx, lot.ID,
					lot.AvailableQuantity+res.Quantity,
					lot.ReservedQuantity-res.Quantity,
					lot.Status); err != nil {
					return fmt.Errorf("restore lot quantities: %w", err)
				}
				sl, err := txRepo.GetStockLevelByIDForUpdate(ctx, lot.StockLevelID)
				if err != nil {
					return fmt.Errorf("get stock level for release: %w", err)
				}
				if _, err := txRepo.UpdateStockLevelQuantities(ctx, sl.ID,
					sl.AvailableQuantity+res.Quantity,
					sl.ReservedQuantity-res.Quantity,
					sl.TotalQuantity); err != nil {
					return fmt.Errorf("restore stock level quantities: %w", err)
				}
				if _, err := txRepo.UpdateReservationStatus(ctx, res.ID,
					inventory.InventoryReservationStatusReleased); err != nil {
					return fmt.Errorf("release reservation: %w", err)
				}
			}
		}
		return nil
	})
}
