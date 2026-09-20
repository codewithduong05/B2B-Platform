package router_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/inventory/router"
	"github.com/atlas-platform/backend/internal/modules/inventory/schema"
	"github.com/atlas-platform/backend/internal/modules/inventory/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
			_, _ = rawDB.Exec("DROP SCHEMA IF EXISTS catalog CASCADE; DROP SCHEMA IF EXISTS inventory CASCADE; DROP SCHEMA IF EXISTS identity CASCADE; DROP SCHEMA IF EXISTS pricing CASCADE; DROP SCHEMA IF EXISTS commerce CASCADE; DROP SCHEMA IF EXISTS payments CASCADE; DROP SCHEMA IF EXISTS promotions CASCADE; DROP SCHEMA IF EXISTS crm CASCADE; DROP SCHEMA IF EXISTS suppliers CASCADE; DROP SCHEMA IF EXISTS cms CASCADE; DROP SCHEMA IF EXISTS reports CASCADE; DROP SCHEMA IF EXISTS erp CASCADE; DROP SCHEMA IF EXISTS platform CASCADE; DROP SCHEMA IF EXISTS ai CASCADE; DROP SCHEMA IF EXISTS analytics CASCADE; DROP TABLE IF EXISTS schema_migrations CASCADE; DROP TYPE IF EXISTS catalog_handling_class_type CASCADE;")

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
	var out []eventRecord
	for _, e := range p.events {
		if e.RoutingKey == routingKey {
			out = append(out, e)
		}
	}
	return out
}

type testEnv struct {
	db        *database.DB
	srv       *httptest.Server
	publisher *recordingPublisher
}

func nopMiddleware(next http.Handler) http.Handler {
	return next
}

func denyStatusMiddleware(status int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		})
	}
}

func withPrincipalMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(router.WithPrincipalID(r.Context(), 7)))
	})
}

func setupEnv(t *testing.T, auth, admin func(http.Handler) http.Handler) *testEnv {
	t.Helper()
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
	rt := router.New(svc)
	rt.RegisterRoutes(auth, admin)

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Mount("/api/v1", rt.ChiRouter())
	server := httptest.NewServer(mux)

	t.Cleanup(func() {
		server.Close()
		db.Close()
	})

	return &testEnv{db: db, srv: server, publisher: publisher}
}

func doJSON(t *testing.T, env *testEnv, method, path, body string) (*http.Response, []byte) {
	t.Helper()
	return sendJSON(t, env, method, path, body, nil)
}

func sendJSON(t *testing.T, env *testEnv, method, path, body string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	var reqBody *bytes.Reader
	if body == "" {
		reqBody = bytes.NewReader(nil)
	} else {
		reqBody = bytes.NewReader([]byte(body))
	}
	req, err := http.NewRequest(method, env.srv.URL+path, reqBody)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var out bytes.Buffer
	if _, err := out.ReadFrom(resp.Body); err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	return resp, out.Bytes()
}

func decodeErr(t *testing.T, body []byte) map[string]string {
	t.Helper()
	var m map[string]string
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("failed to decode error body %q: %v", string(body), err)
	}
	return m
}

var testCounter int64

func uniqueSuffix() string {
	c := atomic.AddInt64(&testCounter, 1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano()%1000000, c)
}

func numericSupplierID() int64 {
	return 1000000 + atomic.AddInt64(&testCounter, 1)
}

func createTestUser(t *testing.T, env *testEnv, id int64) {
	t.Helper()
	ctx := context.Background()
	code := fmt.Sprintf("usr_%d_%s", id, uniqueSuffix())
	_, err := env.db.Pool.Exec(ctx, `
		INSERT INTO identity.user (id, code, email, password_hash, user_type)
		VALUES ($1, $2, $3, $4, 'staff')
		ON CONFLICT (id) DO NOTHING
	`, id, code, fmt.Sprintf("staff%d@test.local", id), "not-a-real-hash")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
}

