package router_test

// TASK-014 reports tests: RBAC gating on all nine contract endpoints,
// date/filter validation, aggregation correctness (no join fan-out),
// grouping, promotion attribution, operations/finance totals, empty
// results, boundary windows, pagination caps, and concurrent export
// creation/status reads. Shared advisory lock key: test packages share
// one DB and serialize schema recreation through it.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/reports"
	reports_router "github.com/atlas-platform/backend/internal/modules/reports/router"
	reports_schema "github.com/atlas-platform/backend/internal/modules/reports/schema"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const reportsTestDBLockKey = 1946819412

func TestMain(m *testing.M) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err == nil {
		rawDB, err := sql.Open("pgx", cfg.PostgresDSN())
		if err == nil && rawDB != nil {
			_, _ = rawDB.Exec("SELECT pg_advisory_lock($1)", int64(reportsTestDBLockKey))
			_, _ = rawDB.Exec("DROP SCHEMA IF EXISTS catalog CASCADE; DROP SCHEMA IF EXISTS inventory CASCADE; DROP SCHEMA IF EXISTS identity CASCADE; DROP SCHEMA IF EXISTS pricing CASCADE; DROP SCHEMA IF EXISTS commerce CASCADE; DROP SCHEMA IF EXISTS payments CASCADE; DROP SCHEMA IF EXISTS promotions CASCADE; DROP SCHEMA IF EXISTS crm CASCADE; DROP SCHEMA IF EXISTS suppliers CASCADE; DROP SCHEMA IF EXISTS cms CASCADE; DROP SCHEMA IF EXISTS reports CASCADE; DROP SCHEMA IF EXISTS erp CASCADE; DROP SCHEMA IF EXISTS platform CASCADE; DROP SCHEMA IF EXISTS ai CASCADE; DROP SCHEMA IF EXISTS analytics CASCADE; DROP TABLE IF EXISTS schema_migrations CASCADE; DROP TYPE IF EXISTS catalog_handling_class_type CASCADE;")

			if err := database.RunMigrations(ctx, &cfg.Postgres, "file://../../../../migrations"); err != nil {
				fmt.Printf("TestMain migration error: %v\n", err)
			}

			code := m.Run()
			_, _ = rawDB.Exec("SELECT pg_advisory_unlock($1)", int64(reportsTestDBLockKey))
			rawDB.Close()
			os.Exit(code)
		}
	}
	os.Exit(m.Run())
}

const reportsStaffID int64 = 42

type reportsEnv struct {
	db  *database.DB
	srv *httptest.Server
}

