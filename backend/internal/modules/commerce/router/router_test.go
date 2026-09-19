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
	"sync/atomic"
	"testing"
	"time"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/commerce/router"
	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
	commerce_service "github.com/atlas-platform/backend/internal/modules/commerce/service"
	inventory_service "github.com/atlas-platform/backend/internal/modules/inventory/service"
	pricing_service "github.com/atlas-platform/backend/internal/modules/pricing/service"
	promotions_service "github.com/atlas-platform/backend/internal/modules/promotions/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const commerceTestDBLockKey = 1946819412

func TestMain(m *testing.M) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err == nil {
		rawDB, err := sql.Open("pgx", cfg.PostgresDSN())
		if err == nil && rawDB != nil {
			_, _ = rawDB.Exec("SELECT pg_advisory_lock($1)", int64(commerceTestDBLockKey))
			_, _ = rawDB.Exec("DROP SCHEMA IF EXISTS catalog CASCADE; DROP SCHEMA IF EXISTS inventory CASCADE; DROP SCHEMA IF EXISTS identity CASCADE; DROP SCHEMA IF EXISTS pricing CASCADE; DROP SCHEMA IF EXISTS commerce CASCADE; DROP SCHEMA IF EXISTS promotions CASCADE; DROP SCHEMA IF EXISTS crm CASCADE; DROP SCHEMA IF EXISTS payments CASCADE; DROP SCHEMA IF EXISTS reports CASCADE; DROP SCHEMA IF EXISTS erp CASCADE; DROP TABLE IF EXISTS schema_migrations CASCADE; DROP TYPE IF EXISTS catalog_handling_class_type CASCADE;")

			if err := database.RunMigrations(ctx, &cfg.Postgres, "file://../../../../migrations"); err != nil {
				fmt.Printf("TestMain migration error: %v\n", err)
			}

			code := m.Run()
			_, _ = rawDB.Exec("SELECT pg_advisory_unlock($1)", int64(commerceTestDBLockKey))
			rawDB.Close()
			os.Exit(code)
		}
	}
	os.Exit(m.Run())
}

type testEnv struct {
	db  *database.DB
	srv *httptest.Server
}

func nopMiddleware(next http.Handler) http.Handler {
	return next
}

func withPrincipalMiddleware(buyerID int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(router.WithPrincipalID(r.Context(), buyerID)))
		})
	}
}

func setupEnv(t *testing.T, buyerID int64, auth, admin func(http.Handler) http.Handler) *testEnv {
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

	pricingSvc := pricing_service.NewServices(db)
	inventorySvc := inventory_service.NewInventoryService(db, nil)
	promotionSvc := promotions_service.NewPromotionService(db, nil)
	commerceSvc := commerce_service.NewCommerceService(db, pricingSvc.PriceList, inventorySvc, nil, promotionSvc)
	rt := router.New(commerceSvc)
	rt.RegisterRoutes(auth, admin)

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Mount("/api/v1", rt.ChiRouter())
	server := httptest.NewServer(mux)

	t.Cleanup(func() {
		server.Close()
		db.Close()
	})

	return &testEnv{db: db, srv: server}
}

var testCounter int64

func uniqueSuffix() string {
	c := atomic.AddInt64(&testCounter, 1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano()%1000000, c)
}

func createTestBuyer(t *testing.T, env *testEnv) int64 {
	t.Helper()
	ctx := context.Background()
	suffix := uniqueSuffix()
	var userID, buyerProfileID int64

	err := env.db.WithTx(ctx, func(tx *database.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO identity.user (code, email, password_hash, user_type)
			VALUES ($1, $2, 'hash', 'buyer')
			RETURNING id
		`, "usr_"+suffix, "buyer_"+suffix+"@test.local").Scan(&userID)
		if err != nil {
			return err
		}

		err = tx.QueryRow(ctx, `
			INSERT INTO identity.buyer_profile (user_id, code, business_name)
			VALUES ($1, $2, $3)
			RETURNING id
		`, userID, "buyer_prof_"+suffix, "Test Business "+suffix).Scan(&buyerProfileID)
		return err
	})

	if err != nil {
		t.Fatalf("failed to create test buyer: %v", err)
	}
	return buyerProfileID
}

func createTestProduct(t *testing.T, env *testEnv) (string, int64, int64) {
	t.Helper()
	ctx := context.Background()
	suffix := uniqueSuffix()
	var categoryID, unitID, supplierID, productID int64
	var productCode string

	err := env.db.WithTx(ctx, func(tx *database.Tx) error {
		_ = tx.QueryRow(ctx, `
			INSERT INTO catalog.category (code, name, slug)
			VALUES ($1, $2, $3)
			RETURNING id
		`, "cat_"+suffix, "Cat", "cat-"+suffix).Scan(&categoryID)

		_ = tx.QueryRow(ctx, `
			INSERT INTO catalog.unit (code, name, symbol, unit_type)
			VALUES ($1, $2, $3, 'base')
			RETURNING id
		`, "unit_"+suffix, "Piece", "pcs").Scan(&unitID)

		_ = tx.QueryRow(ctx, `
			INSERT INTO catalog.supplier (code, supplier_id, name, slug)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, "sup_"+suffix, 2000+atomic.AddInt64(&testCounter, 1), "Supplier", "sup-"+suffix).Scan(&supplierID)

		productCode = "prod_" + suffix
		err := tx.QueryRow(ctx, `
			INSERT INTO catalog.product (code, slug, name, category_id, handling_class, base_unit_id, supplier_id, status, is_active, base_price_minor)
			VALUES ($1, $2, $3, $4, 'ambient', $5, $6, 'published', true, 1500)
			RETURNING id
		`, productCode, "prod-"+suffix, "Product "+suffix, categoryID, unitID, supplierID).Scan(&productID)
		return err
	})

	if err != nil {
		t.Fatalf("failed to create test product: %v", err)
	}
	return productCode, productID, supplierID
}