func createTestProductAndSupplier(t *testing.T, env *testEnv) (int64, int64) {
	t.Helper()
	ctx := context.Background()
	suffix := uniqueSuffix()

	var categoryID, unitID, supplierID, productID int64

	err := env.db.WithTx(ctx, func(tx *database.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO catalog.category (code, name, slug)
			VALUES ($1, $2, $3)
			RETURNING id
		`, "cat_"+suffix, "Test Category", "test-cat-"+suffix).Scan(&categoryID)
		if err != nil {
			return err
		}

		err = tx.QueryRow(ctx, `
			INSERT INTO catalog.unit (code, name, symbol, unit_type)
			VALUES ($1, $2, $3, 'base')
			RETURNING id
		`, "unit_"+suffix, "Piece", "pcs").Scan(&unitID)
		if err != nil {
			return err
		}

		err = tx.QueryRow(ctx, `
			INSERT INTO catalog.supplier (code, supplier_id, name, slug)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, "sup_"+suffix, numericSupplierID(), "Test Supplier", "test-sup-"+suffix).Scan(&supplierID)
		if err != nil {
			return err
		}

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

func createTestStockAndLot(t *testing.T, env *testEnv, productID, supplierID int64, lotNumber string, initialQty int32, expiresAt time.Time) (int64, int64) {
	t.Helper()
	ctx := context.Background()
	suffix := uniqueSuffix()

	var stockLevelID, lotID int64

	err := env.db.WithTx(ctx, func(tx *database.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO inventory.stock_level (code, product_id, supplier_id, available_quantity, total_quantity)
			VALUES ($1, $2, $3, $4, $4)
			ON CONFLICT (product_id, supplier_id) DO UPDATE
			SET available_quantity = inventory.stock_level.available_quantity + $4,
			    total_quantity = inventory.stock_level.total_quantity + $4
			RETURNING id
		`, "sl_"+suffix, productID, supplierID, initialQty).Scan(&stockLevelID)
		if err != nil {
			err = tx.QueryRow(ctx, `
				SELECT id FROM inventory.stock_level WHERE product_id = $1 AND supplier_id = $2
			`, productID, supplierID).Scan(&stockLevelID)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `
				UPDATE inventory.stock_level
				SET available_quantity = available_quantity + $1, total_quantity = total_quantity + $1
				WHERE id = $2
			`, initialQty, stockLevelID)
			if err != nil {
				return err
			}
		}

		err = tx.QueryRow(ctx, `
			INSERT INTO inventory.lot (code, stock_level_id, lot_number, initial_quantity, available_quantity, status, expires_at)
			VALUES ($1, $2, $3, $4, $4, 'active', $5)
			RETURNING id
		`, "lot_"+suffix, stockLevelID, lotNumber, initialQty, expiresAt).Scan(&lotID)
		return err
	})

	if err != nil {
		t.Fatalf("failed to create test stock and lot: %v", err)
	}
	return stockLevelID, lotID
}

func reserveBody(productID, supplierID int64, quantity int32, requestID string) string {
	payload := fmt.Sprintf(`{"product_id":%d,"supplier_id":%d,"quantity":%d,"request_id":"%s"}`, productID, supplierID, quantity, requestID)
	return payload
}

func TestHTTP_Availability_Valid(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, productID, supplierID, "LOT-AV-1", 10, time.Now().Add(24*time.Hour))

	resp, body := doJSON(t, env, http.MethodGet, fmt.Sprintf("/api/v1/inventory/availability?product_ids=%d&supplier_id=%d", productID, supplierID), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var summaries []schema.StockLevelSummary
	if err := json.Unmarshal(body, &summaries); err != nil {
		t.Fatalf("failed to decode availability response %q: %v", string(body), err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 stock level, got %d", len(summaries))
	}
	if summaries[0].ProductID != productID || summaries[0].SupplierID != supplierID {
		t.Errorf("unexpected stock level identity: %+v", summaries[0])
	}
	if summaries[0].AvailableQuantity != 10 {
		t.Errorf("expected available 10, got %d", summaries[0].AvailableQuantity)
	}
}

func TestHTTP_Availability_InvalidQuery(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)

	cases := []struct {
		name string
		path string
	}{
		{"missing product_ids", "/api/v1/inventory/availability"},
		{"bad product_ids", "/api/v1/inventory/availability?product_ids=abc"},
		{"bad supplier_id", "/api/v1/inventory/availability?product_ids=1&supplier_id=xyz"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := doJSON(t, env, http.MethodGet, tc.path, "")
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(body))
			}
			if code := decodeErr(t, body)["code"]; code != "invalid_query" {
				t.Errorf("expected invalid_query, got %q", code)
			}
		})
	}
}

func TestHTTP_Reservation_Created(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, productID, supplierID, "LOT-CR-1", 5, time.Now().Add(1*time.Hour))
	createTestStockAndLot(t, env, productID, supplierID, "LOT-CR-2", 10, time.Now().Add(2*time.Hour))

	requestID := "req-cre-created" + uniqueSuffix()
	resp, body := doJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations", reserveBody(productID, supplierID, 12, requestID))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	if loc := resp.Header.Get("Location"); loc != "/api/v1/inventory/reservations/request/"+requestID {
		t.Errorf("unexpected Location header %q", loc)
	}

	var out schema.ReservationResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("failed to decode reservation response %q: %v", string(body), err)
	}
	if out.RequestID != requestID {
		t.Errorf("expected request_id %s, got %s", requestID, out.RequestID)
	}
	if out.Status != "reserved" {
		t.Errorf("expected status reserved, got %s", out.Status)
	}
	if out.Quantity != 12 {
		t.Errorf("expected quantity 12, got %d", out.Quantity)
	}
	if len(out.Allocations) != 2 {
		t.Fatalf("expected 2 allocations (multi-lot FEFO), got %d", len(out.Allocations))
	}
	allocMap := make(map[int64]int32)
	for _, a := range out.Allocations {
		allocMap[a.LotID] += a.Quantity
	}
	got := make([]int32, 0, len(allocMap))
	for _, q := range allocMap {
		got = append(got, q)
	}
	if len(got) != 2 {
		t.Fatalf("expected allocations across 2 lots, got %d: %v", len(got), got)
	}
	if out.ExpiresAt.IsZero() {
		t.Errorf("expected non-zero expires_at")
	}
	for _, a := range out.Allocations {
		if a.Status != "reserved" {
			t.Errorf("expected allocation status reserved, got %s", a.Status)
		}
		if a.RequestID != requestID {
			t.Errorf("expected allocation request_id %s, got %s", requestID, a.RequestID)
		}
	}

	var rowCount int
	err := env.db.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM inventory.reservation
		WHERE request_id = $1 AND deleted_at IS NULL
	`, requestID).Scan(&rowCount)
	if err != nil {
		t.Fatalf("failed to count reservation rows: %v", err)
	}
	if rowCount != 2 {
		t.Errorf("expected exactly 2 reservation rows (one per lot), got %d", rowCount)
	}

	reservedEvents := env.publisher.getEvents("inventory.stock.reserved")
	if len(reservedEvents) != 2 {
		t.Errorf("expected 2 inventory.stock.reserved events (one per allocation), got %d", len(reservedEvents))
	}
	for _, e := range reservedEvents {
		if e.Payload == nil {
			t.Errorf("expected reserved event payload, got nil")
		}
	}
}

func TestHTTP_Reservation_InsufficientStock(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, productID, supplierID, "LOT-IS-1", 3, time.Now().Add(24*time.Hour))

	requestID := "req-insufficient-" + uniqueSuffix()
	resp, body := doJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations", reserveBody(productID, supplierID, 5, requestID))
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(body))
	}
	errBody := decodeErr(t, body)
	if code := errBody["code"]; code != "insufficient_stock" {
		t.Errorf("expected insufficient_stock, got %q", code)
	}
	if errBody["detail"] == "" {
		t.Errorf("expected non-empty detail in error envelope")
	}

	var rowCount int
	err := env.db.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM inventory.reservation
		WHERE request_id = $1 AND deleted_at IS NULL
	`, requestID).Scan(&rowCount)
	if err != nil {
		t.Fatalf("failed to count reservation rows: %v", err)
	}
	if rowCount != 0 {
		t.Errorf("expected NO reservation rows on insufficient stock, got %d", rowCount)
	}

	var availQty, reservedQty int32
	err = env.db.Pool.QueryRow(context.Background(), `
		SELECT available_quantity, reserved_quantity FROM inventory.stock_level WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID).Scan(&availQty, &reservedQty)
	if err != nil {
		t.Fatalf("failed to query stock level: %v", err)
	}
	if availQty != 3 || reservedQty != 0 {
		t.Errorf("expected stock unchanged (available 3, reserved 0), got %d / %d", availQty, reservedQty)
	}

	shortfalls := env.publisher.getEvents("inventory.stock.shortfall")
	if len(shortfalls) != 1 {
		t.Errorf("expected 1 inventory.stock.shortfall event, got %d", len(shortfalls))
	}
	if _, ok := shortfalls[0].Payload.(map[string]interface{}); !ok {
		t.Errorf("expected shortfall event payload map, got %T", shortfalls[0].Payload)
	}
}

