package router_test

// TASK-011 suppliers tests: application (anon/auth-linked), approval with
// user linkage, versioned contracts, self-service reads, RBAC, races.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/atlas-platform/backend/internal/database"
	catalog_repo "github.com/atlas-platform/backend/internal/modules/catalog/repository"
	commerce_repo "github.com/atlas-platform/backend/internal/modules/commerce/repository"
	commerce_service "github.com/atlas-platform/backend/internal/modules/commerce/service"
	inventory "github.com/atlas-platform/backend/internal/modules/inventory"
	pricing_service "github.com/atlas-platform/backend/internal/modules/pricing/service"
	promotions "github.com/atlas-platform/backend/internal/modules/promotions"
	"github.com/atlas-platform/backend/internal/modules/suppliers"
	suppliers_router "github.com/atlas-platform/backend/internal/modules/suppliers/router"
	suppliers_schema "github.com/atlas-platform/backend/internal/modules/suppliers/schema"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const suppliersTestDBLockKey = 1946819412

func TestMain(m *testing.M) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err == nil {
		rawDB, err := sql.Open("pgx", cfg.PostgresDSN())
		if err == nil && rawDB != nil {
			_, _ = rawDB.Exec("SELECT pg_advisory_lock($1)", int64(suppliersTestDBLockKey))
			_, _ = rawDB.Exec("DROP SCHEMA IF EXISTS catalog CASCADE; DROP SCHEMA IF EXISTS inventory CASCADE; DROP SCHEMA IF EXISTS identity CASCADE; DROP SCHEMA IF EXISTS pricing CASCADE; DROP SCHEMA IF EXISTS commerce CASCADE; DROP SCHEMA IF EXISTS payments CASCADE; DROP SCHEMA IF EXISTS promotions CASCADE; DROP SCHEMA IF EXISTS crm CASCADE; DROP SCHEMA IF EXISTS suppliers CASCADE; DROP SCHEMA IF EXISTS cms CASCADE; DROP SCHEMA IF EXISTS reports CASCADE; DROP SCHEMA IF EXISTS erp CASCADE; DROP SCHEMA IF EXISTS platform CASCADE; DROP SCHEMA IF EXISTS ai CASCADE; DROP SCHEMA IF EXISTS analytics CASCADE; DROP TABLE IF EXISTS schema_migrations CASCADE; DROP TYPE IF EXISTS catalog_handling_class_type CASCADE;")

			if err := database.RunMigrations(ctx, &cfg.Postgres, "file://../../../../migrations"); err != nil {
				fmt.Printf("TestMain migration error: %v\n", err)
			}

			code := m.Run()
			_, _ = rawDB.Exec("SELECT pg_advisory_unlock($1)", int64(suppliersTestDBLockKey))
			rawDB.Close()
			os.Exit(code)
		}
	}
	os.Exit(m.Run())
}

type supEnv struct {
	db  *database.DB
	srv *httptest.Server
}

func supNop(next http.Handler) http.Handler { return next }

func supPrincipal(id int64, staff map[int64]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(suppliers_router.WithPrincipalID(r.Context(), id)))
		})
	}
}