func setupReportsEnv(t *testing.T, staff bool) *reportsEnv {
	t.Helper()
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("config not available: %v", err)
	}
	db, err := database.New(ctx, &cfg.Postgres)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}
	svc := reports.NewService(db)
	rt := reports.New(svc)
	rt.RegisterRoutes(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r.WithContext(reports_router.WithPrincipalID(r.Context(), reportsStaffID)))
			})
		},
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !staff {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"detail":"staff only","code":"forbidden"}`))
					return
				}
				next.ServeHTTP(w, r)
			})
		},
	)

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Mount("/api/v1", rt.ChiRouter())
	server := httptest.NewServer(mux)

	t.Cleanup(func() {
		server.Close()
		db.Close()
	})
	return &reportsEnv{db: db, srv: server}
}

func reportsDo(t *testing.T, env *reportsEnv, method, path, body string) (*http.Response, []byte) {
	t.Helper()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req, err := http.NewRequest(method, env.srv.URL+path, reader)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	return resp, buf.Bytes()
}

var reportsCounter int64

func reportsSuffix() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36) + strconv.FormatInt(atomic.AddInt64(&reportsCounter, 1), 36)
}

func seedExec(t *testing.T, env *reportsEnv, sql string, args ...any) {
	t.Helper()
	if _, err := env.db.Pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("seed failed: %v\nsql: %s", err, sql)
	}
}

func seedID(t *testing.T, env *reportsEnv, sql string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := env.db.Pool.QueryRow(context.Background(), sql, args...).Scan(&id); err != nil {
		t.Fatalf("seed failed: %v\nsql: %s", err, sql)
	}
	return id
}

// resetReportsData empties every table the reports read, so each test runs
// against a known baseline. The suite holds the shared DB exclusively.
func resetReportsData(t *testing.T, env *reportsEnv) {
	t.Helper()
	seedExec(t, env, `
		TRUNCATE TABLE
			reports.export_job,
			payments.payment_refund, payments.payment_attempt, payments.payment_intent, payments.payment_method,
			promotions.voucher_redemption, promotions.promotion,
			commerce.credit_note, commerce.invoice, commerce.shipment_line, commerce.shipment,
			commerce.order_line, commerce."order",
			catalog.product, catalog.supplier, catalog.category, catalog.unit,
			suppliers.supplier_profile,
			identity.buyer_profile, identity.user
		RESTART IDENTITY CASCADE`)

	seedStaffUser(t, env)
}

// seedStaffUser recreates the injected principal as a real identity.user row;
// export jobs carry an FK to identity.user for created_by.
func seedStaffUser(t *testing.T, env *reportsEnv) {
	t.Helper()
	seedExec(t, env, `
		INSERT INTO identity.user (id, code, email, password_hash, user_type)
		VALUES ($1, 'staff_test', 'staff@test.local', 'hash', 'staff')
		ON CONFLICT (id) DO NOTHING
	`, reportsStaffID)
}

func seedBuyerProfile(t *testing.T, env *reportsEnv) int64 {
	t.Helper()
	suffix := reportsSuffix()
	userID := seedID(t, env, `
		INSERT INTO identity.user (code, email, password_hash, user_type)
		VALUES ($1, $2, 'hash', 'buyer') RETURNING id
	`, "usr_"+suffix, "buyer_"+suffix+"@test.local")
	return seedID(t, env, `
		INSERT INTO identity.buyer_profile (user_id, code, business_name)
		VALUES ($1, $2, $3) RETURNING id
	`, userID, "buy_"+suffix, "Buyer "+suffix)
}

func seedCatalogSupplier(t *testing.T, env *reportsEnv, name string) int64 {
	t.Helper()
	suffix := reportsSuffix()
	return seedID(t, env, `
		INSERT INTO catalog.supplier (code, supplier_id, name, slug)
		VALUES ($1, $2, $3, $4) RETURNING id
	`, "sup_"+suffix, atomic.AddInt64(&reportsCounter, 1000), name, "sup-"+suffix)
}

func seedCategory(t *testing.T, env *reportsEnv, name string) int64 {
	t.Helper()
	suffix := reportsSuffix()
	return seedID(t, env, `
		INSERT INTO catalog.category (code, name, slug)
		VALUES ($1, $2, $3) RETURNING id
	`, "cat_"+suffix, name, "cat-"+suffix)
}

func seedUnit(t *testing.T, env *reportsEnv) int64 {
	t.Helper()
	suffix := reportsSuffix()
	return seedID(t, env, `
		INSERT INTO catalog.unit (code, name, symbol, unit_type)
		VALUES ($1, 'Piece', 'pcs', 'base') RETURNING id
	`, "unit_"+suffix)
}

func seedProduct(t *testing.T, env *reportsEnv, categoryID, unitID, supplierID int64, name string) int64 {
	t.Helper()
	suffix := reportsSuffix()
	return seedID(t, env, `
		INSERT INTO catalog.product (code, slug, name, category_id, handling_class, base_unit_id, supplier_id, status, is_active, base_price_minor)
		VALUES ($1, $2, $3, $4, 'ambient', $5, $6, 'published', true, 1500) RETURNING id
	`, "prod_"+suffix, "prod-"+suffix, name, categoryID, unitID, supplierID)
}

func seedOrderRow(t *testing.T, env *reportsEnv, buyerID, supplierID int64, status string, placedAt time.Time, totalMinor int64, onHold bool) (int64, string) {
	t.Helper()
	code := "ord_" + reportsSuffix()
	return seedID(t, env, `
		INSERT INTO commerce."order" (code, buyer_id, supplier_id, currency, subtotal_minor, discounts_minor, total_minor, status, placed_at, on_hold)
		VALUES ($1, $2, $3, 'USD', $4, 0, $4, $5, $6, $7) RETURNING id
	`, code, buyerID, supplierID, totalMinor, status, placedAt, onHold), code
}

func seedOrderLine(t *testing.T, env *reportsEnv, orderID, productID, supplierID, unitID int64, qty, unitPriceMinor int64, productCode, productName string) {
	t.Helper()
	seedExec(t, env, `
		INSERT INTO commerce.order_line (code, order_id, product_id, supplier_id, unit_id, quantity, unit_price_minor, total_price_minor, currency, product_code, product_name, unit_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'USD', $9, $10, 'pcs')
	`, "lin_"+reportsSuffix(), orderID, productID, supplierID, unitID, qty, unitPriceMinor, qty*unitPriceMinor, productCode, productName)
}

func seedInvoice(t *testing.T, env *reportsEnv, orderID int64, status string, totalMinor, balanceMinor int64, currency string, issuedAt *time.Time) {
	t.Helper()
	seedExec(t, env, `
		INSERT INTO commerce.invoice (code, order_id, subtotal_minor, total_minor, balance_minor, currency, status, issued_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, "inv_"+reportsSuffix(), orderID, totalMinor, totalMinor, balanceMinor, currency, status, issuedAt)
}

func seedCreditNote(t *testing.T, env *reportsEnv, orderID int64, amountMinor int64, status string, createdAt time.Time) {
	t.Helper()
	seedExec(t, env, `
		INSERT INTO commerce.credit_note (code, batch_code, order_id, amount_minor, currency, reason, status, created_at)
		VALUES ($1, $2, $3, $4, 'USD', 'return', $5, $6)
	`, "cn_"+reportsSuffix(), "bat_"+reportsSuffix(), orderID, amountMinor, status, createdAt)
}

func seedShipment(t *testing.T, env *reportsEnv, orderID int64, status string, createdAt time.Time) {
	t.Helper()
	seedExec(t, env, `
		INSERT INTO commerce.shipment (code, order_id, status, created_at)
		VALUES ($1, $2, $3, $4)
	`, "shp_"+reportsSuffix(), orderID, status, createdAt)
}

func seedPromotion(t *testing.T, env *reportsEnv, name string) int64 {
	t.Helper()
	return seedID(t, env, `
		INSERT INTO promotions.promotion (code, name, kind, value_minor, currency, status)
		VALUES ($1, $2, 'fixed', 100, 'USD', 'published') RETURNING id
	`, "promo_"+reportsSuffix(), name)
}

func seedRedemption(t *testing.T, env *reportsEnv, promotionID, buyerID, orderID int64, amountMinor int64, createdAt time.Time) {
	t.Helper()
	seedExec(t, env, `
		INSERT INTO promotions.voucher_redemption (promotion_id, buyer_id, order_id, amount_minor, currency, created_at)
		VALUES ($1, $2, $3, $4, 'USD', $5)
	`, promotionID, buyerID, orderID, amountMinor, createdAt)
}

