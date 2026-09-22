package router_test

// TASK-006 payments tests: intent lifecycle + idempotency, manual B2B
// settlement, invoice balance integrity, RBAC. Mounts both commerce (order
// fixtures) and payments routers, like cmd/api/main.go does.

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
	"github.com/atlas-platform/backend/internal/modules/commerce"
	commerce_router "github.com/atlas-platform/backend/internal/modules/commerce/router"
	commerce_service "github.com/atlas-platform/backend/internal/modules/commerce/service"
	inventory_service "github.com/atlas-platform/backend/internal/modules/inventory/service"
	"github.com/atlas-platform/backend/internal/modules/payments"
	payments_router "github.com/atlas-platform/backend/internal/modules/payments/router"
	payments_schema "github.com/atlas-platform/backend/internal/modules/payments/schema"
	payments_service "github.com/atlas-platform/backend/internal/modules/payments/service"
	pricing_service "github.com/atlas-platform/backend/internal/modules/pricing/service"
	promotions_service "github.com/atlas-platform/backend/internal/modules/promotions/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Same advisory lock key as commerce/inventory suites: test packages share
// one database and TestMain drop+migrate must never run concurrently.
const paymentsTestDBLockKey = 1946819412

func TestMain(m *testing.M) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err == nil {
		rawDB, err := sql.Open("pgx", cfg.PostgresDSN())
		if err == nil && rawDB != nil {
			_, _ = rawDB.Exec("SELECT pg_advisory_lock($1)", int64(paymentsTestDBLockKey))
			_, _ = rawDB.Exec("DROP SCHEMA IF EXISTS catalog CASCADE; DROP SCHEMA IF EXISTS inventory CASCADE; DROP SCHEMA IF EXISTS identity CASCADE; DROP SCHEMA IF EXISTS pricing CASCADE; DROP SCHEMA IF EXISTS commerce CASCADE; DROP SCHEMA IF EXISTS payments CASCADE; DROP SCHEMA IF EXISTS promotions CASCADE; DROP SCHEMA IF EXISTS crm CASCADE; DROP SCHEMA IF EXISTS suppliers CASCADE; DROP SCHEMA IF EXISTS cms CASCADE; DROP SCHEMA IF EXISTS reports CASCADE; DROP SCHEMA IF EXISTS erp CASCADE; DROP SCHEMA IF EXISTS platform CASCADE; DROP SCHEMA IF EXISTS ai CASCADE; DROP SCHEMA IF EXISTS analytics CASCADE; DROP TABLE IF EXISTS schema_migrations CASCADE; DROP TYPE IF EXISTS catalog_handling_class_type CASCADE;")

			if err := database.RunMigrations(ctx, &cfg.Postgres, "file://../../../../migrations"); err != nil {
				fmt.Printf("TestMain migration error: %v\n", err)
			}

			code := m.Run()
			_, _ = rawDB.Exec("SELECT pg_advisory_unlock($1)", int64(paymentsTestDBLockKey))
			rawDB.Close()
			os.Exit(code)
		}
	}
	os.Exit(m.Run())
}

type payEnv struct {
	db          *database.DB
	srv         *httptest.Server
	csrv        *httptest.Server
	commerceSvc *commerce_service.CommerceService
	paymentSvc  *payments_service.PaymentService
}

func payNop(next http.Handler) http.Handler { return next }