func supStaff(staff map[int64]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !staff[suppliers_router.PrincipalIDFromContext(r.Context())] {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, `{"detail":"staff only","code":"forbidden"}`)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func setupSupEnv(t *testing.T, id int64, staff map[int64]bool) *supEnv {
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
	svc := suppliers.NewService(db)
	catalogSupplierRepo := catalog_repo.NewSupplierRepository(db)
	catalogProductRepo := catalog_repo.NewProductRepository(db)
	commerceRepo := commerce_repo.NewCommerceRepository(db)
	inventoryService := inventory.NewService(db, nil)
	promotionService := promotions.NewService(db, nil)
	pricingServices := pricing_service.NewServices(db)
	commerceService := commerce_service.NewCommerceService(db, pricingServices.PriceList, inventoryService, nil, promotionService)
	svc.SetPortalDependencies(catalogSupplierRepo, catalogProductRepo, commerceRepo, commerceService, inventoryService)

	rt := suppliers.New(svc)
	rt.RegisterRoutes(supPrincipal(id, staff), supStaff(staff))

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Mount("/api/v1", rt.ChiRouter())
	server := httptest.NewServer(mux)

	t.Cleanup(func() {
		server.Close()
		db.Close()
	})
	return &supEnv{db: db, srv: server}
}

var supCounter int64

func supSuffix() string {
	c := atomic.AddInt64(&supCounter, 1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano()%1000000, c)
}

func supDo(t *testing.T, env *supEnv, method, path, body string) (*http.Response, []byte) {
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

func supUser(t *testing.T, env *supEnv, userType string) int64 {
	t.Helper()
	suffix := supSuffix()
	var id int64
	err := env.db.Pool.QueryRow(context.Background(), `
		INSERT INTO identity.user (code, email, password_hash, user_type)
		VALUES ($1, $2, 'hash', $3) RETURNING id
	`, "usr_"+suffix, "u_"+suffix+"@test.local", userType).Scan(&id)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return id
}

func supApply(t *testing.T, env *supEnv, body string, want int) suppliers_schema.SupplierResponse {
	t.Helper()
	resp, rbody := supDo(t, env, http.MethodPost, "/api/v1/suppliers/apply", body)
	if resp.StatusCode != want {
		t.Fatalf("apply: expected %d, got %d: %s", want, resp.StatusCode, string(rbody))
	}
	if want != http.StatusCreated {
		return suppliers_schema.SupplierResponse{}
	}
	var out suppliers_schema.SupplierResponse
	_ = json.Unmarshal(rbody, &out)
	if loc := resp.Header.Get("Location"); loc == "" {
		t.Errorf("expected Location on 201")
	}
	return out
}

func supID(t *testing.T, env *supEnv, code string) int64 {
	t.Helper()
	var id int64
	if err := env.db.Pool.QueryRow(context.Background(), `SELECT id FROM suppliers.supplier_profile WHERE code=$1`, code).Scan(&id); err != nil {
		t.Fatalf("supplier id: %v", err)
	}
	return id
}

func TestSuppliers_ApplyAndApprove(t *testing.T) {
	env := setupSupEnv(t, 0, map[int64]bool{})
	staffUser := supUser(t, env, "staff")
	staff := map[int64]bool{staffUser: true}
	senv := setupSupEnv(t, staffUser, staff)

	// Anonymous application (no user link).
	anon := supApply(t, env, `{"company_name":"Anon Foods","contact_email":"a@b.local"}`, http.StatusCreated)
	if anon.Status != "applied" {
		t.Fatalf("expected applied: %+v", anon)
	}

	// Empty company → 400.
	supApply(t, env, `{"company_name":""}`, http.StatusBadRequest)
	supApply(t, env, `{"company_name":""}`, http.StatusBadRequest)

	// Authenticated applicant links automatically.
	supUserID := supUser(t, env, "supplier")
	linkedEnv := setupSupEnv(t, supUserID, staff)
	linked := supApply(t, linkedEnv, `{"company_name":"Linked Foods"}`, http.StatusCreated)
	var linkedUser *int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT user_id FROM suppliers.supplier_profile WHERE code=$1`, linked.Code).Scan(&linkedUser)
	if linkedUser == nil || *linkedUser != supUserID {
		t.Errorf("expected user link %d, got %+v", supUserID, linkedUser)
	}

	// Approve with user link.
	oid := supID(t, env, anon.Code)
	resp, body := supDo(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/suppliers/%d/approve", oid),
		fmt.Sprintf(`{"user_id":%d}`, supUserID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve: %d: %s", resp.StatusCode, string(body))
	}
	var approved suppliers_schema.SupplierResponse
	_ = json.Unmarshal(body, &approved)
	if approved.Status != "approved" {
		t.Errorf("expected approved: %+v", approved)
	}

	// Double approve → 422.
	resp, _ = supDo(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/suppliers/%d/approve", oid), `{}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("double approve: expected 422, got %d", resp.StatusCode)
	}

	// Approve with bogus user → 422 (FK guarded).
	oid2 := supID(t, env, linked.Code)
	resp, body = supDo(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/suppliers/%d/approve", oid2),
		`{"user_id":999999999}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bogus user: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// Missing supplier → 404.
	resp, _ = supDo(t, senv, http.MethodPost, "/api/v1/admin/suppliers/999999999/approve", `{}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing: expected 404, got %d", resp.StatusCode)
	}
	// Malformed id → 400.
	resp, _ = supDo(t, senv, http.MethodPost, "/api/v1/admin/suppliers/abc/approve", `{}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad id: expected 400, got %d", resp.StatusCode)
	}
}

func TestSuppliers_Contracts(t *testing.T) {
	env := setupSupEnv(t, 0, map[int64]bool{})
	staffUser := supUser(t, env, "staff")
	staff := map[int64]bool{staffUser: true}
	senv := setupSupEnv(t, staffUser, staff)
	app := supApply(t, env, `{"company_name":"Contract Foods"}`, http.StatusCreated)
	oid := supID(t, env, app.Code)
	resp, _ := supDo(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/suppliers/%d/approve", oid), `{}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve: %d", resp.StatusCode)
	}

	setContract := func(body string, want int) suppliers_schema.ContractResponse {
		t.Helper()
		resp, rbody := supDo(t, senv, http.MethodPut, fmt.Sprintf("/api/v1/admin/suppliers/%d/contracts", oid), body)
		if resp.StatusCode != want {
			t.Fatalf("set contract: expected %d, got %d: %s", want, resp.StatusCode, string(rbody))
		}
		if want != http.StatusCreated {
			return suppliers_schema.ContractResponse{}
		}
		var out suppliers_schema.ContractResponse
		_ = json.Unmarshal(rbody, &out)
		return out
	}

	// Empty terms → 400. Inverted window → 400.	setContract(`{"terms":""}`, http.StatusBadRequest)
	setContract(`{"terms":"t","valid_from":"2026-02-01T00:00:00Z","valid_to":"2026-01-01T00:00:00Z"}`, http.StatusBadRequest)

	// v1 then v2: versions increment, old deactivates.
	c1 := setContract(`{"terms":"net 30"}`, http.StatusCreated)
	if c1.Version != 1 || !c1.IsCurrent {
		t.Fatalf("v1 wrong: %+v", c1)
	}
	c2 := setContract(`{"terms":"net 15"}`, http.StatusCreated)
	if c2.Version != 2 || !c2.IsCurrent {
		t.Fatalf("v2 wrong: %+v", c2)
	}
	resp, body := supDo(t, senv, http.MethodGet, fmt.Sprintf("/api/v1/admin/suppliers/%d/contracts", oid), "")
	var list []suppliers_schema.ContractResponse
	_ = json.Unmarshal(body, &list)
	if len(list) != 2 || list[0].Version != 2 || list[1].Version != 1 {
		t.Errorf("history wrong: %+v", list)
	}
	var current int
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM suppliers.supplier_contract WHERE supplier_id=$1 AND is_current`, oid).Scan(&current)
	if current != 1 {
		t.Errorf("expected exactly 1 current contract, got %d", current)
	}

	// Contracts for missing supplier → 404.
	resp, _ = supDo(t, senv, http.MethodGet, "/api/v1/admin/suppliers/999999999/contracts", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing contracts: expected 404, got %d", resp.StatusCode)
	}
	_ = resp
}

func TestSuppliers_MeAndRBAC(t *testing.T) {
	env := setupSupEnv(t, 0, map[int64]bool{})
	staffUser := supUser(t, env, "staff")
	staff := map[int64]bool{staffUser: true}
	senv := setupSupEnv(t, staffUser, staff)

	// Unlinked principal → 404 on /supplier/me (never 403).
	anonID := supUser(t, env, "supplier")
	anonEnv := setupSupEnv(t, anonID, staff)
	resp, _ := supDo(t, anonEnv, http.MethodGet, "/api/v1/supplier/me", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unlinked me: expected 404, got %d", resp.StatusCode)
	}

	// Apply as this user, approve with link, then read + update own profile.
	app := supApply(t, anonEnv, `{"company_name":"Self Foods","contact_email":"s@s.local"}`, http.StatusCreated)
	oid := supID(t, env, app.Code)
	resp, _ = supDo(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/suppliers/%d/approve", oid),
		fmt.Sprintf(`{"user_id":%d}`, anonID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve+link: %d", resp.StatusCode)
	}
	resp, body := supDo(t, anonEnv, http.MethodGet, "/api/v1/supplier/me", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me: %d: %s", resp.StatusCode, string(body))
	}
	var me suppliers_schema.SupplierResponse
	_ = json.Unmarshal(body, &me)
	if me.Code != app.Code || me.Status != "approved" {
		t.Errorf("bad profile: %+v", me)
	}
	resp, body = supDo(t, anonEnv, http.MethodPatch, "/api/v1/supplier/me",
		`{"contact_phone":"+1000","company_name":"Self Foods LLC"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update me: %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &me)
	if me.CompanyName != "Self Foods LLC" {
		t.Errorf("update not applied: %+v", me)
	}

	// Other supplier cannot reach it (their /me resolves to themselves).
	otherID := supUser(t, env, "supplier")
	otherEnv := setupSupEnv(t, otherID, staff)
	resp, _ = supDo(t, otherEnv, http.MethodGet, "/api/v1/supplier/me", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("other supplier me: expected 404, got %d", resp.StatusCode)
	}

	// Buyer principal on admin + supplier paths → 403 / scoped properly.
	buyerEnv := setupSupEnv(t, 999001, map[int64]bool{})
	for _, p := range [][2]string{
		{http.MethodGet, "/api/v1/admin/suppliers"},
		{http.MethodPost, fmt.Sprintf("/api/v1/admin/suppliers/%d/approve", oid)},
		{http.MethodGet, fmt.Sprintf("/api/v1/admin/suppliers/%d/contracts", oid)},
		{http.MethodPut, fmt.Sprintf("/api/v1/admin/suppliers/%d/contracts", oid)},
	} {
		resp, _ := supDo(t, buyerEnv, p[0], p[1], `{"user_id":1}`)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s: expected 403, got %d", p[0], p[1], resp.StatusCode)
		}
	}
	// Admin list envelope + status filter.
	resp, body = supDo(t, senv, http.MethodGet, "/api/v1/admin/suppliers?status=approved", "")
	var envelope struct {
		Items []suppliers_schema.SupplierResponse `json:"items"`
		Total int                                 `json:"total"`
		Page  int                                 `json:"page"`
	}
	_ = json.Unmarshal(body, &envelope)
	if envelope.Total < 1 || len(envelope.Items) != envelope.Total || envelope.Page != 1 {
		t.Errorf("envelope broken: %+v", envelope)
	}
	_ = resp
}