func seedIntent(t *testing.T, env *reportsEnv, buyerID, orderID int64) int64 {
	t.Helper()
	suffix := reportsSuffix()
	return seedID(t, env, `
		INSERT INTO payments.payment_intent (code, buyer_id, order_id, order_code, amount_minor, currency, status, idem_key)
		VALUES ($1, $2, $3, $4, 100, 'USD', 'succeeded', $5) RETURNING id
	`, "int_"+suffix, buyerID, orderID, "ord_"+suffix, "idem_"+suffix)
}

func seedRefund(t *testing.T, env *reportsEnv, intentID, amountMinor int64, status string, createdAt time.Time) {
	t.Helper()
	seedExec(t, env, `
		INSERT INTO payments.payment_refund (code, intent_id, amount_minor, currency, reason, status, created_at)
		VALUES ($1, $2, $3, 'USD', 'buyer request', $4, $5)
	`, "rf_"+reportsSuffix(), intentID, amountMinor, status, createdAt)
}

func decodeBody[T any](t *testing.T, body []byte) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode body: %v\nbody: %s", err, string(body))
	}
	return out
}

type pageEnvelope[T any] struct {
	Items    []T  `json:"items"`
	Page     int  `json:"page"`
	PageSize int  `json:"page_size"`
	Total    int  `json:"total"`
	HasNext  bool `json:"has_next"`
}

func assertStatus(t *testing.T, resp *http.Response, body []byte, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("expected %d, got %d: %s", want, resp.StatusCode, string(body))
	}
}

func TestReports_RBAC_DeniesNonStaff(t *testing.T) {
	env := setupReportsEnv(t, false)

	getPaths := []string{
		"/api/v1/admin/reports/sales?group_by=period",
		"/api/v1/admin/reports/buyers",
		"/api/v1/admin/reports/products",
		"/api/v1/admin/reports/suppliers",
		"/api/v1/admin/reports/promotions",
		"/api/v1/admin/reports/operations",
		"/api/v1/admin/reports/finance",
		"/api/v1/admin/reports/exports",
	}
	for _, p := range getPaths {
		resp, body := reportsDo(t, env, http.MethodGet, p, "")
		assertStatus(t, resp, body, http.StatusForbidden)
		if errObj := decodeBody[map[string]string](t, body); errObj["code"] != "forbidden" {
			t.Fatalf("%s: expected forbidden code, got %v", p, errObj)
		}
	}
	resp, body := reportsDo(t, env, http.MethodPost, "/api/v1/admin/reports/exports", `{"report_type":"sales"}`)
	assertStatus(t, resp, body, http.StatusForbidden)
}

func TestReports_RBAC_AllowsStaff(t *testing.T) {
	env := setupReportsEnv(t, true)
	resetReportsData(t, env)
	seedStaffUser(t, env)

	getPaths := []string{
		"/api/v1/admin/reports/sales?group_by=period",
		"/api/v1/admin/reports/buyers",
		"/api/v1/admin/reports/products",
		"/api/v1/admin/reports/suppliers",
		"/api/v1/admin/reports/promotions",
		"/api/v1/admin/reports/operations",
		"/api/v1/admin/reports/finance",
		"/api/v1/admin/reports/exports",
	}
	for _, p := range getPaths {
		resp, body := reportsDo(t, env, http.MethodGet, p, "")
		assertStatus(t, resp, body, http.StatusOK)
	}
	resp, body := reportsDo(t, env, http.MethodPost, "/api/v1/admin/reports/exports", `{"report_type":"sales"}`)
	assertStatus(t, resp, body, http.StatusAccepted)
}

func TestReports_SalesGrouping(t *testing.T) {
	env := setupReportsEnv(t, true)
	resetReportsData(t, env)

	buyerID := seedBuyerProfile(t, env)
	s1 := seedCatalogSupplier(t, env, "Supplier One")
	s2 := seedCatalogSupplier(t, env, "Supplier Two")
	c1 := seedCategory(t, env, "Drinks")
	c2 := seedCategory(t, env, "Snacks")
	unitID := seedUnit(t, env)
	p1 := seedProduct(t, env, c1, unitID, s1, "Cola")
	p2 := seedProduct(t, env, c2, unitID, s1, "Chips")

	jan := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	feb := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)

	// O1 spans two categories and two lines; a line join that also sums the
	// order total would report 2000 for January instead of 1000.
	o1, _ := seedOrderRow(t, env, buyerID, s1, "delivered", jan, 1000, false)
	seedOrderLine(t, env, o1, p1, s1, unitID, 2, 200, "prod_cola", "Cola")
	seedOrderLine(t, env, o1, p2, s1, unitID, 3, 200, "prod_chips", "Chips")
	o2, _ := seedOrderRow(t, env, buyerID, s2, "shipped", jan, 500, false)
	seedOrderLine(t, env, o2, p1, s2, unitID, 1, 500, "prod_cola", "Cola")
	o3, _ := seedOrderRow(t, env, buyerID, s1, "confirmed", feb, 300, false)
	seedOrderLine(t, env, o3, p1, s1, unitID, 1, 300, "prod_cola", "Cola")

	resp, body := reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/sales?group_by=period", "")
	assertStatus(t, resp, body, http.StatusOK)
	page := decodeBody[pageEnvelope[reports_schema.SalesReportItem]](t, body)
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("period groups: want 2, got total=%d items=%d", page.Total, len(page.Items))
	}
	if page.Items[0].Dimension != "2026-01" || page.Items[0].OrderCount != 2 || page.Items[0].RevenueMinor != 1500 {
		t.Fatalf("jan period wrong: %+v", page.Items[0])
	}
	if page.Items[1].Dimension != "2026-02" || page.Items[1].OrderCount != 1 || page.Items[1].RevenueMinor != 300 {
		t.Fatalf("feb period wrong: %+v", page.Items[1])
	}
	if page.Items[0].Currency != "USD" {
		t.Fatalf("currency not preserved: %+v", page.Items[0])
	}

	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/sales?group_by=supplier", "")
	assertStatus(t, resp, body, http.StatusOK)
	bySupplier := decodeBody[pageEnvelope[reports_schema.SalesReportItem]](t, body)
	if len(bySupplier.Items) != 2 || bySupplier.Items[0].Dimension != "Supplier One" ||
		bySupplier.Items[0].OrderCount != 2 || bySupplier.Items[0].RevenueMinor != 1300 {
		t.Fatalf("supplier grouping wrong: %+v", bySupplier.Items)
	}
	if bySupplier.Items[1].Dimension != "Supplier Two" || bySupplier.Items[1].RevenueMinor != 500 {
		t.Fatalf("supplier grouping wrong: %+v", bySupplier.Items[1])
	}

	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/sales?group_by=category", "")
	assertStatus(t, resp, body, http.StatusOK)
	byCategory := decodeBody[pageEnvelope[reports_schema.SalesReportItem]](t, body)
	if len(byCategory.Items) != 2 {
		t.Fatalf("category groups: want 2, got %+v", byCategory.Items)
	}
	if byCategory.Items[0].Dimension != "Drinks" || byCategory.Items[0].RevenueMinor != 1200 || byCategory.Items[0].OrderCount != 3 {
		t.Fatalf("drinks grouping wrong: %+v", byCategory.Items[0])
	}
	if byCategory.Items[1].Dimension != "Snacks" || byCategory.Items[1].RevenueMinor != 600 || byCategory.Items[1].OrderCount != 1 {
		t.Fatalf("snacks grouping wrong: %+v", byCategory.Items[1])
	}

	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/sales?group_by=bogus", "")
	assertStatus(t, resp, body, http.StatusBadRequest)
}

