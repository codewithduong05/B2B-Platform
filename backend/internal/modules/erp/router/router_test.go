package router_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/erp"
	erp_schema "github.com/atlas-platform/backend/internal/modules/erp/schema"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const erpTestDBLockKey = 1946819412

func TestMain(m *testing.M) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err == nil {
		rawDB, err := sql.Open("pgx", cfg.PostgresDSN())
		if err == nil && rawDB != nil {
			_, _ = rawDB.Exec("SELECT pg_advisory_lock($1)", int64(erpTestDBLockKey))
			_, _ = rawDB.Exec("DROP SCHEMA IF EXISTS catalog CASCADE; DROP SCHEMA IF EXISTS inventory CASCADE; DROP SCHEMA IF EXISTS identity CASCADE; DROP SCHEMA IF EXISTS pricing CASCADE; DROP SCHEMA IF EXISTS commerce CASCADE; DROP SCHEMA IF EXISTS payments CASCADE; DROP SCHEMA IF EXISTS promotions CASCADE; DROP SCHEMA IF EXISTS crm CASCADE; DROP SCHEMA IF EXISTS suppliers CASCADE; DROP SCHEMA IF EXISTS cms CASCADE; DROP SCHEMA IF EXISTS reports CASCADE; DROP SCHEMA IF EXISTS erp CASCADE; DROP SCHEMA IF EXISTS platform CASCADE; DROP SCHEMA IF EXISTS ai CASCADE; DROP SCHEMA IF EXISTS analytics CASCADE; DROP TABLE IF EXISTS schema_migrations CASCADE; DROP TYPE IF EXISTS catalog_handling_class_type CASCADE;")

			if err := database.RunMigrations(ctx, &cfg.Postgres, "file://../../../../migrations"); err != nil {
				fmt.Printf("TestMain migration error: %v\n", err)
			}

			code := m.Run()
			_, _ = rawDB.Exec("SELECT pg_advisory_unlock($1)", int64(erpTestDBLockKey))
			rawDB.Close()
			os.Exit(code)
		}
	}
	os.Exit(m.Run())
}

type erpEnv struct {
	db  *database.DB
	srv *httptest.Server
}

func setupERPEnv(t *testing.T) *erpEnv {
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
	svc := erp.NewService(db)
	svc.SetWebhookSecret("test-erp-secret")
	rt := erp.New(svc)
	rt.RegisterRoutes(
		func(next http.Handler) http.Handler { return next },
		func(next http.Handler) http.Handler { return next },
	)

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Mount("/api/v1", rt.ChiRouter())
	mux.Mount("/", rt.WebhookRouter())
	server := httptest.NewServer(mux)

	t.Cleanup(func() {
		server.Close()
		db.Close()
	})
	return &erpEnv{db: db, srv: server}
}

func erpDo(t *testing.T, env *erpEnv, method, path, body string) (*http.Response, []byte) {
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

func signWebhook(secret string, body []byte) (timestamp, signature string) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))
	return ts, sig
}