func TestHTTP_Reservation_InvalidQuantity(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, productID, supplierID, "LOT-IV-1", 10, time.Now().Add(24*time.Hour))

	body := fmt.Sprintf(`{"product_id":%d,"supplier_id":%d,"quantity":0,"request_id":"req-%s"}`, productID, supplierID, uniqueSuffix())
	resp, out := doJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(out))
	}
	if code := decodeErr(t, out)["code"]; code != "invalid_quantity" {
		t.Errorf("expected invalid_quantity, got %q", code)
	}
}

func TestHTTP_Reservation_MissingRequestID(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, productID, supplierID, "LOT-MR-1", 10, time.Now().Add(24*time.Hour))

	body := fmt.Sprintf(`{"product_id":%d,"supplier_id":%d,"quantity":1}`, productID, supplierID)
	resp, out := doJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(out))
	}
	if code := decodeErr(t, out)["code"]; code != "invalid_body" {
		t.Errorf("expected invalid_body, got %q", code)
	}
}

func TestHTTP_Reservation_IdempotentReplay(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, productID, supplierID, "LOT-IP-1", 10, time.Now().Add(24*time.Hour))

	requestID := "req-replay-" + uniqueSuffix()
	first, body1 := doJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations", reserveBody(productID, supplierID, 4, requestID))
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("expected first request 201, got %d: %s", first.StatusCode, string(body1))
	}

	second, body2 := doJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations", reserveBody(productID, supplierID, 4, requestID))
	if second.StatusCode != http.StatusOK {
		t.Fatalf("expected replay 200, got %d: %s", second.StatusCode, string(body2))
	}
	if loc := second.Header.Get("Location"); loc != "" {
		t.Errorf("expected no Location header on replay, got %q", loc)
	}

	var out1, out2 schema.ReservationResponse
	if err := json.Unmarshal(body1, &out1); err != nil {
		t.Fatalf("decode first: %v", err)
	}
	if err := json.Unmarshal(body2, &out2); err != nil {
		t.Fatalf("decode second: %v", err)
	}

	if out1.Quantity != out2.Quantity || out1.Quantity != 4 {
		t.Errorf("expected identical logical quantity 4 on both, got %d / %d", out1.Quantity, out2.Quantity)
	}
	if len(out1.Allocations) != len(out2.Allocations) || len(out1.Allocations) != 1 {
		t.Errorf("expected 1 allocation on both, got %d / %d", len(out1.Allocations), len(out2.Allocations))
	}
	if out2.Allocations[0].ID != out1.Allocations[0].ID {
		t.Errorf("expected replay to return the existing reservation, got id %d vs %d", out2.Allocations[0].ID, out1.Allocations[0].ID)
	}

	var rowCount int
	err := env.db.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM inventory.reservation
		WHERE request_id = $1 AND deleted_at IS NULL
	`, requestID).Scan(&rowCount)
	if err != nil {
		t.Fatalf("failed to count reservation rows: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("expected exactly 1 reservation row after replay (no duplicate), got %d", rowCount)
	}

	var lotAvail, lotReserved int32
	err = env.db.Pool.QueryRow(context.Background(), `
		SELECT available_quantity, reserved_quantity FROM inventory.lot WHERE id = $1
	`, out1.Allocations[0].LotID).Scan(&lotAvail, &lotReserved)
	if err != nil {
		t.Fatalf("failed to query lot: %v", err)
	}
	if lotAvail != 6 || lotReserved != 4 {
		t.Errorf("expected lot available 6 / reserved 4 (single deduction), got %d / %d", lotAvail, lotReserved)
	}
}

func TestHTTP_Reservation_DuplicateKeyConflict(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, productID, supplierID, "LOT-RACE-1", 10, time.Now().Add(24*time.Hour))

	requestID := "req-race-" + uniqueSuffix()
	body := reserveBody(productID, supplierID, 1, requestID)

	type result struct {
		status int
		body   []byte
	}
	results := make(chan result, 10)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodPost, env.srv.URL+"/api/v1/inventory/reservations", bytes.NewReader([]byte(body)))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(resp.Body)
			results <- result{status: resp.StatusCode, body: buf.Bytes()}
		}()
	}
	wg.Wait()
	close(results)

	sawCreated, sawConflict := false, false
	for res := range results {
		switch res.status {
		case http.StatusCreated:
			sawCreated = true
		case http.StatusConflict:
			sawConflict = true
		case http.StatusOK:
			// replay is also valid
		default:
			t.Errorf("unexpected status %d: %s", res.status, string(res.body))
		}
	}

	if !sawCreated {
		t.Errorf("expected at least one 201")
	}
	if !sawConflict {
		t.Errorf("expected at least one 409 duplicate_request under concurrent race")
	}

	var reservedQty int32
	err := env.db.Pool.QueryRow(context.Background(), `
		SELECT COALESCE(SUM(quantity), 0) FROM inventory.reservation WHERE request_id = $1 AND deleted_at IS NULL
	`, requestID).Scan(&reservedQty)
	if err != nil {
		t.Fatalf("failed to count reservations: %v", err)
	}
	if reservedQty != 1 {
		t.Errorf("expected exactly 1 unit reserved despite concurrent storm, got %d", reservedQty)
	}
}

func TestHTTP_GetReservation_Valid(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, productID, supplierID, "LOT-GR-1", 10, time.Now().Add(24*time.Hour))

	requestID := "req-get-" + uniqueSuffix()
	resp, body := doJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations", reserveBody(productID, supplierID, 3, requestID))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("setup reservation failed: %d %s", resp.StatusCode, string(body))
	}

	var created schema.ReservationResponse
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	allocationID := created.Allocations[0].ID

	resp, out := doJSON(t, env, http.MethodGet, fmt.Sprintf("/api/v1/inventory/reservations/%d", allocationID), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(out))
	}
	var single schema.ReservationSummary
	if err := json.Unmarshal(out, &single); err != nil {
		t.Fatalf("decode single: %v", err)
	}
	if single.ID != allocationID || single.RequestID != requestID {
		t.Errorf("unexpected reservation: %+v", single)
	}
}

func TestHTTP_GetReservation_InvalidAndMissing(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)

	resp, out := doJSON(t, env, http.MethodGet, "/api/v1/inventory/reservations/abc", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d: %s", resp.StatusCode, string(out))
	}
	if code := decodeErr(t, out)["code"]; code != "invalid_id" {
		t.Errorf("expected invalid_id, got %q", code)
	}

	resp, out = doJSON(t, env, http.MethodGet, "/api/v1/inventory/reservations/99999999", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing reservation, got %d: %s", resp.StatusCode, string(out))
	}
	if code := decodeErr(t, out)["code"]; code != "reservation_not_found" {
		t.Errorf("expected reservation_not_found, got %q", code)
	}
}

func TestHTTP_GetReservationByRequestID(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, productID, supplierID, "LOT-RQ-1", 8, time.Now().Add(1*time.Hour))
	createTestStockAndLot(t, env, productID, supplierID, "LOT-RQ-2", 5, time.Now().Add(2*time.Hour))

	requestID := "req-lookup-" + uniqueSuffix()
	resp, body := doJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations", reserveBody(productID, supplierID, 10, requestID))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("setup reservation failed: %d %s", resp.StatusCode, string(body))
	}

	resp, out := doJSON(t, env, http.MethodGet, "/api/v1/inventory/reservations/request/"+requestID, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(out))
	}
	var envelope schema.ReservationResponse
	if err := json.Unmarshal(out, &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.RequestID != requestID {
		t.Errorf("expected request_id %s, got %s", requestID, envelope.RequestID)
	}
	if envelope.Quantity != 10 {
		t.Errorf("expected quantity 10, got %d", envelope.Quantity)
	}
	if len(envelope.Allocations) != 2 {
		t.Errorf("expected 2 allocations, got %d", len(envelope.Allocations))
	}

	resp, out = doJSON(t, env, http.MethodGet, "/api/v1/inventory/reservations/request/req-does-not-exist-"+uniqueSuffix(), "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing request_id, got %d: %s", resp.StatusCode, string(out))
	}
	if code := decodeErr(t, out)["code"]; code != "reservation_not_found" {
		t.Errorf("expected reservation_not_found, got %q", code)
	}

	tooLong := fmt.Sprintf("req-%s%s", uniqueSuffix(), string(bytes.Repeat([]byte("x"), 95)))
	resp, out = doJSON(t, env, http.MethodGet, "/api/v1/inventory/reservations/request/"+tooLong, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for over-length request_id, got %d: %s", resp.StatusCode, string(out))
	}
	if code := decodeErr(t, out)["code"]; code != "invalid_request_id" {
		t.Errorf("expected invalid_request_id, got %q", code)
	}
}

func TestHTTP_Quarantine_Valid(t *testing.T) {
	env := setupEnv(t, nopMiddleware, withPrincipalMiddleware)
	createTestUser(t, env, 7)
	productID, supplierID := createTestProductAndSupplier(t, env)
	_, lotID := createTestStockAndLot(t, env, productID, supplierID, "LOT-QV-1", 10, time.Now().Add(24*time.Hour))

	request := `{"reason":"quality hold"}`
	resp, body := doJSON(t, env, http.MethodPost, fmt.Sprintf("/api/v1/inventory/lots/%d/quarantine", lotID), request)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var lot schema.LotSummary
	if err := json.Unmarshal(body, &lot); err != nil {
		t.Fatalf("decode lot: %v", err)
	}
	if !lot.IsQuarantined {
		t.Errorf("expected lot quarantined")
	}
	if lot.AvailableQuantity != 0 {
		t.Errorf("expected available quantity 0, got %d", lot.AvailableQuantity)
	}
	if lot.Status != "quarantined" {
		t.Errorf("expected status quarantined, got %s", lot.Status)
	}

	var adjustedBy sql.NullInt64
	var reason string
	err := env.db.Pool.QueryRow(context.Background(), `
		SELECT reason, adjusted_by FROM inventory.quarantine_record WHERE lot_id = $1 AND deleted_at IS NULL
	`, lotID).Scan(&reason, &adjustedBy)
	if err != nil {
		t.Fatalf("failed to query quarantine record: %v", err)
	}
	if reason != "quality hold" {
		t.Errorf("expected reason stored, got %q", reason)
	}
	if !adjustedBy.Valid || adjustedBy.Int64 != 7 {
		t.Errorf("expected adjusted_by 7 from authenticated principal, got %+v", adjustedBy)
	}
}

func TestHTTP_Quarantine_InvalidAndMissing(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, productID, supplierID, "LOT-QM-1", 10, time.Now().Add(24*time.Hour))

	resp, out := doJSON(t, env, http.MethodPost, "/api/v1/inventory/lots/abc/quarantine", `{"reason":"x"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid lot id, got %d: %s", resp.StatusCode, string(out))
	}
	if code := decodeErr(t, out)["code"]; code != "invalid_id" {
		t.Errorf("expected invalid_id, got %q", code)
	}

	resp, out = doJSON(t, env, http.MethodPost, "/api/v1/inventory/lots/99999999/quarantine", `{"reason":"x"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing lot, got %d: %s", resp.StatusCode, string(out))
	}
	if code := decodeErr(t, out)["code"]; code != "lot_not_found" {
		t.Errorf("expected lot_not_found, got %q", code)
	}
}

func TestHTTP_Quarantine_MissingReason(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	_, lotID := createTestStockAndLot(t, env, productID, supplierID, "LOT-QR-1", 10, time.Now().Add(24*time.Hour))

	resp, out := doJSON(t, env, http.MethodPost, fmt.Sprintf("/api/v1/inventory/lots/%d/quarantine", lotID), `{}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing reason, got %d: %s", resp.StatusCode, string(out))
	}
	if code := decodeErr(t, out)["code"]; code != "invalid_body" {
		t.Errorf("expected invalid_body, got %q", code)
	}
}