func TestSuppliers_ConcurrentApprove(t *testing.T) {
	env := setupSupEnv(t, 0, map[int64]bool{})
	staffUser := supUser(t, env, "staff")
	staff := map[int64]bool{staffUser: true}
	senv := setupSupEnv(t, staffUser, staff)
	app := supApply(t, env, `{"company_name":"Race Foods"}`, http.StatusCreated)
	oid := supID(t, env, app.Code)

	// Concurrent approves: exactly one acts; rest see terminal state.
	const n = 8
	var codes [n]int
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			resp, _ := supDo(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/suppliers/%d/approve", oid), `{}`)
			codes[i] = resp.StatusCode
		}(i)
	}
	close(start)
	wg.Wait()
	ok := 0
	for i, c := range codes {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusUnprocessableEntity:
		default:
			t.Errorf("req %d: unexpected %d", i, c)
		}
	}
	if ok != 1 {
		t.Errorf("expected exactly 1 approval, got %d (%v)", ok, codes)
	}
	var status string
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT status FROM suppliers.supplier_profile WHERE id=$1`, oid).Scan(&status)
	if status != "approved" {
		t.Errorf("expected approved, got %q", status)
	}
}

func TestSuppliers_PortalOperations(t *testing.T) {
	env := setupSupEnv(t, 0, map[int64]bool{})
	staffUser := supUser(t, env, "staff")
	staff := map[int64]bool{staffUser: true}
	senv := setupSupEnv(t, staffUser, staff)

	supUserID := supUser(t, env, "supplier")
	supEnvInst := setupSupEnv(t, supUserID, staff)

	// Apply and approve supplier profile linked to supUserID
	app := supApply(t, supEnvInst, `{"company_name":"Portal Foods"}`, http.StatusCreated)
	oid := supID(t, supEnvInst, app.Code)
	resp, _ := supDo(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/suppliers/%d/approve", oid),
		fmt.Sprintf(`{"user_id":%d}`, supUserID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve: %d", resp.StatusCode)
	}

	// Create catalog supplier linked to profile
	ctx := context.Background()
	suffix := supSuffix()
	var catSupID int64
	err := supEnvInst.db.Pool.QueryRow(ctx, `
		INSERT INTO catalog.supplier (code, name, slug, supplier_id, is_active)
		VALUES ($1, $2, $3, $4, true) RETURNING id
	`, "csup_"+suffix, "Catalog Portal Foods", "csup-"+suffix, oid).Scan(&catSupID)
	if err != nil {
		t.Fatalf("create catalog supplier: %v", err)
	}

	// Create category & product for the supplier
	var catID, unitID, prodID int64
	err = supEnvInst.db.Pool.QueryRow(ctx, `INSERT INTO catalog.category (code, name, slug) VALUES ($1, $2, $3) RETURNING id`, "cat_"+suffix, "Category", "cat-"+suffix).Scan(&catID)
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	err = supEnvInst.db.Pool.QueryRow(ctx, `INSERT INTO catalog.unit (code, name, symbol) VALUES ($1, $2, $3) RETURNING id`, "u_"+suffix, "Unit", "u").Scan(&unitID)
	if err != nil {
		t.Fatalf("create unit: %v", err)
	}
	err = supEnvInst.db.Pool.QueryRow(ctx, `
		INSERT INTO catalog.product (code, name, slug, category_id, base_unit_id, supplier_id, handling_class, status, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, 'ambient', 'published', true) RETURNING id
	`, "prod_"+suffix, "Test Product", "prod-"+suffix, catID, unitID, catSupID).Scan(&prodID)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	// Test GET /supplier/me/orders (initially empty)
	resp, body := supDo(t, supEnvInst, http.MethodGet, "/api/v1/supplier/me/orders", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("my orders: %d: %s", resp.StatusCode, string(body))
	}
	var orders []suppliers_schema.SupplierOrderResponse
	_ = json.Unmarshal(body, &orders)
	if len(orders) != 0 {
		t.Errorf("expected 0 orders, got %d", len(orders))
	}

	// Insert test order in 'placed' state directed to catSupID
	userIdForBuyer := supUser(t, supEnvInst, "buyer")
	var buyerProfileID int64
	err = supEnvInst.db.Pool.QueryRow(ctx, `
		INSERT INTO identity.buyer_profile (code, business_name, user_id)
		VALUES ($1, $2, $3) RETURNING id
	`, "bprof_"+suffix, "Test Buyer", userIdForBuyer).Scan(&buyerProfileID)
	if err != nil {
		t.Fatalf("create buyer profile: %v", err)
	}

	var orderID int64
	err = supEnvInst.db.Pool.QueryRow(ctx, `
		INSERT INTO commerce."order" (code, buyer_id, supplier_id, status, currency, subtotal_minor, discounts_minor, total_minor, placed_at)
		VALUES ($1, $2, $3, 'placed', 'USD', 1000, 0, 1000, NOW()) RETURNING id
	`, "ord_"+suffix, buyerProfileID, catSupID).Scan(&orderID)
	if err != nil {
		t.Fatalf("insert test order: %v", err)
	}
	_, _ = supEnvInst.db.Pool.Exec(ctx, `
		INSERT INTO commerce.order_line (order_id, product_code, product_name, quantity, unit_code, unit_price_minor, total_price_minor)
		VALUES ($1, $2, 'Test Product', 2, 'u_'+$3, 500, 1000)
	`, orderID, "prod_"+suffix, suffix)

	// GET /supplier/me/orders now returns 1 order
	resp, body = supDo(t, supEnvInst, http.MethodGet, "/api/v1/supplier/me/orders", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("my orders: %d", resp.StatusCode)
	}
	_ = json.Unmarshal(body, &orders)
	if len(orders) != 1 || orders[0].Code != "ord_"+suffix || orders[0].Status != "placed" {
		t.Errorf("unexpected orders response: %+v", orders)
	}

	// PATCH /supplier/me/orders/{code} (acknowledge -> confirmed)
	resp, body = supDo(t, supEnvInst, http.MethodPatch, fmt.Sprintf("/api/v1/supplier/me/orders/%s", "ord_"+suffix),
		`{"note":"ready for dispatch"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("acknowledge order: %d: %s", resp.StatusCode, string(body))
	}
	var acked suppliers_schema.SupplierOrderResponse
	_ = json.Unmarshal(body, &acked)
	if acked.Status != "confirmed" {
		t.Errorf("expected confirmed status, got %q", acked.Status)
	}

	// Second acknowledgment (exact-once / duplicate transition) -> 422
	resp, _ = supDo(t, supEnvInst, http.MethodPatch, fmt.Sprintf("/api/v1/supplier/me/orders/%s", "ord_"+suffix), `{}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 on duplicate acknowledge, got %d", resp.StatusCode)
	}

	// Other supplier cannot acknowledge or see order
	otherID := supUser(t, env, "supplier")
	otherEnv := setupSupEnv(t, otherID, staff)
	resp, _ = supDo(t, otherEnv, http.MethodPatch, fmt.Sprintf("/api/v1/supplier/me/orders/%s", "ord_"+suffix), `{}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("other supplier ack: expected 404, got %d", resp.StatusCode)
	}

	// PATCH /supplier/me/stock (push stock)
	resp, body = supDo(t, supEnvInst, http.MethodPatch, "/api/v1/supplier/me/stock",
		fmt.Sprintf(`{"product_code":"%s","lot_number":"LOT-999","quantity":50}`, "prod_"+suffix))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("push stock: %d: %s", resp.StatusCode, string(body))
	}
	var stockPush suppliers_schema.StockPushResponse
	_ = json.Unmarshal(body, &stockPush)
	if stockPush.AvailableQuantity != 50 || stockPush.LotNumber != "LOT-999" {
		t.Errorf("unexpected stock push response: %+v", stockPush)
	}

	// Invalid quantity -> 400
	resp, _ = supDo(t, supEnvInst, http.MethodPatch, "/api/v1/supplier/me/stock",
		fmt.Sprintf(`{"product_code":"%s","lot_number":"LOT-100","quantity":0}`, "prod_"+suffix))
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 on zero quantity, got %d", resp.StatusCode)
	}

	// Unowned product -> 404
	resp, _ = supDo(t, supEnvInst, http.MethodPatch, "/api/v1/supplier/me/stock",
		`{"product_code":"nonexistent","lot_number":"LOT-100","quantity":10}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 on unowned product, got %d", resp.StatusCode)
	}
}