func payPrincipal(staff map[int64]bool, id int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := commerce_router.WithPrincipalID(r.Context(), id)
			ctx = payments_router.WithPrincipalID(ctx, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func payStaff(staff map[int64]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !staff[payments_router.PrincipalIDFromContext(r.Context())] {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, `{"detail":"staff only","code":"forbidden"}`)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func setupPayEnv(t *testing.T, id int64, staff map[int64]bool) *payEnv {
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

	pricingSvc := pricing_service.NewServices(db)
	inventorySvc := inventory_service.NewInventoryService(db, nil)
	promotionSvc := promotions_service.NewPromotionService(db, nil)
	commerceSvc := commerce_service.NewCommerceService(db, pricingSvc.PriceList, inventorySvc, nil, promotionSvc)
	paymentSvc := payments_service.NewPaymentService(db, commerceSvc, nil)

	crt := commerce.New(commerceSvc)
	crt.RegisterRoutes(payPrincipal(staff, id), payStaff(staff))
	prt := payments.New(paymentSvc)
	prt.RegisterRoutes(payPrincipal(staff, id), payStaff(staff))

	cmux := chi.NewRouter()
	cmux.Use(middleware.RequestID)
	cmux.Mount("/api/v1", crt.ChiRouter())
	cserver := httptest.NewServer(cmux)

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Mount("/api/v1", prt.ChiRouter())
	mux.Mount("/", prt.WebhookRouter())
	server := httptest.NewServer(mux)

	t.Cleanup(func() {
		server.Close()
		cserver.Close()
		db.Close()
	})
	return &payEnv{db: db, srv: server, csrv: cserver, commerceSvc: commerceSvc, paymentSvc: paymentSvc}
}

var payCounter int64

func paySuffix() string {
	c := atomic.AddInt64(&payCounter, 1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano()%1000000, c)
}

func payBuyer(t *testing.T, env *payEnv) int64 {
	t.Helper()
	suffix := paySuffix()
	var userID, buyerID int64
	err := env.db.WithTx(context.Background(), func(tx *database.Tx) error {
		if err := tx.QueryRow(context.Background(), `
			INSERT INTO identity.user (code, email, password_hash, user_type)
			VALUES ($1, $2, 'hash', 'buyer') RETURNING id
		`, "usr_"+suffix, "buyer_"+suffix+"@test.local").Scan(&userID); err != nil {
			return err
		}
		return tx.QueryRow(context.Background(), `
			INSERT INTO identity.buyer_profile (user_id, code, business_name)
			VALUES ($1, $2, $3) RETURNING id
		`, userID, "buyer_prof_"+suffix, "Biz "+suffix).Scan(&buyerID)
	})
	if err != nil {
		t.Fatalf("create buyer: %v", err)
	}
	return buyerID
}

func payProduct(t *testing.T, env *payEnv) (string, int64, int64) {
	t.Helper()
	ctx := context.Background()
	suffix := paySuffix()
	var categoryID, unitID, supplierID, productID int64
	var code string
	err := env.db.WithTx(ctx, func(tx *database.Tx) error {
		_ = tx.QueryRow(ctx, `INSERT INTO catalog.category (code, name, slug) VALUES ($1,$2,$3) RETURNING id`,
			"cat_"+suffix, "Cat", "cat-"+suffix).Scan(&categoryID)
		_ = tx.QueryRow(ctx, `INSERT INTO catalog.unit (code, name, symbol, unit_type) VALUES ($1,$2,$3,'base') RETURNING id`,
			"unit_"+suffix, "Piece", "pcs").Scan(&unitID)
		_ = tx.QueryRow(ctx, `INSERT INTO catalog.supplier (code, supplier_id, name, slug) VALUES ($1,$2,$3,$4) RETURNING id`,
			"sup_"+suffix, 3000+atomic.AddInt64(&payCounter, 1), "Sup", "sup-"+suffix).Scan(&supplierID)
		code = "prod_" + suffix
		return tx.QueryRow(ctx, `
			INSERT INTO catalog.product (code, slug, name, category_id, handling_class, base_unit_id, supplier_id, status, is_active, base_price_minor)
			VALUES ($1,$2,$3,$4,'ambient',$5,$6,'published',true,1500) RETURNING id
		`, code, "prod-"+suffix, "Product "+suffix, categoryID, unitID, supplierID).Scan(&productID)
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	return code, productID, supplierID
}

func payStock(t *testing.T, env *payEnv, productID, supplierID int64, qty int32) {
	t.Helper()
	ctx := context.Background()
	suffix := paySuffix()
	var slID int64
	if err := env.db.Pool.QueryRow(ctx, `
		INSERT INTO inventory.stock_level (code, product_id, supplier_id, available_quantity, total_quantity)
		VALUES ($1,$2,$3,$4,$4)
		ON CONFLICT (product_id, supplier_id) DO UPDATE
		SET available_quantity = inventory.stock_level.available_quantity + $4,
		    total_quantity = inventory.stock_level.total_quantity + $4
		RETURNING id
	`, "sl_"+suffix, productID, supplierID, qty).Scan(&slID); err != nil {
		t.Fatalf("stock level: %v", err)
	}
	if _, err := env.db.Pool.Exec(ctx, `
		INSERT INTO inventory.lot (code, stock_level_id, lot_number, initial_quantity, available_quantity, status, expires_at)
		VALUES ($1,$2,$3,$4,$4,'active',$5)
	`, "lot_"+suffix, slID, "LOT-"+suffix, qty, time.Now().Add(30*24*time.Hour)); err != nil {
		t.Fatalf("lot: %v", err)
	}
}

func payDo(t *testing.T, env *payEnv, method, path, body string, headers map[string]string) (*http.Response, []byte) {
	return payDoAgainst(t, env.srv.URL, method, path, body, headers)
}

func payDoCommerce(t *testing.T, env *payEnv, method, path, body string, headers map[string]string) (*http.Response, []byte) {
	return payDoAgainst(t, env.csrv.URL, method, path, body, headers)
}

func payDoAgainst(t *testing.T, base, method, path, body string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req, err := http.NewRequest(method, base+path, reader)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	return resp, buf.Bytes()
}

// payOrderFixture checks out a 1-line cart and returns the order code.
func payOrderFixture(t *testing.T, env *payEnv, buyerEnv *payEnv, prodCode string, key string) string {
	t.Helper()
	resp, body := payDoCommerce(t, buyerEnv, http.MethodPost, "/api/v1/commerce/cart/items",
		fmt.Sprintf(`{"product_code":%q,"quantity":2}`, prodCode), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add item: %d: %s", resp.StatusCode, string(body))
	}
	_ = env
	resp, body = payDoCommerce(t, buyerEnv, http.MethodPost, "/api/v1/commerce/checkout", "",
		map[string]string{"Idempotency-Key": key})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("checkout: %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		Orders []struct {
			Code string `json:"code"`
		} `json:"orders"`
	}
	if err := json.Unmarshal(body, &out); err != nil || len(out.Orders) != 1 {
		t.Fatalf("decode checkout: %v %s", err, string(body))
	}
	return out.Orders[0].Code
}

func payMethod(t *testing.T, env *payEnv, buyerID int64) string {
	t.Helper()
	m, err := env.paymentSvc.CreateMethod(context.Background(), buyerID, "bank_transfer", "Bank transfer")
	if err != nil {
		t.Fatalf("create method: %v", err)
	}
	return m.Code
}

func TestPayments_IntentLifecycle(t *testing.T) {
	env := setupPayEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{999001: true}
	prodCode, productID, supplierID := payProduct(t, env)
	payStock(t, env, productID, supplierID, 10)
	buyerEnv := setupPayEnv(t, buyerID, staff)
	staffEnv := setupPayEnv(t, 999001, staff)
	methodCode := payMethod(t, buyerEnv, buyerID)

	orderCode := payOrderFixture(t, env, buyerEnv, prodCode, "pay-lc-1")

	// Create intent: amount snapshotted from order total (2 x 1500).
	resp, body := payDo(t, buyerEnv, http.MethodPost, "/api/v1/payments/intents",
		fmt.Sprintf(`{"order_code":%q,"method_code":%q,"idem_key":"intent-1"}`, orderCode, methodCode), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create intent: %d: %s", resp.StatusCode, string(body))
	}
	var intent payments_schema.IntentResponse
	_ = json.Unmarshal(body, &intent)
	if intent.AmountMinor != 3000 || intent.Status != "requires_action" {
		t.Errorf("unexpected intent: %+v", intent)
	}

	// Issue an invoice, then complete the intent → balance cleared.
	resp, body = payDoCommerce(t, staffEnv, http.MethodPost, "/api/v1/admin/invoices",
		fmt.Sprintf(`{"order_id":%d}`, payOrderID(t, env, orderCode)), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create invoice: %d: %s", resp.StatusCode, string(body))
	}
	var inv struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(body, &inv)
	var invID int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT id FROM commerce.invoice WHERE code=$1`, inv.Code).Scan(&invID)
	resp, body = payDoCommerce(t, staffEnv, http.MethodPost, fmt.Sprintf("/api/v1/admin/invoices/%d/issue", invID), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("issue invoice: %d: %s", resp.StatusCode, string(body))
	}

	resp, body = payDo(t, staffEnv, http.MethodPost, "/api/v1/admin/payments/intents/"+intent.Code+"/complete", `{"note":"wire received"}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("complete: %d: %s", resp.StatusCode, string(body))
	}
	var done payments_schema.IntentResponse
	_ = json.Unmarshal(body, &done)
	if done.Status != "succeeded" || len(done.Attempts) != 1 {
		t.Errorf("unexpected completed intent: %+v", done)
	}
	var balance int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT balance_minor FROM commerce.invoice WHERE id=$1`, invID).Scan(&balance)
	if balance != 0 {
		t.Errorf("expected invoice balance 0 after payment, got %d", balance)
	}

	// Double complete → 409 terminal.
	resp, body = payDo(t, staffEnv, http.MethodPost, "/api/v1/admin/payments/intents/"+intent.Code+"/complete", `{}`, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("double complete: expected 409, got %d: %s", resp.StatusCode, string(body))
	}

	// Foreign buyer cannot read the intent.
	otherID := payBuyer(t, env)
	otherEnv := setupPayEnv(t, otherID, staff)
	resp, _ = payDo(t, otherEnv, http.MethodGet, "/api/v1/payments/intents/"+intent.Code, "", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("foreign intent read: expected 404, got %d", resp.StatusCode)
	}
}

func payOrderID(t *testing.T, env *payEnv, code string) int64 {
	t.Helper()
	var id int64
	if err := env.db.Pool.QueryRow(context.Background(), `SELECT id FROM commerce."order" WHERE code=$1`, code).Scan(&id); err != nil {
		t.Fatalf("order id: %v", err)
	}
	return id
}

func TestPayments_IntentIdempotency(t *testing.T) {
	env := setupPayEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{}
	prodCode, productID, supplierID := payProduct(t, env)
	payStock(t, env, productID, supplierID, 10)
	buyerEnv := setupPayEnv(t, buyerID, staff)
	orderCode := payOrderFixture(t, env, buyerEnv, prodCode, "pay-idem-1")

	first := func() (int, []byte) {
		resp, body := payDo(t, buyerEnv, http.MethodPost, "/api/v1/payments/intents",
			fmt.Sprintf(`{"order_code":%q,"idem_key":"idem-a"}`, orderCode), nil)
		return resp.StatusCode, body
	}
	st1, b1 := first()
	if st1 != http.StatusCreated {
		t.Fatalf("first: %d: %s", st1, string(b1))
	}
	st2, b2 := first()
	if st2 != http.StatusOK {
		t.Fatalf("replay: expected 200, got %d: %s", st2, string(b2))
	}
	var r1, r2 payments_schema.IntentResponse
	_ = json.Unmarshal(b1, &r1)
	_ = json.Unmarshal(b2, &r2)
	if r1.Code != r2.Code {
		t.Errorf("replay returned different intent: %q vs %q", r1.Code, r2.Code)
	}

	// Same key, different order → 409.
	prodCode2, productID2, supplierID2 := payProduct(t, env)
	payStock(t, env, productID2, supplierID2, 10)
	resp, body := payDoCommerce(t, buyerEnv, http.MethodPost, "/api/v1/commerce/cart/items",
		fmt.Sprintf(`{"product_code":%q,"quantity":1}`, prodCode2), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add: %d: %s", resp.StatusCode, string(body))
	}
	resp, body = payDoCommerce(t, buyerEnv, http.MethodPost, "/api/v1/commerce/checkout", "",
		map[string]string{"Idempotency-Key": "pay-idem-2"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("second checkout: %d: %s", resp.StatusCode, string(body))
	}
	var co struct {
		Orders []struct {
			Code string `json:"code"`
		} `json:"orders"`
	}
	_ = json.Unmarshal(body, &co)
	resp, body = payDo(t, buyerEnv, http.MethodPost, "/api/v1/payments/intents",
		fmt.Sprintf(`{"order_code":%q,"idem_key":"idem-a"}`, co.Orders[0].Code), nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("different target: expected 409, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestPayments_FailAndRBAC(t *testing.T) {
	env := setupPayEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{999002: true}
	prodCode, productID, supplierID := payProduct(t, env)
	payStock(t, env, productID, supplierID, 10)
	buyerEnv := setupPayEnv(t, buyerID, staff)
	staffEnv := setupPayEnv(t, 999002, staff)
	orderCode := payOrderFixture(t, env, buyerEnv, prodCode, "pay-fail-1")

	resp, body := payDo(t, buyerEnv, http.MethodPost, "/api/v1/payments/intents",
		fmt.Sprintf(`{"order_code":%q,"idem_key":"fail-1"}`, orderCode), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d: %s", resp.StatusCode, string(body))
	}
	var intent payments_schema.IntentResponse
	_ = json.Unmarshal(body, &intent)

	// Buyer cannot mark (staff-only).
	resp, _ = payDo(t, buyerEnv, http.MethodPost, "/api/v1/admin/payments/intents/"+intent.Code+"/complete", `{}`, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("buyer mark: expected 403, got %d", resp.StatusCode)
	}

	// Staff fail path.
	resp, body = payDo(t, staffEnv, http.MethodPost, "/api/v1/admin/payments/intents/"+intent.Code+"/fail", `{"note":"declined"}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fail: %d: %s", resp.StatusCode, string(body))
	}
	var failed payments_schema.IntentResponse
	_ = json.Unmarshal(body, &failed)
	if failed.Status != "failed" || len(failed.Attempts) != 1 {
		t.Errorf("unexpected failed intent: %+v", failed)
	}

	// Admin list envelope.
	resp, body = payDo(t, staffEnv, http.MethodGet, "/api/v1/admin/payments/?status=failed", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin list: %d: %s", resp.StatusCode, string(body))
	}
	var envelope struct {
		Items   []payments_schema.IntentResponse `json:"items"`
		Page    int                              `json:"page"`
		Total   int                              `json:"total"`
		HasNext bool                             `json:"has_next"`
	}
	_ = json.Unmarshal(body, &envelope)
	if envelope.Total < 1 || len(envelope.Items) < 1 {
		t.Errorf("expected failed intent in list: %+v", envelope)
	}

	// Buyer methods list + unknown method on create.
	resp, body = payDo(t, buyerEnv, http.MethodGet, "/api/v1/payments/methods", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("methods: %d: %s", resp.StatusCode, string(body))
	}
	resp, body = payDo(t, buyerEnv, http.MethodPost, "/api/v1/payments/intents",
		fmt.Sprintf(`{"order_code":%q,"method_code":"nope","idem_key":"fail-2"}`, orderCode), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("bad method: expected 404, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestPayments_OverpaymentRejected(t *testing.T) {
	env := setupPayEnv(t, 0, map[int64]bool{})
	buyerID := payBuyer(t, env)
	staff := map[int64]bool{999003: true}
	prodCode, productID, supplierID := payProduct(t, env)
	payStock(t, env, productID, supplierID, 10)
	buyerEnv := setupPayEnv(t, buyerID, staff)
	staffEnv := setupPayEnv(t, 999003, staff)
	orderCode := payOrderFixture(t, env, buyerEnv, prodCode, "pay-over-1")
	oid := payOrderID(t, env, orderCode)

	// Issue one invoice for the full total (3000).
	resp, body := payDoCommerce(t, staffEnv, http.MethodPost, "/api/v1/admin/invoices",
		fmt.Sprintf(`{"order_id":%d}`, oid), nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("invoice: %d: %s", resp.StatusCode, string(body))
	}
	var inv struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(body, &inv)
	var invID int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT id FROM commerce.invoice WHERE code=$1`, inv.Code).Scan(&invID)
	resp, _ = payDoCommerce(t, staffEnv, http.MethodPost, fmt.Sprintf("/api/v1/admin/invoices/%d/issue", invID), "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("issue: %d", resp.StatusCode)
	}

	mkIntent := func(key string) string {
		t.Helper()
		resp, body := payDo(t, buyerEnv, http.MethodPost, "/api/v1/payments/intents",
			fmt.Sprintf(`{"order_code":%q,"idem_key":%q}`, orderCode, key), nil)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("intent %s: %d: %s", key, resp.StatusCode, string(body))
		}
		var out payments_schema.IntentResponse
		_ = json.Unmarshal(body, &out)
		return out.Code
	}
	first := mkIntent("over-1")
	second := mkIntent("over-2")

	// First completion clears the balance.
	resp, _ = payDo(t, staffEnv, http.MethodPost, "/api/v1/admin/payments/intents/"+first+"/complete", `{}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("complete first: %d", resp.StatusCode)
	}
	// Second completion for the same total exceeds the balance → 422, and
	// the invoice balance stays at zero (no double-apply).
	resp, body = payDo(t, staffEnv, http.MethodPost, "/api/v1/admin/payments/intents/"+second+"/complete", `{}`, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("overpayment: expected 422, got %d: %s", resp.StatusCode, string(body))
	}
	var balance int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT balance_minor FROM commerce.invoice WHERE id=$1`, invID).Scan(&balance)
	if balance != 0 {
		t.Errorf("balance changed by rejected overpayment: %d", balance)
	}
}