func doJSON(t *testing.T, env *testEnv, method, path, body string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	var reqBody *bytes.Reader
	if body == "" {
		reqBody = bytes.NewReader(nil)
	} else {
		reqBody = bytes.NewReader([]byte(body))
	}
	req, err := http.NewRequest(method, env.srv.URL+path, reqBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
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

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	return resp, buf.Bytes()
}

func decodeErr(t *testing.T, body []byte) map[string]string {
	t.Helper()
	var m map[string]string
	_ = json.Unmarshal(body, &m)
	return m
}

func TestHTTP_Cart_GetEmpty(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)

	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)

	resp, body := doJSON(t, envAuth, http.MethodGet, "/api/v1/commerce/cart", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var cart schema.CartResponse
	if err := json.Unmarshal(body, &cart); err != nil {
		t.Fatalf("failed to decode cart response: %v", err)
	}
	if cart.BuyerID != buyerID {
		t.Errorf("expected buyer_id %d, got %d", buyerID, cart.BuyerID)
	}
	if len(cart.Suppliers) != 0 {
		t.Errorf("expected 0 suppliers in empty cart, got %d", len(cart.Suppliers))
	}
}

func TestHTTP_Cart_ItemsCRUD(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	prodCode, _, _ := createTestProduct(t, env)

	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)

	// 1. Add Item
	addReq := fmt.Sprintf(`{"product_code":"%s","quantity":3}`, prodCode)
	resp, body := doJSON(t, envAuth, http.MethodPost, "/api/v1/commerce/cart/items", addReq, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 on add item, got %d: %s", resp.StatusCode, string(body))
	}

	var cart schema.CartResponse
	if err := json.Unmarshal(body, &cart); err != nil {
		t.Fatalf("decode cart: %v", err)
	}
	if len(cart.Suppliers) != 1 {
		t.Fatalf("expected 1 supplier group, got %d", len(cart.Suppliers))
	}
	if len(cart.Suppliers[0].Items) != 1 {
		t.Fatalf("expected 1 cart line, got %d", len(cart.Suppliers[0].Items))
	}
	line := cart.Suppliers[0].Items[0]
	if line.Quantity != 3 || line.ProductCode != prodCode {
		t.Errorf("unexpected cart line: %+v", line)
	}
	lineCode := line.Code

	// 2. Update Item Quantity (PATCH)
	updateReq := `{"quantity":5}`
	resp, body = doJSON(t, envAuth, http.MethodPatch, fmt.Sprintf("/api/v1/commerce/cart/items/%s", lineCode), updateReq, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on update item, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &cart)
	if cart.Suppliers[0].Items[0].Quantity != 5 {
		t.Errorf("expected quantity 5, got %d", cart.Suppliers[0].Items[0].Quantity)
	}

	// 3. Quote Cart (POST /commerce/cart/quote)
	resp, body = doJSON(t, envAuth, http.MethodPost, "/api/v1/commerce/cart/quote", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on quote, got %d: %s", resp.StatusCode, string(body))
	}
	var quote schema.CartQuoteResponse
	if err := json.Unmarshal(body, &quote); err != nil {
		t.Fatalf("decode quote: %v", err)
	}
	if quote.TotalMinor != 7500 { // 5 * 1500
		t.Errorf("expected total minor 7500, got %d", quote.TotalMinor)
	}
	if len(quote.Suppliers) != 1 || len(quote.Suppliers[0].Items) != 1 {
		t.Errorf("expected quoted supplier items: %+v", quote)
	}

	// 4. Delete Item (DELETE)
	resp, body = doJSON(t, envAuth, http.MethodDelete, fmt.Sprintf("/api/v1/commerce/cart/items/%s", lineCode), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on delete item, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &cart)
	if len(cart.Suppliers) != 0 {
		t.Errorf("expected empty cart suppliers after deletion, got %d", len(cart.Suppliers))
	}
}

func TestHTTP_Cart_ValidationAndErrors(t *testing.T) {
	env := setupEnv(t, 0, nopMiddleware, nopMiddleware)
	buyerID := createTestBuyer(t, env)
	envAuth := setupEnv(t, buyerID, withPrincipalMiddleware(buyerID), nopMiddleware)

	// Add nonexistent product -> 404
	resp, body := doJSON(t, envAuth, http.MethodPost, "/api/v1/commerce/cart/items", `{"product_code":"nonexistent","quantity":1}`, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent product, got %d: %s", resp.StatusCode, string(body))
	}
	if code := decodeErr(t, body)["code"]; code != "product_not_found" {
		t.Errorf("expected product_not_found, got %q", code)
	}

	// Add invalid quantity -> 400
	resp, body = doJSON(t, envAuth, http.MethodPost, "/api/v1/commerce/cart/items", `{"product_code":"any","quantity":0}`, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid quantity, got %d: %s", resp.StatusCode, string(body))
	}
	if code := decodeErr(t, body)["code"]; code != "invalid_quantity" {
		t.Errorf("expected invalid_quantity, got %q", code)
	}

	// Delete non-existent line -> 404
	resp, body = doJSON(t, envAuth, http.MethodDelete, "/api/v1/commerce/cart/items/nonexistent", "", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent cart line, got %d: %s", resp.StatusCode, string(body))
	}
	if code := decodeErr(t, body)["code"]; code != "cart_line_not_found" {
		t.Errorf("expected cart_line_not_found, got %q", code)
	}
}