func TestReports_BuyerProductSupplierGrouping(t *testing.T) {
	env := setupReportsEnv(t, true)
	resetReportsData(t, env)

	b1 := seedBuyerProfile(t, env)
	_ = seedBuyerProfile(t, env) // zero-order buyer: acquisition view keeps it
	b3 := seedBuyerProfile(t, env)
	s1 := seedCatalogSupplier(t, env, "Alpha Supply")
	s2 := seedCatalogSupplier(t, env, "Beta Supply")
	c1 := seedCategory(t, env, "General")
	unitID := seedUnit(t, env)
	p1 := seedProduct(t, env, c1, unitID, s1, "Widget")
	p2 := seedProduct(t, env, c1, unitID, s1, "Gadget")

	jan := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	feb := time.Date(2026, 2, 10, 10, 0, 0, 0, time.UTC)

	o1, _ := seedOrderRow(t, env, b1, s1, "delivered", jan, 1000, false)
	seedOrderLine(t, env, o1, p1, s1, unitID, 2, 200, "prod_widget", "Widget")
	seedOrderLine(t, env, o1, p2, s1, unitID, 3, 200, "prod_gadget", "Gadget")
	o2, _ := seedOrderRow(t, env, b3, s2, "cancelled", jan, 500, false)
	seedOrderLine(t, env, o2, p1, s2, unitID, 1, 500, "prod_widget", "Widget")
	o3, _ := seedOrderRow(t, env, b1, s1, "shipped", feb, 300, false)
	seedOrderLine(t, env, o3, p1, s1, unitID, 1, 300, "prod_widget", "Widget")

	resp, body := reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/buyers", "")
	assertStatus(t, resp, body, http.StatusOK)
	buyers := decodeBody[pageEnvelope[reports_schema.BuyerReportItem]](t, body)
	if buyers.Total != 3 || len(buyers.Items) != 3 {
		t.Fatalf("buyer rows: want 3, got %d", len(buyers.Items))
	}
	if buyers.Items[0].OrderCount != 2 || buyers.Items[0].TotalSpendMinor != 1300 {
		t.Fatalf("top buyer wrong: %+v", buyers.Items[0])
	}
	if buyers.Items[0].LastOrderAt == nil || !buyers.Items[0].LastOrderAt.Equal(feb) {
		t.Fatalf("last_order_at wrong: %+v", buyers.Items[0])
	}
	if buyers.Items[1].OrderCount != 1 || buyers.Items[1].TotalSpendMinor != 500 {
		t.Fatalf("second buyer wrong: %+v", buyers.Items[1])
	}
	if buyers.Items[2].OrderCount != 0 || buyers.Items[2].TotalSpendMinor != 0 || buyers.Items[2].LastOrderAt != nil {
		t.Fatalf("zero-order buyer wrong: %+v", buyers.Items[2])
	}

	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/products", "")
	assertStatus(t, resp, body, http.StatusOK)
	products := decodeBody[pageEnvelope[reports_schema.ProductReportItem]](t, body)
	if len(products.Items) != 2 {
		t.Fatalf("product rows: want 2, got %+v", products.Items)
	}
	if products.Items[0].ProductName != "Widget" || products.Items[0].QuantitySold != 4 || products.Items[0].RevenueMinor != 1200 || products.Items[0].OrderCount != 3 {
		t.Fatalf("widget row wrong: %+v", products.Items[0])
	}
	if products.Items[1].ProductName != "Gadget" || products.Items[1].QuantitySold != 3 || products.Items[1].RevenueMinor != 600 {
		t.Fatalf("gadget row wrong: %+v", products.Items[1])
	}

	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/suppliers", "")
	assertStatus(t, resp, body, http.StatusOK)
	suppliers := decodeBody[pageEnvelope[reports_schema.SupplierReportItem]](t, body)
	if len(suppliers.Items) != 2 {
		t.Fatalf("supplier rows: want 2, got %+v", suppliers.Items)
	}
	if suppliers.Items[0].SupplierName != "Alpha Supply" || suppliers.Items[0].OrderCount != 2 ||
		suppliers.Items[0].FulfilledCount != 2 || suppliers.Items[0].CancelledCount != 0 || suppliers.Items[0].TotalMinor != 1300 {
		t.Fatalf("alpha row wrong: %+v", suppliers.Items[0])
	}
	if suppliers.Items[1].SupplierName != "Beta Supply" || suppliers.Items[1].CancelledCount != 1 || suppliers.Items[1].FulfilledCount != 0 {
		t.Fatalf("beta row wrong: %+v", suppliers.Items[1])
	}
}