func TestHTTP_Quarantine_AlreadyQuarantined(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	_, lotID := createTestStockAndLot(t, env, productID, supplierID, "LOT-QA-1", 10, time.Now().Add(24*time.Hour))

	request := `{"reason":"first quarantine"}`
	resp, body := doJSON(t, env, http.MethodPost, fmt.Sprintf("/api/v1/inventory/lots/%d/quarantine", lotID), request)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first quarantine expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var first schema.LotSummary
	if err := json.Unmarshal(body, &first); err != nil {
		t.Fatalf("decode first: %v", err)
	}

	resp, body = doJSON(t, env, http.MethodPost, fmt.Sprintf("/api/v1/inventory/lots/%d/quarantine", lotID), `{"reason":"again"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("idempotent re-quarantine expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var second schema.LotSummary
	if err := json.Unmarshal(body, &second); err != nil {
		t.Fatalf("decode second: %v", err)
	}
	if !second.IsQuarantined {
		t.Errorf("expected lot still quarantined")
	}
	if second.ID != first.ID {
		t.Errorf("expected current lot returned, got id %d", second.ID)
	}

	var count int
	err := env.db.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM inventory.quarantine_record WHERE lot_id = $1 AND deleted_at IS NULL
	`, lotID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count quarantine records: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 quarantine record (idempotent), got %d", count)
	}
}

