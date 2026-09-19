package service_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/inventory/schema"
	"github.com/atlas-platform/backend/internal/modules/inventory/service"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// inventoryTestDBLockKey serializes schema recreation across test binaries that
// share the same PostgreSQL database (service and router packages both drop the
// inventory/catalog/identity/pricing schemas and re-run migrations in TestMain).
const inventoryTestDBLockKey = 1946819412

func TestMain(m *testing.M) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err == nil {
		rawDB, err := sql.Open("pgx", cfg.PostgresDSN())
		if err == nil && rawDB != nil {
			_, _ = rawDB.Exec("SELECT pg_advisory_lock($1)", int64(inventoryTestDBLockKey))
			_, _ = rawDB.Exec("DROP SCHEMA IF EXISTS catalog CASCADE; DROP SCHEMA IF EXISTS inventory CASCADE; DROP SCHEMA IF EXISTS identity CASCADE; DROP SCHEMA IF EXISTS pricing CASCADE; DROP SCHEMA IF EXISTS commerce CASCADE; DROP SCHEMA IF EXISTS payments CASCADE; DROP SCHEMA IF EXISTS promotions CASCADE; DROP SCHEMA IF EXISTS crm CASCADE; DROP SCHEMA IF EXISTS reports CASCADE; DROP SCHEMA IF EXISTS erp CASCADE; DROP TABLE IF EXISTS schema_migrations CASCADE; DROP TYPE IF EXISTS catalog_handling_class_type CASCADE;")

			if err := database.RunMigrations(ctx, &cfg.Postgres, "file://../../../../migrations"); err != nil {
				fmt.Printf("TestMain migration error: %v\n", err)
			}

			code := m.Run()
			_, _ = rawDB.Exec("SELECT pg_advisory_unlock($1)", int64(inventoryTestDBLockKey))
			rawDB.Close()
			os.Exit(code)
		}
	}
	os.Exit(m.Run())
}

type recordingPublisher struct {
	mu     sync.Mutex
	events []eventRecord
}

type eventRecord struct {
	RoutingKey string
	Payload    interface{}
}

func (p *recordingPublisher) Publish(ctx context.Context, routingKey string, payload interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, eventRecord{RoutingKey: routingKey, Payload: payload})
	return nil
}

func (p *recordingPublisher) getEvents(routingKey string) []eventRecord {
	p.mu.Lock()
	defer p.mu.Unlock()
	var matched []eventRecord
	for _, e := range p.events {
		if e.RoutingKey == routingKey {
			matched = append(matched, e)
		}
	}
	return matched
}

func setupTestDB(t *testing.T) (*database.DB, *service.InventoryService, *recordingPublisher) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Skipping integration test: configuration not available: %v", err)
	}

	db, err := database.New(ctx, &cfg.Postgres)
	if err != nil {
		t.Skipf("Skipping integration test: database connection failed: %v", err)
	}

	publisher := &recordingPublisher{}
	svc := service.NewInventoryService(db, publisher)

	return db, svc, publisher
}

var testCounter int64

func uniqueSuffix() string {
	c := atomic.AddInt64(&testCounter, 1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano()%1000000, c)
}