func TestReports_PromotionAttribution(t *testing.T) {
	env := setupReportsEnv(t, true)
	resetReportsData(t, env)

	buyerID := seedBuyerProfile(t, env)
	s1 := seedCatalogSupplier(t, env, "Promo Supply")
	jan := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	o1, _ := seedOrderRow(t, env, buyerID, s1, "placed", jan, 1000, false)
	o2, _ := seedOrderRow(t, env, buyerID, s1, "placed", jan, 1000, false)
	o3, _ := seedOrderRow(t, env, buyerID, s1, "placed", jan.AddDate(0, 1, 0), 1000, false)

	promoA := seedPromotion(t, env, "Launch Deal")
	_ = seedPromotion(t, env, "Unused Deal")
	seedRedemption(t, env, promoA, buyerID, o1, 300, jan)
	seedRedemption(t, env, promoA, buyerID, o2, 200, jan)
	seedRedemption(t, env, promoA, buyerID, o3, 150, jan.AddDate(0, 1, 0))

	resp, body := reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/promotions", "")
	assertStatus(t, resp, body, http.StatusOK)
	all := decodeBody[pageEnvelope[reports_schema.PromotionReportItem]](t, body)
	if all.Total != 2 || len(all.Items) != 2 {
		t.Fatalf("promotion rows: want 2, got %+v", all.Items)
	}
	if all.Items[0].PromotionName != "Launch Deal" || all.Items[0].Redemptions != 3 || all.Items[0].DiscountMinor != 650 {
		t.Fatalf("launch deal wrong: %+v", all.Items[0])
	}
	if all.Items[1].PromotionName != "Unused Deal" || all.Items[1].Redemptions != 0 || all.Items[1].DiscountMinor != 0 {
		t.Fatalf("unused deal wrong: %+v", all.Items[1])
	}

	windowFrom := jan.Format(time.RFC3339)
	windowTo := jan.AddDate(0, 1, 0).Format(time.RFC3339)
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/promotions?from="+windowFrom+"&to="+windowTo, "")
	assertStatus(t, resp, body, http.StatusOK)
	windowed := decodeBody[pageEnvelope[reports_schema.PromotionReportItem]](t, body)
	if windowed.Items[0].Redemptions != 2 || windowed.Items[0].DiscountMinor != 500 {
		t.Fatalf("windowed attribution wrong: %+v", windowed.Items[0])
	}
}

func TestReports_OperationsStates(t *testing.T) {
	env := setupReportsEnv(t, true)
	resetReportsData(t, env)

	buyerID := seedBuyerProfile(t, env)
	s1 := seedCatalogSupplier(t, env, "Ops Supply")
	jan := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	o1, _ := seedOrderRow(t, env, buyerID, s1, "placed", jan, 100, true)
	_, _ = seedOrderRow(t, env, buyerID, s1, "placed", jan, 100, false)
	_, _ = seedOrderRow(t, env, buyerID, s1, "confirmed", jan, 100, false)
	_, _ = seedOrderRow(t, env, buyerID, s1, "processing", jan, 100, false)
	_, _ = seedOrderRow(t, env, buyerID, s1, "shipped", jan, 100, false)
	_, _ = seedOrderRow(t, env, buyerID, s1, "delivered", jan, 100, false)
	_, _ = seedOrderRow(t, env, buyerID, s1, "cancelled", jan, 100, false)
	_, _ = seedOrderRow(t, env, buyerID, s1, "placed", jan.AddDate(-1, 0, 0), 100, false)

	seedShipment(t, env, o1, "preparing", jan)
	seedShipment(t, env, o1, "shipped", jan)
	seedShipment(t, env, o1, "shipped", jan)
	seedShipment(t, env, o1, "delivered", jan)
	seedShipment(t, env, o1, "cancelled", jan)
	seedShipment(t, env, o1, "preparing", jan.AddDate(-1, 0, 0))

	windowFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	windowTo := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	resp, body := reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/operations?from="+windowFrom+"&to="+windowTo, "")
	assertStatus(t, resp, body, http.StatusOK)
	ops := decodeBody[reports_schema.OperationsReportResponse](t, body)

	if ops.Orders.Total != 7 {
		t.Fatalf("orders total: want 7, got %d", ops.Orders.Total)
	}
	if ops.Orders.Placed != 2 || ops.Orders.Confirmed != 1 || ops.Orders.Processing != 1 ||
		ops.Orders.Shipped != 1 || ops.Orders.Delivered != 1 || ops.Orders.Cancelled != 1 {
		t.Fatalf("order status counts wrong: %+v", ops.Orders)
	}
	if ops.Orders.OnHold != 1 {
		t.Fatalf("on hold count: want 1, got %d", ops.Orders.OnHold)
	}
	if ops.Shipments.Total != 5 || ops.Shipments.Preparing != 1 || ops.Shipments.Shipped != 2 ||
		ops.Shipments.Delivered != 1 || ops.Shipments.Cancelled != 1 {
		t.Fatalf("shipment counts wrong: %+v", ops.Shipments)
	}
}