func TestHTTP_Quarantine_Unauthorized(t *testing.T) {
	deny := denyStatusMiddleware(http.StatusUnauthorized)
	env := setupEnv(t, deny, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	_, lotID := createTestStockAndLot(t, env, productID, supplierID, "LOT-QU-1", 10, time.Now().Add(24*time.Hour))

	resp, body := doJSON(t, env, http.MethodPost, fmt.Sprintf("/api/v1/inventory/lots/%d/quarantine", lotID), `{"reason":"x"}`)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestHTTP_Quarantine_Forbidden(t *testing.T) {
	deny := denyStatusMiddleware(http.StatusForbidden)
	env := setupEnv(t, nopMiddleware, deny)
	productID, supplierID := createTestProductAndSupplier(t, env)
	_, lotID := createTestStockAndLot(t, env, productID, supplierID, "LOT-QF-1", 10, time.Now().Add(24*time.Hour))

	resp, body := doJSON(t, env, http.MethodPost, fmt.Sprintf("/api/v1/inventory/lots/%d/quarantine", lotID), `{"reason":"x"}`)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestHTTP_Availability_Filtering(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	p1, s1 := createTestProductAndSupplier(t, env)
	p2, s2 := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, p1, s1, "LOT-AVF-1", 4, time.Now().Add(24*time.Hour))
	createTestStockAndLot(t, env, p1, s2, "LOT-AVF-2", 9, time.Now().Add(24*time.Hour))

	var sls []schema.StockLevelSummary

	// supplier filter: only that supplier's stock levels, unrelated product absent
	resp, body := doJSON(t, env, http.MethodGet, fmt.Sprintf("/api/v1/inventory/availability?product_ids=%d,%d&supplier_id=%d", p1, p2, s1), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, &sls); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(sls) != 1 {
		t.Fatalf("expected 1 stock level for supplier filter, got %d", len(sls))
	}
	if sls[0].SupplierID != s1 || sls[0].AvailableQuantity != 4 {
		t.Errorf("unexpected filtered stock level: %+v", sls[0])
	}

	// product without any stock level is absent (filter semantics, no error)
	resp, body = doJSON(t, env, http.MethodGet, fmt.Sprintf("/api/v1/inventory/availability?product_ids=%d", p2), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, &sls); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(sls) != 0 {
		t.Errorf("expected empty list for product without stock, got %d", len(sls))
	}

	// no supplier filter: one summary per supplier
	resp, body = doJSON(t, env, http.MethodGet, fmt.Sprintf("/api/v1/inventory/availability?product_ids=%d", p1), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, &sls); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(sls) != 2 {
		t.Fatalf("expected 2 stock levels (one per supplier), got %d", len(sls))
	}
	suppliers := map[int64]bool{}
	for _, s := range sls {
		suppliers[s.SupplierID] = true
	}
	if !suppliers[s1] || !suppliers[s2] {
		t.Errorf("expected stock levels for both suppliers, got %v", suppliers)
	}
}

func TestHTTP_Reservation_OverLengthRequestID(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, productID, supplierID, "LOT-OR-1", 10, time.Now().Add(24*time.Hour))

	requestID := "req-" + strings.Repeat("x", 100)
	if len(requestID) <= 100 {
		t.Fatalf("helper: want over-length request_id, got %d chars", len(requestID))
	}
	resp, out := doJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations", reserveBody(productID, supplierID, 1, requestID))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for over-length request_id, got %d: %s", resp.StatusCode, string(out))
	}
	if code := decodeErr(t, out)["code"]; code != "invalid_request_id" {
		t.Errorf("expected invalid_request_id, got %q", code)
	}
}