func erpWebhookDo(t *testing.T, env *erpEnv, topic string, body []byte, secret string) (*http.Response, []byte) {
	t.Helper()
	ts, sig := signWebhook(secret, body)
	req, err := http.NewRequest(http.MethodPost, env.srv.URL+"/webhooks/erp/"+topic, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Timestamp", ts)
	req.Header.Set("X-Webhook-Signature", sig)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	return resp, buf.Bytes()
}

func seedOrder(t *testing.T, env *erpEnv) int64 {
	t.Helper()
	ctx := context.Background()
	var userID, buyerID, categoryID, supplierID, unitID, productID int64
	_ = env.db.Pool.QueryRow(ctx,
		`INSERT INTO identity."user" (code, email, password_hash, user_type) VALUES ('usr_erp001', 'erp_user@test.com', 'hash', 'buyer') RETURNING id`).Scan(&userID)
	_ = env.db.Pool.QueryRow(ctx,
		`INSERT INTO identity.buyer_profile (code, business_name, user_id) VALUES ('test_buyer_erp', 'Test', $1) RETURNING id`,
		userID).Scan(&buyerID)
	_ = env.db.Pool.QueryRow(ctx,
		`INSERT INTO catalog.category (code, name, slug) VALUES ('erp_cat', 'ERP Category', 'erp-cat') RETURNING id`).Scan(&categoryID)
	_ = env.db.Pool.QueryRow(ctx,
		`INSERT INTO catalog.supplier (code, supplier_id, name, slug) VALUES ('test_sup_erp', 1, 'TestSup', 'test-sup-erp') RETURNING id`).Scan(&supplierID)
	_ = env.db.Pool.QueryRow(ctx,
		`INSERT INTO catalog.unit (code, name, symbol) VALUES ('erp_unit', 'ERP Unit', 'eu') RETURNING id`).Scan(&unitID)
	_ = env.db.Pool.QueryRow(ctx,
		`INSERT INTO catalog.product (code, slug, name, category_id, handling_class, supplier_id, base_unit_id, base_price_minor, status) VALUES ('erp_prod', 'erp-prod', 'ERP Product', $1, 'ambient', $2, $3, 100, 'published') RETURNING id`,
		categoryID, supplierID, unitID).Scan(&productID)
	var orderID int64
	_ = env.db.Pool.QueryRow(ctx,
		`INSERT INTO commerce."order" (code, buyer_id, supplier_id, total_minor, status) VALUES ('erp_ord_test', $1, $2, 100, 'confirmed') RETURNING id`,
		buyerID, supplierID).Scan(&orderID)
	return orderID
}

func seedProduct(t *testing.T, env *erpEnv) {
	t.Helper()
	ctx := context.Background()
	var categoryID, supplierID, unitID int64
	_ = env.db.Pool.QueryRow(ctx,
		`INSERT INTO catalog.category (code, name, slug) VALUES ('erp_cat_sync', 'ERP Category Sync', 'erp-cat-sync') RETURNING id`).Scan(&categoryID)
	_ = env.db.Pool.QueryRow(ctx,
		`INSERT INTO catalog.supplier (code, supplier_id, name, slug) VALUES ('test_sup_erp2', 1, 'TestSup2', 'test-sup-erp2') RETURNING id`).Scan(&supplierID)
	_ = env.db.Pool.QueryRow(ctx,
		`INSERT INTO catalog.unit (code, name, symbol) VALUES ('erp_unit2', 'ERP Unit2', 'eu2') RETURNING id`).Scan(&unitID)
	_, _ = env.db.Pool.Exec(ctx,
		`INSERT INTO catalog.product (code, slug, name, category_id, handling_class, supplier_id, base_unit_id, base_price_minor, status) VALUES ('erp_prod_sync', 'erp-prod-sync', 'ERP Sync Product', $1, 'ambient', $2, $3, 200, 'published')`,
		categoryID, supplierID, unitID)
}

func TestERP_WebhookIngestion(t *testing.T) {
	env := setupERPEnv(t)
	_, _ = env.db.Pool.Exec(context.Background(), "TRUNCATE TABLE erp.erp_webhook_event RESTART IDENTITY CASCADE")

	body := []byte(`{"event_id":"erp_evt_001","type":"stock.update"}`)
	resp, respBody := erpWebhookDo(t, env, "stock", body, "test-erp-secret")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook: expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}
	var ev erp_schema.WebhookEventResponse
	_ = json.Unmarshal(respBody, &ev)
	if ev.Status != "applied" {
		t.Errorf("expected applied, got %s", ev.Status)
	}

	resp, respBody = erpWebhookDo(t, env, "stock", body, "test-erp-secret")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("replay webhook: expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	resp, respBody = erpWebhookDo(t, env, "stock", body, "wrong-secret")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("bad secret: expected 401, got %d: %s", resp.StatusCode, string(respBody))
	}

	oldTs := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
	mac := hmac.New(sha256.New, []byte("test-erp-secret"))
	mac.Write([]byte(oldTs + "."))
	mac.Write(body)
	staleSig := hex.EncodeToString(mac.Sum(nil))
	req, _ := http.NewRequest(http.MethodPost, env.srv.URL+"/webhooks/erp/stock", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Timestamp", oldTs)
	req.Header.Set("X-Webhook-Signature", staleSig)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("stale webhook: expected 401, got %d", resp.StatusCode)
	}
}

func TestERP_TriggerCatalogSync(t *testing.T) {
	env := setupERPEnv(t)
	_, _ = env.db.Pool.Exec(context.Background(), "TRUNCATE TABLE erp.sync_job, erp.sync_drift RESTART IDENTITY CASCADE")
	seedProduct(t, env)

	resp, body := erpDo(t, env, http.MethodPost, "/api/v1/admin/erp/sync/catalog", "")
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("trigger catalog sync: expected 202, got %d: %s", resp.StatusCode, string(body))
	}
	var job erp_schema.SyncJobDetailResponse
	_ = json.Unmarshal(body, &job)
	if job.Code == "" {
		t.Error("expected non-empty job code")
	}
	if job.Status != "completed" {
		t.Errorf("expected completed, got %s", job.Status)
	}
	if job.JobType != "catalog" {
		t.Errorf("expected catalog, got %s", job.JobType)
	}
	if job.TotalItems < 1 {
		t.Errorf("expected at least 1 item, got %d", job.TotalItems)
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Error("expected Location header")
	}
}