func TestReports_FinanceTotals(t *testing.T) {
	env := setupReportsEnv(t, true)
	resetReportsData(t, env)

	buyerID := seedBuyerProfile(t, env)
	s1 := seedCatalogSupplier(t, env, "Finance Supply")
	jan := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	dec := time.Date(2025, 12, 15, 10, 0, 0, 0, time.UTC)
	o1, _ := seedOrderRow(t, env, buyerID, s1, "delivered", jan, 1000, false)

	seedInvoice(t, env, o1, "issued", 1000, 400, "USD", &jan)
	seedInvoice(t, env, o1, "issued", 800, 800, "EUR", &jan)
	seedInvoice(t, env, o1, "draft", 5000, 5000, "USD", nil)
	seedInvoice(t, env, o1, "issued", 700, 700, "USD", &dec)

	seedCreditNote(t, env, o1, 100, "applied", jan)
	seedCreditNote(t, env, o1, 50, "void", jan)
	seedCreditNote(t, env, o1, 70, "applied", dec)

	intentID := seedIntent(t, env, buyerID, o1)
	seedRefund(t, env, intentID, 200, "approved", jan)
	seedRefund(t, env, intentID, 300, "pending_approval", jan)
	seedRefund(t, env, intentID, 40, "applied", dec)

	windowFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	windowTo := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	resp, body := reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/finance?from="+windowFrom+"&to="+windowTo, "")
	assertStatus(t, resp, body, http.StatusOK)
	fin := decodeBody[reports_schema.FinanceReportResponse](t, body)

	if len(fin.Items) != 2 {
		t.Fatalf("finance currencies: want 2, got %+v", fin.Items)
	}
	eur, usd := fin.Items[0], fin.Items[1]
	if eur.Currency != "EUR" || eur.TotalInvoicedMinor != 800 || eur.TotalCollectedMinor != 0 || eur.OutstandingBalanceMinor != 800 {
		t.Fatalf("eur row wrong: %+v", eur)
	}
	if usd.Currency != "USD" || usd.TotalInvoicedMinor != 1000 || usd.TotalCollectedMinor != 600 ||
		usd.OutstandingBalanceMinor != 400 || usd.CreditsAppliedMinor != 100 || usd.RefundsMinor != 200 {
		t.Fatalf("usd row wrong: %+v", usd)
	}
}

func TestReports_DateWindowsAndBoundaries(t *testing.T) {
	env := setupReportsEnv(t, true)
	resetReportsData(t, env)

	buyerID := seedBuyerProfile(t, env)
	s1 := seedCatalogSupplier(t, env, "Window Supply")

	t1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	march1 := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	march2 := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	_, _ = seedOrderRow(t, env, buyerID, s1, "placed", t1, 100, false)
	_, _ = seedOrderRow(t, env, buyerID, s1, "placed", march1, 200, false)
	_, _ = seedOrderRow(t, env, buyerID, s1, "placed", march2, 400, false)

	// RFC 3339 window is half-open [from, to): the boundary order at `to` is out.
	resp, body := reportsDo(t, env, http.MethodGet,
		"/api/v1/admin/reports/sales?group_by=period&from=2026-03-01T00:00:00Z&to=2026-03-02T00:00:00Z", "")
	assertStatus(t, resp, body, http.StatusOK)
	tsWindow := decodeBody[pageEnvelope[reports_schema.SalesReportItem]](t, body)
	if tsWindow.Items[0].OrderCount != 2 || tsWindow.Items[0].RevenueMinor != 300 {
		t.Fatalf("timestamp window wrong: %+v", tsWindow.Items[0])
	}

	// Date-only bounds cover the whole named day: `to=2026-03-01` includes
	// 23:59:59 of that day and still excludes the next midnight.
	resp, body = reportsDo(t, env, http.MethodGet,
		"/api/v1/admin/reports/sales?group_by=period&from=2026-03-01&to=2026-03-01", "")
	assertStatus(t, resp, body, http.StatusOK)
	dayWindow := decodeBody[pageEnvelope[reports_schema.SalesReportItem]](t, body)
	if dayWindow.Items[0].OrderCount != 2 || dayWindow.Items[0].RevenueMinor != 300 {
		t.Fatalf("date-only window wrong: %+v", dayWindow.Items[0])
	}

	badQueries := []string{
		"from=not-a-date",
		"to=2026-13-45",
		"from=2026-03-02&to=2026-03-01",
		"from=2026-03-01T00:00:00Z&to=2026-03-01T00:00:00Z",
	}
	for _, q := range badQueries {
		resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/sales?"+q, "")
		assertStatus(t, resp, body, http.StatusBadRequest)
	}

	// Validation is shared across every report endpoint.
	reportPaths := []string{"sales", "buyers", "products", "suppliers", "promotions", "operations", "finance"}
	for _, p := range reportPaths {
		resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/"+p+"?from=bad", "")
		assertStatus(t, resp, body, http.StatusBadRequest)
	}

	// A window with no matching rows is empty, not an error.
	resp, body = reportsDo(t, env, http.MethodGet,
		"/api/v1/admin/reports/sales?group_by=period&from=2027-01-01&to=2027-02-01", "")
	assertStatus(t, resp, body, http.StatusOK)
	empty := decodeBody[pageEnvelope[reports_schema.SalesReportItem]](t, body)
	if empty.Total != 0 || len(empty.Items) != 0 || empty.HasNext {
		t.Fatalf("empty window wrong: %+v", empty)
	}
}