func TestHTTP_Reservation_PayloadMismatchReplay(t *testing.T) {
	// Current documented behavior (PRODUCT DECISION REQUIRED): the same request_id
	// with a different payload is replayed as-is — recorded allocations are
	// returned with 200 and no further deduction. Payload equality is NOT enforced.
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, productID, supplierID, "LOT-PM-1", 10, time.Now().Add(24*time.Hour))

	requestID := "req-mismatch-" + uniqueSuffix()

	first, body1 := doJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations", reserveBody(productID, supplierID, 5, requestID))
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("expected first request 201, got %d: %s", first.StatusCode, string(body1))
	}

	second, body2 := doJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations", reserveBody(productID, supplierID, 8, requestID))
	if second.StatusCode != http.StatusOK {
		t.Fatalf("expected replay 200, got %d: %s", second.StatusCode, string(body2))
	}

	var out1, out2 schema.ReservationResponse
	if err := json.Unmarshal(body1, &out1); err != nil {
		t.Fatalf("decode first: %v", err)
	}
	if err := json.Unmarshal(body2, &out2); err != nil {
		t.Fatalf("decode second: %v", err)
	}
	if out2.Quantity != 5 || out1.Quantity != 5 {
		t.Errorf("expected replay to return recorded quantity 5, got %d (first %d)", out2.Quantity, out1.Quantity)
	}
	if len(out2.Allocations) != 1 || out2.Allocations[0].ID != out1.Allocations[0].ID {
		t.Errorf("expected replay to return the recorded allocation, got %+v", out2.Allocations)
	}

	var availQty int32
	err := env.db.Pool.QueryRow(context.Background(), `
		SELECT available_quantity FROM inventory.stock_level WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID).Scan(&availQty)
	if err != nil {
		t.Fatalf("query stock: %v", err)
	}
	if availQty != 5 {
		t.Errorf("expected available 5 (single deduction of 5 despite mismatched retry), got %d", availQty)
	}

	var rowCount int
	err = env.db.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM inventory.reservation WHERE request_id = $1 AND deleted_at IS NULL
	`, requestID).Scan(&rowCount)
	if err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("expected exactly 1 reservation row, got %d", rowCount)
	}
}