func createTestProductAndSupplier(t *testing.T, db *database.DB) (int64, int64) {
	ctx := context.Background()
	suffix := uniqueSuffix()

	var categoryID, unitID, supplierID, productID int64

	err := db.WithTx(ctx, func(tx *database.Tx) error {
		// Category
		err := tx.QueryRow(ctx, `
			INSERT INTO catalog.category (code, name, slug)
			VALUES ($1, $2, $3)
			RETURNING id
		`, "cat_"+suffix, "Test Category", "test-cat-"+suffix).Scan(&categoryID)
		if err != nil {
			return err
		}

		// Unit
		err = tx.QueryRow(ctx, `
			INSERT INTO catalog.unit (code, name, symbol, unit_type)
			VALUES ($1, $2, $3, 'base')
			RETURNING id
		`, "unit_"+suffix, "Piece", "pcs").Scan(&unitID)
		if err != nil {
			return err
		}

		// Supplier
		err = tx.QueryRow(ctx, `
			INSERT INTO catalog.supplier (code, supplier_id, name, slug)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, "sup_"+suffix, 1001, "Test Supplier", "test-sup-"+suffix).Scan(&supplierID)
		if err != nil {
			return err
		}

		// Product
		err = tx.QueryRow(ctx, `
			INSERT INTO catalog.product (code, slug, name, category_id, handling_class, base_unit_id, status, is_active)
			VALUES ($1, $2, $3, $4, 'ambient', $5, 'published', true)
			RETURNING id
		`, "prod_"+suffix, "test-prod-"+suffix, "Test Product", categoryID, unitID).Scan(&productID)
		return err
	})

	if err != nil {
		t.Fatalf("failed to setup test product and supplier: %v", err)
	}

	return productID, supplierID
}

func createTestStockAndLot(t *testing.T, db *database.DB, productID, supplierID int64, lotNumber string, initialQty int32, expiresAt time.Time) (int64, int64) {
	ctx := context.Background()
	suffix := uniqueSuffix()

	var stockLevelID, lotID int64

	err := db.WithTx(ctx, func(tx *database.Tx) error {
		// Stock level (upsert or insert)
		err := tx.QueryRow(ctx, `
			INSERT INTO inventory.stock_level (code, product_id, supplier_id, available_quantity, total_quantity)
			VALUES ($1, $2, $3, $4, $4)
			ON CONFLICT (product_id, supplier_id) DO UPDATE
			SET available_quantity = inventory.stock_level.available_quantity + $4,
			    total_quantity = inventory.stock_level.total_quantity + $4
			RETURNING id
		`, "sl_"+suffix, productID, supplierID, initialQty).Scan(&stockLevelID)
		if err != nil {
			// If stock level already existed, fetch its ID
			err = tx.QueryRow(ctx, `
				SELECT id FROM inventory.stock_level WHERE product_id = $1 AND supplier_id = $2
			`, productID, supplierID).Scan(&stockLevelID)
			if err != nil {
				return err
			}
			// Update quantities
			_, err = tx.Exec(ctx, `
				UPDATE inventory.stock_level
				SET available_quantity = available_quantity + $1, total_quantity = total_quantity + $1
				WHERE id = $2
			`, initialQty, stockLevelID)
			if err != nil {
				return err
			}
		}

		// Lot
		err = tx.QueryRow(ctx, `
			INSERT INTO inventory.lot (code, stock_level_id, lot_number, initial_quantity, available_quantity, status, expires_at)
			VALUES ($1, $2, $3, $4, $4, 'active', $5)
			RETURNING id
		`, "lot_"+suffix, stockLevelID, lotNumber, initialQty, expiresAt).Scan(&lotID)
		return err
	})

	if err != nil {
		t.Fatalf("failed to create stock and lot: %v", err)
	}

	return stockLevelID, lotID
}

func TestIntegration_BasicReservation(t *testing.T) {
	db, svc, publisher := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	productID, supplierID := createTestProductAndSupplier(t, db)
	createTestStockAndLot(t, db, productID, supplierID, "LOT-BASIC-1", 10, time.Now().Add(24*time.Hour))

	req := schema.ReserveStockRequest{
		ProductID:  productID,
		SupplierID: supplierID,
		Quantity:   7,
		RequestID:  fmt.Sprintf("req-basic-%s", uniqueSuffix()),
	}

	res, err := svc.ReserveStock(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error reserving stock: %v", err)
	}

	if len(res) == 0 {
		t.Fatalf("expected reservations, got 0")
	}

	if res[0].Quantity != 7 {
		t.Errorf("expected reserved quantity 7, got %d", res[0].Quantity)
	}

	// Verify database state
	var availQty, resQty int32
	err = db.Pool.QueryRow(context.Background(), `
		SELECT available_quantity, reserved_quantity FROM inventory.stock_level WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID).Scan(&availQty, &resQty)
	if err != nil {
		t.Fatalf("failed to query stock level: %v", err)
	}

	if availQty != 3 {
		t.Errorf("expected available stock 3, got %d", availQty)
	}
	if resQty != 7 {
		t.Errorf("expected reserved stock 7, got %d", resQty)
	}

	// Check events
	events := publisher.getEvents("inventory.stock.reserved")
	if len(events) == 0 {
		t.Errorf("expected inventory.stock.reserved event to be published")
	}
}

func TestIntegration_FEFOAllocation(t *testing.T) {
	db, svc, _ := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	productID, supplierID := createTestProductAndSupplier(t, db)
	now := time.Now()

	_, lotA := createTestStockAndLot(t, db, productID, supplierID, "LOT-FEFO-A", 5, now.Add(1*time.Hour))
	_, lotB := createTestStockAndLot(t, db, productID, supplierID, "LOT-FEFO-B", 10, now.Add(2*time.Hour))
	_, lotC := createTestStockAndLot(t, db, productID, supplierID, "LOT-FEFO-C", 20, now.Add(3*time.Hour))

	req := schema.ReserveStockRequest{
		ProductID:  productID,
		SupplierID: supplierID,
		Quantity:   12,
		RequestID:  fmt.Sprintf("req-fefo-%s", uniqueSuffix()),
	}

	reservations, err := svc.ReserveStock(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should allocate 5 from lotA and 7 from lotB
	allocMap := make(map[int64]int32)
	for _, r := range reservations {
		allocMap[r.LotID] += r.Quantity
	}

	if allocMap[lotA] != 5 {
		t.Errorf("expected lot A to have 5 allocated, got %d", allocMap[lotA])
	}
	if allocMap[lotB] != 7 {
		t.Errorf("expected lot B to have 7 allocated, got %d", allocMap[lotB])
	}
	if allocMap[lotC] != 0 {
		t.Errorf("expected lot C to have 0 allocated, got %d", allocMap[lotC])
	}
}

func TestIntegration_EqualExpiration(t *testing.T) {
	db, svc, _ := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	productID, supplierID := createTestProductAndSupplier(t, db)
	sameExpiry := time.Now().Add(24 * time.Hour)

	_, lot1 := createTestStockAndLot(t, db, productID, supplierID, "LOT-EQ-1", 10, sameExpiry)
	_, _ = createTestStockAndLot(t, db, productID, supplierID, "LOT-EQ-2", 10, sameExpiry)

	req := schema.ReserveStockRequest{
		ProductID:  productID,
		SupplierID: supplierID,
		Quantity:   5,
		RequestID:  fmt.Sprintf("req-eq-%s", uniqueSuffix()),
	}

	reservations, err := svc.ReserveStock(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reservations) != 1 || reservations[0].LotID != lot1 {
		t.Errorf("expected allocation from lot1 (due to lower ID), got lot ID %d", reservations[0].LotID)
	}
}

func TestIntegration_InsufficientStockRollback(t *testing.T) {
	db, svc, publisher := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	productID, supplierID := createTestProductAndSupplier(t, db)
	createTestStockAndLot(t, db, productID, supplierID, "LOT-INS-1", 5, time.Now().Add(24*time.Hour))

	req := schema.ReserveStockRequest{
		ProductID:  productID,
		SupplierID: supplierID,
		Quantity:   20,
		RequestID:  fmt.Sprintf("req-ins-%s", uniqueSuffix()),
	}

	_, err := svc.ReserveStock(context.Background(), req)
	if !errorsIs(err, service.ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}

	// Verify stock levels unchanged
	var availQty int32
	err = db.Pool.QueryRow(context.Background(), `
		SELECT available_quantity FROM inventory.stock_level WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID).Scan(&availQty)
	if err != nil {
		t.Fatalf("failed to query stock: %v", err)
	}
	if availQty != 5 {
		t.Errorf("expected available stock to remain 5, got %d", availQty)
	}

	// Verify shortfall event published
	shortfalls := publisher.getEvents("inventory.stock.shortfall")
	if len(shortfalls) == 0 {
		t.Errorf("expected inventory.stock.shortfall event")
	}
}

func TestIntegration_SequentialIdempotency(t *testing.T) {
	db, svc, _ := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	productID, supplierID := createTestProductAndSupplier(t, db)
	createTestStockAndLot(t, db, productID, supplierID, "LOT-IDEM-1", 10, time.Now().Add(24*time.Hour))

	reqID := fmt.Sprintf("req-idem-%s", uniqueSuffix())
	req := schema.ReserveStockRequest{
		ProductID:  productID,
		SupplierID: supplierID,
		Quantity:   4,
		RequestID:  reqID,
	}

	res1, err := svc.ReserveStock(context.Background(), req)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	res2, err := svc.ReserveStock(context.Background(), req)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}

	if len(res1) == 0 || len(res2) == 0 || res1[0].ID != res2[0].ID {
		t.Errorf("expected identical reservation ID for idempotent requests, got %v vs %v", res1, res2)
	}

	// Verify stock deducted only once (available should be 6)
	var availQty int32
	_ = db.Pool.QueryRow(context.Background(), `
		SELECT available_quantity FROM inventory.stock_level WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID).Scan(&availQty)
	if availQty != 6 {
		t.Errorf("expected available stock 6 after duplicate request, got %d", availQty)
	}
}

func TestIntegration_ConcurrentIdempotency(t *testing.T) {
	db, svc, _ := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	productID, supplierID := createTestProductAndSupplier(t, db)
	createTestStockAndLot(t, db, productID, supplierID, "LOT-CONC-IDEM", 20, time.Now().Add(24*time.Hour))

	reqID := fmt.Sprintf("req-conc-idem-%s", uniqueSuffix())
	req := schema.ReserveStockRequest{
		ProductID:  productID,
		SupplierID: supplierID,
		Quantity:   5,
		RequestID:  reqID,
	}

	var wg sync.WaitGroup
	var successes int32
	var failures int32
	concurrentRequests := 10

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.ReserveStock(context.Background(), req)
			if err == nil {
				atomic.AddInt32(&successes, 1)
			} else {
				atomic.AddInt32(&failures, 1)
			}
		}()
	}

	wg.Wait()

	if successes == 0 {
		t.Fatalf("expected at least one success")
	}

	// Verify stock deducted exactly once (5 units)
	var availQty, resQty int32
	_ = db.Pool.QueryRow(context.Background(), `
		SELECT available_quantity, reserved_quantity FROM inventory.stock_level WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID).Scan(&availQty, &resQty)

	if availQty != 15 {
		t.Errorf("expected available stock 15, got %d", availQty)
	}
	if resQty != 5 {
		t.Errorf("expected reserved stock 5, got %d", resQty)
	}
}

func TestIntegration_OversellingConcurrency(t *testing.T) {
	db, svc, _ := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	productID, supplierID := createTestProductAndSupplier(t, db)
	createTestStockAndLot(t, db, productID, supplierID, "LOT-OVERS", 10, time.Now().Add(24*time.Hour))

	var wg sync.WaitGroup
	var successCount int32
	var shortfallCount int32

	// Two requests trying to reserve 7 units each (total 14 > 10 available)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := schema.ReserveStockRequest{
				ProductID:  productID,
				SupplierID: supplierID,
				Quantity:   7,
				RequestID:  fmt.Sprintf("req-overs-%s-%d", uniqueSuffix(), idx),
			}
			_, err := svc.ReserveStock(context.Background(), req)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else if errorsIs(err, service.ErrInsufficientStock) {
				atomic.AddInt32(&shortfallCount, 1)
			}
		}(i)
	}

	wg.Wait()

	if successCount != 1 {
		t.Errorf("expected exactly 1 successful reservation, got %d", successCount)
	}
	if shortfallCount != 1 {
		t.Errorf("expected exactly 1 insufficient stock failure, got %d", shortfallCount)
	}

	// Verify final state
	var availQty, resQty int32
	_ = db.Pool.QueryRow(context.Background(), `
		SELECT available_quantity, reserved_quantity FROM inventory.stock_level WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID).Scan(&availQty, &resQty)

	if availQty != 3 {
		t.Errorf("expected available stock 3, got %d", availQty)
	}
	if resQty != 7 {
		t.Errorf("expected reserved stock 7, got %d", resQty)
	}
}

func TestIntegration_StaleReservationRelease(t *testing.T) {
	db, svc, _ := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	productID, supplierID := createTestProductAndSupplier(t, db)
	_, lotID := createTestStockAndLot(t, db, productID, supplierID, "LOT-STALE-1", 10, time.Now().Add(24*time.Hour))

	// Directly insert an expired reservation
	var resID int64
	pastTime := time.Now().Add(-1 * time.Hour)
	err := db.Pool.QueryRow(context.Background(), `
		INSERT INTO inventory.reservation (code, lot_id, request_id, quantity, status, expires_at)
		VALUES ($1, $2, $3, 4, 'reserved', $4)
		RETURNING id
	`, "res_stale_"+uniqueSuffix(), lotID, "req-stale-"+uniqueSuffix(), pastTime).Scan(&resID)
	if err != nil {
		t.Fatalf("failed to insert expired reservation: %v", err)
	}

	// Update lot reserved quantity to reflect active reservation
	_, _ = db.Pool.Exec(context.Background(), `
		UPDATE inventory.lot SET available_quantity = 6, reserved_quantity = 4 WHERE id = $1
	`, lotID)
	_, _ = db.Pool.Exec(context.Background(), `
		UPDATE inventory.stock_level SET available_quantity = 6, reserved_quantity = 4 WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID)

	released, err := svc.ReleaseExpiredReservations(context.Background())
	if err != nil {
		t.Fatalf("failed to release expired reservations: %v", err)
	}

	if released < 1 {
		t.Errorf("expected at least 1 released reservation, got %d", released)
	}

	// Verify stock restored
	var availQty int32
	_ = db.Pool.QueryRow(context.Background(), `
		SELECT available_quantity FROM inventory.stock_level WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID).Scan(&availQty)
	if availQty != 10 {
		t.Errorf("expected available stock restored to 10, got %d", availQty)
	}
}

func TestIntegration_QuarantineExclusion(t *testing.T) {
	db, svc, _ := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	productID, supplierID := createTestProductAndSupplier(t, db)
	_, lotID := createTestStockAndLot(t, db, productID, supplierID, "LOT-QUAR-1", 10, time.Now().Add(24*time.Hour))

	// Quarantine lot
	_, err := svc.QuarantineLot(context.Background(), schema.QuarantineLotRequest{
		LotID:  lotID,
		Reason: "Quality check failure",
	}, 0)
	if err != nil {
		t.Fatalf("failed to quarantine lot: %v", err)
	}

	// Attempt reservation
	req := schema.ReserveStockRequest{
		ProductID:  productID,
		SupplierID: supplierID,
		Quantity:   5,
		RequestID:  fmt.Sprintf("req-quar-%s", uniqueSuffix()),
	}

	_, err = svc.ReserveStock(context.Background(), req)
	if !errorsIs(err, service.ErrInsufficientStock) {
		t.Errorf("expected ErrInsufficientStock when allocating quarantined lot, got %v", err)
	}
}

func errorsIs(err, target error) bool {
	return err != nil && (err.Error() == target.Error() || fmt.Sprintf("%v", err) == fmt.Sprintf("%v", target))
}