func TestReports_EmptyResults(t *testing.T) {
	env := setupReportsEnv(t, true)
	resetReportsData(t, env)

	resp, body := reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/sales", "")
	assertStatus(t, resp, body, http.StatusOK)
	if sales := decodeBody[pageEnvelope[reports_schema.SalesReportItem]](t, body); sales.Total != 0 || len(sales.Items) != 0 {
		t.Fatalf("empty sales wrong: %+v", sales)
	}
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/buyers", "")
	assertStatus(t, resp, body, http.StatusOK)
	if buyers := decodeBody[pageEnvelope[reports_schema.BuyerReportItem]](t, body); buyers.Total != 0 || len(buyers.Items) != 0 {
		t.Fatalf("empty buyers wrong: %+v", buyers)
	}
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/products", "")
	assertStatus(t, resp, body, http.StatusOK)
	if products := decodeBody[pageEnvelope[reports_schema.ProductReportItem]](t, body); products.Total != 0 || len(products.Items) != 0 {
		t.Fatalf("empty products wrong: %+v", products)
	}
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/suppliers", "")
	assertStatus(t, resp, body, http.StatusOK)
	if suppliers := decodeBody[pageEnvelope[reports_schema.SupplierReportItem]](t, body); suppliers.Total != 0 || len(suppliers.Items) != 0 {
		t.Fatalf("empty suppliers wrong: %+v", suppliers)
	}
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/promotions", "")
	assertStatus(t, resp, body, http.StatusOK)
	if promos := decodeBody[pageEnvelope[reports_schema.PromotionReportItem]](t, body); promos.Total != 0 || len(promos.Items) != 0 {
		t.Fatalf("empty promotions wrong: %+v", promos)
	}
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/operations", "")
	assertStatus(t, resp, body, http.StatusOK)
	if ops := decodeBody[reports_schema.OperationsReportResponse](t, body); ops.Orders.Total != 0 || ops.Shipments.Total != 0 {
		t.Fatalf("empty operations wrong: %+v", ops)
	}
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/finance", "")
	assertStatus(t, resp, body, http.StatusOK)
	if fin := decodeBody[reports_schema.FinanceReportResponse](t, body); len(fin.Items) != 0 {
		t.Fatalf("empty finance wrong: %+v", fin)
	}
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/exports", "")
	assertStatus(t, resp, body, http.StatusOK)
	if jobs := decodeBody[pageEnvelope[reports_schema.ExportJobResponse]](t, body); jobs.Total != 0 || len(jobs.Items) != 0 {
		t.Fatalf("empty exports wrong: %+v", jobs)
	}
}

func TestReports_Pagination(t *testing.T) {
	env := setupReportsEnv(t, true)
	resetReportsData(t, env)
	seedStaffUser(t, env)

	for i := 0; i < 25; i++ {
		_ = seedBuyerProfile(t, env)
	}

	resp, body := reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/buyers?page=1&page_size=10", "")
	assertStatus(t, resp, body, http.StatusOK)
	p1 := decodeBody[pageEnvelope[reports_schema.BuyerReportItem]](t, body)
	if len(p1.Items) != 10 || p1.Total != 25 || !p1.HasNext || p1.Page != 1 || p1.PageSize != 10 {
		t.Fatalf("page 1 wrong: %+v", p1)
	}

	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/buyers?page=3&page_size=10", "")
	assertStatus(t, resp, body, http.StatusOK)
	p3 := decodeBody[pageEnvelope[reports_schema.BuyerReportItem]](t, body)
	if len(p3.Items) != 5 || p3.HasNext {
		t.Fatalf("page 3 wrong: %+v", p3)
	}

	// page_size is capped at 200 by the schema, not by the caller.
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/buyers?page_size=1000", "")
	assertStatus(t, resp, body, http.StatusOK)
	capped := decodeBody[pageEnvelope[reports_schema.BuyerReportItem]](t, body)
	if capped.PageSize != 200 {
		t.Fatalf("page_size cap wrong: %+v", capped)
	}

	// Invalid page values fall back to defaults rather than erroring.
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/buyers?page=0&page_size=-1", "")
	assertStatus(t, resp, body, http.StatusOK)
	if def := decodeBody[pageEnvelope[reports_schema.BuyerReportItem]](t, body); def.Page != 1 || def.PageSize != 20 {
		t.Fatalf("defaults wrong: %+v", def)
	}

	// Exports list paginates the same way.
	for i := 0; i < 3; i++ {
		resp, body = reportsDo(t, env, http.MethodPost, "/api/v1/admin/reports/exports", `{"report_type":"sales"}`)
		assertStatus(t, resp, body, http.StatusAccepted)
	}
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/exports?page=1&page_size=2", "")
	assertStatus(t, resp, body, http.StatusOK)
	e1 := decodeBody[pageEnvelope[reports_schema.ExportJobResponse]](t, body)
	if len(e1.Items) != 2 || e1.Total != 3 || !e1.HasNext {
		t.Fatalf("exports page 1 wrong: %+v", e1)
	}
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/exports?page=2&page_size=2", "")
	assertStatus(t, resp, body, http.StatusOK)
	e2 := decodeBody[pageEnvelope[reports_schema.ExportJobResponse]](t, body)
	if len(e2.Items) != 1 || e2.HasNext {
		t.Fatalf("exports page 2 wrong: %+v", e2)
	}
}