func TestHTTP_Reservation_MultiLotFEFOExact(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	now := time.Now()
	_, lotA := createTestStockAndLot(t, env, productID, supplierID, "LOT-FEFO-A", 5, now.Add(1*time.Hour))
	_, lotB := createTestStockAndLot(t, env, productID, supplierID, "LOT-FEFO-B", 10, now.Add(2*time.Hour))
	_, lotC := createTestStockAndLot(t, env, productID, supplierID, "LOT-FEFO-C", 20, now.Add(3*time.Hour))

	requestID := "req-fefo-" + uniqueSuffix()
	resp, body := doJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations", reserveBody(productID, supplierID, 12, requestID))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	var out schema.ReservationResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode response %q: %v", string(body), err)
	}
	if out.Quantity != 12 {
		t.Errorf("expected total quantity 12, got %d", out.Quantity)
	}

	allocMap := make(map[int64]int32)
	for _, a := range out.Allocations {
		allocMap[a.LotID] += a.Quantity
	}
	if len(allocMap) != 2 {
		t.Fatalf("expected 2 lots used (FEFO A then B), got %d: %v", len(allocMap), allocMap)
	}
	if allocMap[lotA] != 5 {
		t.Errorf("expected lot A allocated 5 (earliest expiry, full consumption), got %d", allocMap[lotA])
	}
	if allocMap[lotB] != 7 {
		t.Errorf("expected lot B allocated 7 (remainder), got %d", allocMap[lotB])
	}
	if _, used := allocMap[lotC]; used {
		t.Errorf("lot C must not be allocated (latest expiry)")
	}

	if out.Allocations[0].LotID != lotA || out.Allocations[1].LotID != lotB {
		t.Errorf("expected allocations ordered FEFO (A then B), got %+v", out.Allocations)
	}

	lotState := func(lotID int64) (avail, reserved int32) {
		err := env.db.Pool.QueryRow(context.Background(), `
			SELECT available_quantity, reserved_quantity FROM inventory.lot WHERE id = $1
		`, lotID).Scan(&avail, &reserved)
		if err != nil {
			t.Fatalf("query lot %d: %v", lotID, err)
		}
		return avail, reserved
	}
	if a, r := lotState(lotA); a != 0 || r != 5 {
		t.Errorf("lot A expected available 0 / reserved 5, got %d / %d", a, r)
	}
	if a, r := lotState(lotB); a != 3 || r != 7 {
		t.Errorf("lot B expected available 3 / reserved 7, got %d / %d", a, r)
	}
	if a, r := lotState(lotC); a != 20 || r != 0 {
		t.Errorf("lot C expected untouched (20/0), got %d / %d", a, r)
	}
}

func TestHTTP_Quarantine_BlocksAllocation(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	_, lotA := createTestStockAndLot(t, env, productID, supplierID, "LOT-QBLK-A", 10, time.Now().Add(24*time.Hour))
	_, lotB := createTestStockAndLot(t, env, productID, supplierID, "LOT-QBLK-B", 10, time.Now().Add(25*time.Hour))

	resp, body := doJSON(t, env, http.MethodPost, fmt.Sprintf("/api/v1/inventory/lots/%d/quarantine", lotA), `{"reason":"block test"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("quarantine lot A failed: %d %s", resp.StatusCode, string(body))
	}

	reqID := "req-qblk-" + uniqueSuffix()
	resp, body = doJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations", reserveBody(productID, supplierID, 15, reqID))
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 after quarantining lot A (only 10 allocatable), got %d: %s", resp.StatusCode, string(body))
	}
	if code := decodeErr(t, body)["code"]; code != "insufficient_stock" {
		t.Errorf("expected insufficient_stock, got %q", code)
	}
	var rowCount int
	err := env.db.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM inventory.reservation WHERE request_id = $1 AND deleted_at IS NULL
	`, reqID).Scan(&rowCount)
	if err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if rowCount != 0 {
		t.Errorf("expected no rows for failed reservation, got %d", rowCount)
	}

	reqID2 := "req-qblk-ok-" + uniqueSuffix()
	resp, body = doJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations", reserveBody(productID, supplierID, 5, reqID2))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 allocated from lot B only, got %d: %s", resp.StatusCode, string(body))
	}
	var out schema.ReservationResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Allocations) != 1 || out.Allocations[0].LotID != lotB {
		t.Errorf("expected single allocation from lot B (lot A excluded by quarantine), got %+v", out.Allocations)
	}
}

func TestHTTP_Quarantine_AdjustedByNotSpoofable(t *testing.T) {
	env := setupEnv(t, nopMiddleware, withPrincipalMiddleware)
	createTestUser(t, env, 7)
	productID, supplierID := createTestProductAndSupplier(t, env)
	_, lotID := createTestStockAndLot(t, env, productID, supplierID, "LOT-SPOOF-1", 10, time.Now().Add(24*time.Hour))

	// Body attempts to set adjusted_by and lot_id; both must be ignored.
	spoof := fmt.Sprintf(`{"reason":"quality hold","adjusted_by":999,"lot_id":%d}`, lotID+99999)
	resp, out := doJSON(t, env, http.MethodPost, fmt.Sprintf("/api/v1/inventory/lots/%d/quarantine", lotID), spoof)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(out))
	}

	var recLotID int64
	var adjustedBy sql.NullInt64
	err := env.db.Pool.QueryRow(context.Background(), `
		SELECT lot_id, adjusted_by FROM inventory.quarantine_record WHERE lot_id = $1 AND deleted_at IS NULL
	`, lotID).Scan(&recLotID, &adjustedBy)
	if err != nil {
		t.Fatalf("query record: %v", err)
	}
	if recLotID != lotID {
		t.Errorf("expected record lot_id from path (%d), not from body, got %d", lotID, recLotID)
	}
	if !adjustedBy.Valid || adjustedBy.Int64 != 7 {
		t.Errorf("expected adjusted_by from authenticated principal (7), not body (999), got %+v", adjustedBy)
	}
}