func TestERP_TriggerStockSync(t *testing.T) {
	env := setupERPEnv(t)
	_, _ = env.db.Pool.Exec(context.Background(), "TRUNCATE TABLE erp.sync_job, erp.sync_drift RESTART IDENTITY CASCADE")

	resp, body := erpDo(t, env, http.MethodPost, "/api/v1/admin/erp/sync/stock", "")
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("trigger stock sync: expected 202, got %d: %s", resp.StatusCode, string(body))
	}
	var job erp_schema.SyncJobDetailResponse
	_ = json.Unmarshal(body, &job)
	if job.JobType != "stock" {
		t.Errorf("expected stock, got %s", job.JobType)
	}
}

func TestERP_GetSyncJob(t *testing.T) {
	env := setupERPEnv(t)
	_, _ = env.db.Pool.Exec(context.Background(), "TRUNCATE TABLE erp.sync_job, erp.sync_drift RESTART IDENTITY CASCADE")
	seedProduct(t, env)

	resp, body := erpDo(t, env, http.MethodPost, "/api/v1/admin/erp/sync/catalog", "")
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("trigger: expected 202, got %d: %s", resp.StatusCode, string(body))
	}
	var created erp_schema.SyncJobDetailResponse
	_ = json.Unmarshal(body, &created)

	resp, body = erpDo(t, env, http.MethodGet, "/api/v1/admin/erp/sync/"+created.Code, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get job: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var fetched erp_schema.SyncJobDetailResponse
	_ = json.Unmarshal(body, &fetched)
	if fetched.Code != created.Code {
		t.Errorf("expected code %s, got %s", created.Code, fetched.Code)
	}

	resp, _ = erpDo(t, env, http.MethodGet, "/api/v1/admin/erp/sync/sj_nonexistent", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing job: expected 404, got %d", resp.StatusCode)
	}
}

func TestERP_DispatchOrder(t *testing.T) {
	env := setupERPEnv(t)
	_, _ = env.db.Pool.Exec(context.Background(), "TRUNCATE TABLE erp.order_dispatch RESTART IDENTITY CASCADE")
	orderID := seedOrder(t, env)

	resp, body := erpDo(t, env, http.MethodPost, fmt.Sprintf("/api/v1/admin/erp/orders/%d/dispatch", orderID), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dispatch: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var dispatch erp_schema.OrderDispatchResponse
	_ = json.Unmarshal(body, &dispatch)
	if dispatch.Status != "dispatched" {
		t.Errorf("expected dispatched, got %s", dispatch.Status)
	}
	if dispatch.OrderID != orderID {
		t.Errorf("expected order_id %d, got %d", orderID, dispatch.OrderID)
	}

	resp, body = erpDo(t, env, http.MethodPost, fmt.Sprintf("/api/v1/admin/erp/orders/%d/dispatch", orderID), "")
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("double dispatch: expected 409, got %d: %s", resp.StatusCode, string(body))
	}

	resp, _ = erpDo(t, env, http.MethodPost, "/api/v1/admin/erp/orders/999999/dispatch", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing order: expected 404, got %d", resp.StatusCode)
	}

	resp, _ = erpDo(t, env, http.MethodPost, "/api/v1/admin/erp/orders/abc/dispatch", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad id: expected 400, got %d", resp.StatusCode)
	}
}

func TestERP_WebhookDedup(t *testing.T) {
	env := setupERPEnv(t)
	_, _ = env.db.Pool.Exec(context.Background(), "TRUNCATE TABLE erp.erp_webhook_event RESTART IDENTITY CASCADE")

	body := []byte(`{"event_id":"erp_evt_dedup_001","type":"catalog.update"}`)

	resp1, body1 := erpWebhookDo(t, env, "catalog", body, "test-erp-secret")
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first: expected 200, got %d: %s", resp1.StatusCode, string(body1))
	}

	resp2, body2 := erpWebhookDo(t, env, "catalog", body, "test-erp-secret")
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("replay: expected 200, got %d: %s", resp2.StatusCode, string(body2))
	}

	var count int
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM erp.erp_webhook_event WHERE provider_event_id = 'erp_evt_dedup_001'`).Scan(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 row, got %d", count)
	}
}