func TestReports_ExportLifecycleValidation(t *testing.T) {
	env := setupReportsEnv(t, true)
	resetReportsData(t, env)
	seedStaffUser(t, env)

	resp, body := reportsDo(t, env, http.MethodPost, "/api/v1/admin/reports/exports", `{"report_type":"sales"}`)
	assertStatus(t, resp, body, http.StatusAccepted)
	job := decodeBody[reports_schema.ExportJobResponse](t, body)
	if job.Code == "" || job.ReportType != "sales" || job.Status != "pending" || job.Format != "csv" {
		t.Fatalf("job shape wrong: %+v", job)
	}
	if job.DownloadUrl != nil {
		t.Fatalf("download_url must stay null until generation exists: %+v", *job.DownloadUrl)
	}
	if job.StatusUrl != "/api/v1/admin/reports/exports" {
		t.Fatalf("status_url wrong: %q", job.StatusUrl)
	}

	resp, body = reportsDo(t, env, http.MethodPost, "/api/v1/admin/reports/exports",
		`{"report_type":"finance","format":"csv","from":"2026-01-01","to":"2026-02-01"}`)
	assertStatus(t, resp, body, http.StatusAccepted)

	badBodies := []string{
		`{"report_type":"bogus"}`,
		`{"report_type":""}`,
		`{"report_type":"sales","format":"xlsx"}`,
		`{"report_type":"sales","from":"not-a-date"}`,
		`{"report_type":"sales","from":"2026-02-01","to":"2026-01-01"}`,
		`not json`,
	}
	for _, b := range badBodies {
		resp, body = reportsDo(t, env, http.MethodPost, "/api/v1/admin/reports/exports", b)
		assertStatus(t, resp, body, http.StatusBadRequest)
	}

	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/exports", "")
	assertStatus(t, resp, body, http.StatusOK)
	if all := decodeBody[pageEnvelope[reports_schema.ExportJobResponse]](t, body); all.Total != 2 {
		t.Fatalf("export list total wrong: %+v", all)
	}
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/exports?status=pending", "")
	assertStatus(t, resp, body, http.StatusOK)
	if pending := decodeBody[pageEnvelope[reports_schema.ExportJobResponse]](t, body); pending.Total != 2 {
		t.Fatalf("pending filter wrong: %+v", pending)
	}
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/exports?status=completed", "")
	assertStatus(t, resp, body, http.StatusOK)
	if completed := decodeBody[pageEnvelope[reports_schema.ExportJobResponse]](t, body); completed.Total != 0 {
		t.Fatalf("completed filter wrong: %+v", completed)
	}
	resp, body = reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/exports?status=bogus", "")
	assertStatus(t, resp, body, http.StatusBadRequest)
}

func TestReports_ExportConcurrency(t *testing.T) {
	env := setupReportsEnv(t, true)
	resetReportsData(t, env)
	seedStaffUser(t, env)

	const creates = 8
	const readers = 4
	var wg sync.WaitGroup
	var mu sync.Mutex
	codes := map[string]bool{}
	failures := []string{}

	for i := 0; i < creates; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, body := reportsDo(t, env, http.MethodPost, "/api/v1/admin/reports/exports", `{"report_type":"sales"}`)
			mu.Lock()
			defer mu.Unlock()
			if resp.StatusCode != http.StatusAccepted {
				failures = append(failures, fmt.Sprintf("create status %d: %s", resp.StatusCode, string(body)))
				return
			}
			job := decodeBody[reports_schema.ExportJobResponse](t, body)
			if codes[job.Code] {
				failures = append(failures, "duplicate export code "+job.Code)
			}
			codes[job.Code] = true
		}()
	}
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, body := reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/exports", "")
			mu.Lock()
			defer mu.Unlock()
			if resp.StatusCode != http.StatusOK {
				failures = append(failures, fmt.Sprintf("read status %d: %s", resp.StatusCode, string(body)))
			}
		}()
	}
	wg.Wait()

	if len(failures) > 0 {
		t.Fatalf("concurrent export failures: %v", failures)
	}
	if len(codes) != creates {
		t.Fatalf("expected %d distinct codes, got %d", creates, len(codes))
	}

	resp, body := reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/exports", "")
	assertStatus(t, resp, body, http.StatusOK)
	if jobs := decodeBody[pageEnvelope[reports_schema.ExportJobResponse]](t, body); jobs.Total != creates {
		t.Fatalf("export count after concurrency: want %d, got %+v", creates, jobs)
	}
}

func TestReports_ReadOnlyProjection(t *testing.T) {
	env := setupReportsEnv(t, true)
	resetReportsData(t, env)

	buyerID := seedBuyerProfile(t, env)
	s1 := seedCatalogSupplier(t, env, "Readonly Supply")
	jan := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	orderID, _ := seedOrderRow(t, env, buyerID, s1, "delivered", jan, 1000, false)

	var before time.Time
	if err := env.db.Pool.QueryRow(context.Background(),
		`SELECT updated_at FROM commerce."order" WHERE id = $1`, orderID).Scan(&before); err != nil {
		t.Fatalf("read order timestamp: %v", err)
	}

	paths := []string{"sales", "buyers", "products", "suppliers", "promotions", "operations", "finance", "exports"}
	for _, p := range paths {
		resp, body := reportsDo(t, env, http.MethodGet, "/api/v1/admin/reports/"+p, "")
		assertStatus(t, resp, body, http.StatusOK)
	}

	var after time.Time
	if err := env.db.Pool.QueryRow(context.Background(),
		`SELECT updated_at FROM commerce."order" WHERE id = $1`, orderID).Scan(&after); err != nil {
		t.Fatalf("re-read order timestamp: %v", err)
	}
	if !before.Equal(after) {
		t.Fatalf("report endpoints mutated transactional data: %v -> %v", before, after)
	}
}