func TestHTTP_ErrorEnvelope_RequestIDPropagation(t *testing.T) {
	const correlationID = "corr-abc-123"
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, productID, supplierID, "LOT-RID-1", 3, time.Now().Add(24*time.Hour))

	// 422: error envelope echoes the HTTP correlation id, not the idempotency key.
	requestID := "req-rid-" + uniqueSuffix()
	resp, body := sendJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations",
		reserveBody(productID, supplierID, 5, requestID), map[string]string{"X-Request-Id": correlationID})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, string(body))
	}
	if got := decodeErr(t, body)["request_id"]; got != correlationID {
		t.Errorf("expected error envelope to echo correlation id %q, got %q", correlationID, got)
	}

	// 400: same propagation for validation errors.
	resp, body = sendJSON(t, env, http.MethodGet, "/api/v1/inventory/availability?product_ids=abc",
		"", map[string]string{"X-Request-Id": correlationID})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if got := decodeErr(t, body)["request_id"]; got != correlationID {
		t.Errorf("expected correlation id in validation error, got %q", got)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("decode raw error body: %v", err)
	}
	for _, field := range []string{"detail", "code", "request_id"} {
		if raw[field] == nil {
			t.Errorf("expected %q in error envelope, got keys %v", field, raw)
		}
	}
	if raw["errors"] != nil {
		t.Errorf("errors field must not be emitted (reserved, not currently emitted), got %s", raw["errors"])
	}

	// Success envelope: request_id is the inventory idempotency key, distinct from correlation id.
	reqID2 := "req-rid-create-" + uniqueSuffix()
	resp, body = sendJSON(t, env, http.MethodPost, "/api/v1/inventory/reservations",
		reserveBody(productID, supplierID, 2, reqID2), map[string]string{"X-Request-Id": correlationID})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var out schema.ReservationResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode reservation: %v", err)
	}
	if out.RequestID != reqID2 {
		t.Errorf("reservation body request_id must be the idempotency key (%s), got %s", reqID2, out.RequestID)
	}
}

func TestHTTP_ReleaseLot_Valid(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	_, lotID := createTestStockAndLot(t, env, productID, supplierID, "LOT-REL-1", 10, time.Now().Add(24*time.Hour))

	// Quarantine first
	resp, body := doJSON(t, env, http.MethodPost, fmt.Sprintf("/api/v1/inventory/lots/%d/quarantine", lotID), `{"reason":"hold"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("quarantine failed: %d %s", resp.StatusCode, string(body))
	}

	// Release lot
	resp, body = doJSON(t, env, http.MethodPost, fmt.Sprintf("/api/v1/inventory/lots/%d/release", lotID), `{"reason":"passed inspection"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on release, got %d: %s", resp.StatusCode, string(body))
	}

	var lot schema.LotSummary
	if err := json.Unmarshal(body, &lot); err != nil {
		t.Fatalf("decode lot: %v", err)
	}
	if lot.IsQuarantined {
		t.Errorf("expected lot not quarantined after release")
	}
	if lot.AvailableQuantity != 10 {
		t.Errorf("expected available quantity 10, got %d", lot.AvailableQuantity)
	}
}

func TestHTTP_ListLowStock_Valid(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	_, _ = createTestStockAndLot(t, env, productID, supplierID, "LOT-LOW-1", 2, time.Now().Add(24*time.Hour))

	// Set safety stock on stock level to 5 (available is 2 < 5)
	_, err := env.db.Pool.Exec(context.Background(), `
		UPDATE inventory.stock_level SET safety_stock = 5 WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID)
	if err != nil {
		t.Fatalf("failed to update safety stock: %v", err)
	}

	resp, body := doJSON(t, env, http.MethodGet, "/api/v1/inventory/low-stock", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var summaries []schema.LowStockLotSummary
	if err := json.Unmarshal(body, &summaries); err != nil {
		t.Fatalf("decode low stock: %v", err)
	}
	if len(summaries) == 0 {
		t.Errorf("expected low stock lots, got 0")
	}
}

func TestHTTP_ListExpiring_Valid(t *testing.T) {
	env := setupEnv(t, nopMiddleware, nopMiddleware)
	productID, supplierID := createTestProductAndSupplier(t, env)
	createTestStockAndLot(t, env, productID, supplierID, "LOT-EXP-1", 10, time.Now().Add(2*time.Hour))

	resp, body := doJSON(t, env, http.MethodGet, "/api/v1/inventory/expiring?horizon_days=1", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var summaries []schema.ExpiringLotSummary
	if err := json.Unmarshal(body, &summaries); err != nil {
		t.Fatalf("decode expiring: %v", err)
	}
	if len(summaries) == 0 {
		t.Errorf("expected expiring lots, got 0")
	}
}

func TestHTTP_AdjustStock_Valid(t *testing.T) {
	env := setupEnv(t, nopMiddleware, withPrincipalMiddleware)
	createTestUser(t, env, 7)
	productID, supplierID := createTestProductAndSupplier(t, env)
	_, lotID := createTestStockAndLot(t, env, productID, supplierID, "LOT-ADJ-1", 10, time.Now().Add(24*time.Hour))

	reqBody := fmt.Sprintf(`{"lot_id":%d,"quantity_delta":5,"reason_code":"CYCLE_COUNT","reason":"found extra"}`, lotID)
	resp, body := doJSON(t, env, http.MethodPost, "/api/v1/inventory/adjust", reqBody)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var adj schema.StockAdjustmentSummary
	if err := json.Unmarshal(body, &adj); err != nil {
		t.Fatalf("decode adjustment: %v", err)
	}
	if adj.QuantityDelta != 5 {
		t.Errorf("expected quantity delta 5, got %d", adj.QuantityDelta)
	}
	if adj.PreviousQuantity != 10 {
		t.Errorf("expected previous quantity 10, got %d", adj.PreviousQuantity)
	}
	if adj.NewQuantity != 15 {
		t.Errorf("expected new quantity 15, got %d", adj.NewQuantity)
	}
}
