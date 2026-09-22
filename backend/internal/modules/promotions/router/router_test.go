package router_test

// TASK-009 promotions tests: admin CRUD/publish/list, buyer active
// visibility, reports. Cart/checkout voucher flows live in the commerce
// package suite. Shared advisory lock key: test packages share one DB.

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
	"github.com/atlas-platform/backend/internal/modules/promotions"
	promotions_router "github.com/atlas-platform/backend/internal/modules/promotions/router"
	promotions_schema "github.com/atlas-platform/backend/internal/modules/promotions/schema"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const promotionsTestDBLockKey = 1946819412

func TestMain(m *testing.M) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err == nil {
		rawDB, err := sql.Open("pgx", cfg.PostgresDSN())
		if err == nil && rawDB != nil {
			_, _ = rawDB.Exec("SELECT pg_advisory_lock($1)", int64(promotionsTestDBLockKey))
			_, _ = rawDB.Exec("DROP SCHEMA IF EXISTS catalog CASCADE; DROP SCHEMA IF EXISTS inventory CASCADE; DROP SCHEMA IF EXISTS identity CASCADE; DROP SCHEMA IF EXISTS pricing CASCADE; DROP SCHEMA IF EXISTS commerce CASCADE; DROP SCHEMA IF EXISTS payments CASCADE; DROP SCHEMA IF EXISTS promotions CASCADE; DROP SCHEMA IF EXISTS crm CASCADE; DROP SCHEMA IF EXISTS suppliers CASCADE; DROP SCHEMA IF EXISTS cms CASCADE; DROP SCHEMA IF EXISTS reports CASCADE; DROP SCHEMA IF EXISTS erp CASCADE; DROP SCHEMA IF EXISTS platform CASCADE; DROP SCHEMA IF EXISTS ai CASCADE; DROP SCHEMA IF EXISTS analytics CASCADE; DROP TABLE IF EXISTS schema_migrations CASCADE; DROP TYPE IF EXISTS catalog_handling_class_type CASCADE;")

			if err := database.RunMigrations(ctx, &cfg.Postgres, "file://../../../../migrations"); err != nil {
				fmt.Printf("TestMain migration error: %v\n", err)
			}

			code := m.Run()
			_, _ = rawDB.Exec("SELECT pg_advisory_unlock($1)", int64(promotionsTestDBLockKey))
			rawDB.Close()
			os.Exit(code)
		}
	}
	os.Exit(m.Run())
}

type promoEnv struct {
	db  *database.DB
	srv *httptest.Server
}

func promoNop(next http.Handler) http.Handler { return next }

func promoPrincipal(id int64, staff map[int64]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(promotions_router.WithPrincipalID(r.Context(), id)))
		})
	}
}

func promoStaff(staff map[int64]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !staff[promotions_router.PrincipalIDFromContext(r.Context())] {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, `{"detail":"staff only","code":"forbidden"}`)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func setupPromoEnv(t *testing.T, id int64, staff map[int64]bool) *promoEnv {
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
	svc := promotions.NewService(db, nil)
	rt := promotions.New(svc)
	rt.RegisterRoutes(promoPrincipal(id, staff), promoStaff(staff))

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Mount("/api/v1", rt.ChiRouter())
	server := httptest.NewServer(mux)

	t.Cleanup(func() {
		server.Close()
		db.Close()
	})
	return &promoEnv{db: db, srv: server}
}

var promoCounter int64

func promoSuffix() string {
	c := atomic.AddInt64(&promoCounter, 1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano()%1000000, c)
}

func promoDo(t *testing.T, env *promoEnv, method, path, body string) (*http.Response, []byte) {
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

func promoBuyer(t *testing.T, env *promoEnv) int64 {
	t.Helper()
	suffix := promoSuffix()
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

func createPromo(t *testing.T, env *promoEnv, body string, want int) promotions_schema.PromotionResponse {
	t.Helper()
	resp, rbody := promoDo(t, env, http.MethodPost, "/api/v1/admin/promotions", body)
	if resp.StatusCode != want {
		t.Fatalf("create promotion: expected %d, got %d: %s", want, resp.StatusCode, string(rbody))
	}
	if want != http.StatusCreated {
		return promotions_schema.PromotionResponse{}
	}
	var out promotions_schema.PromotionResponse
	_ = json.Unmarshal(rbody, &out)
	if loc := resp.Header.Get("Location"); loc == "" {
		t.Errorf("expected Location header on 201")
	}
	return out
}

func TestPromotions_AdminCRUD(t *testing.T) {
	env := setupPromoEnv(t, 0, map[int64]bool{})
	staffID := promoBuyer(t, env)
	staff := map[int64]bool{staffID: true}
	senv := setupPromoEnv(t, staffID, staff)

	// Invalid: bad kind, percent > 100, empty code, inverted window.
	for _, body := range []string{
		fmt.Sprintf(`{"code":"%s","name":"x","kind":"bogus","value_minor":10}`, "p1-"+promoSuffix()),
		fmt.Sprintf(`{"code":"%s","name":"x","kind":"percent","value_minor":101}`, "p2-"+promoSuffix()),
		`{"code":"","name":"x","kind":"fixed","value_minor":10}`,
		fmt.Sprintf(`{"code":"%s","name":"x","kind":"fixed","value_minor":10,"valid_from":"2026-02-01T00:00:00Z","valid_to":"2026-01-01T00:00:00Z"}`, "p3-"+promoSuffix()),
	} {
		createPromo(t, senv, body, http.StatusBadRequest)
	}

	p := createPromo(t, senv, fmt.Sprintf(`{"code":"%s","name":"Ten off","kind":"percent","value_minor":10}`, "p4-"+promoSuffix()), http.StatusCreated)
	if p.Status != "draft" {
		t.Fatalf("expected draft, got %+v", p)
	}

	// Invalid transition straight to archived? No — draft->archived allowed.
	// Unknown id → 404.
	resp, _ := promoDo(t, senv, http.MethodPatch, "/api/v1/admin/promotions/999999999", `{"name":"x"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing promotion: expected 404, got %d", resp.StatusCode)
	}
	var promoID int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT id FROM promotions.promotion WHERE code=$1`, p.Code).Scan(&promoID)

	// Patch name + budget.
	resp, rbody := promoDo(t, senv, http.MethodPatch, fmt.Sprintf("/api/v1/admin/promotions/%d", promoID), `{"name":"Ten off v2","max_redemptions":5}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch: %d: %s", resp.StatusCode, string(rbody))
	}
	var patched promotions_schema.PromotionResponse
	_ = json.Unmarshal(rbody, &patched)
	if patched.Name != "Ten off v2" || patched.MaxRedemptions == nil || *patched.MaxRedemptions != 5 {
		t.Errorf("unexpected patched: %+v", patched)
	}

	// Publish, then publish again → 422. Draft→published→archived ok.
	resp, _ = promoDo(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/promotions/%d/publish", promoID), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("publish: %d", resp.StatusCode)
	}
	resp, _ = promoDo(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/promotions/%d/publish", promoID), "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("re-publish: expected 422, got %d", resp.StatusCode)
	}

	// Malformed id → 400.
	resp, _ = promoDo(t, senv, http.MethodPatch, "/api/v1/admin/promotions/abc", `{"name":"x"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad id: expected 400, got %d", resp.StatusCode)
	}
	// Admin list envelope.
	resp, rbody = promoDo(t, senv, http.MethodGet, "/api/v1/admin/promotions", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin list: %d", resp.StatusCode)
	}
	var envelope struct {
		Items []promotions_schema.PromotionResponse `json:"items"`
		Total int                                   `json:"total"`
	}
	_ = json.Unmarshal(rbody, &envelope)
	if envelope.Total < 1 || len(envelope.Items) < 1 {
		t.Errorf("unexpected envelope: %+v", envelope)
	}
}

func TestPromotions_ActiveAndReports(t *testing.T) {
	env := setupPromoEnv(t, 0, map[int64]bool{})
	buyerID := promoBuyer(t, env)
	staffID := promoBuyer(t, env)
	staff := map[int64]bool{staffID: true}
	buyerEnv := setupPromoEnv(t, buyerID, staff)
	senv := setupPromoEnv(t, staffID, staff)

	pub := createPromo(t, senv, fmt.Sprintf(`{"code":"%s","name":"Live","kind":"fixed","value_minor":500}`, "live-"+promoSuffix()), http.StatusCreated)
	var pubID int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT id FROM promotions.promotion WHERE code=$1`, pub.Code).Scan(&pubID)
	resp, _ := promoDo(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/promotions/%d/publish", pubID), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("publish: %d", resp.StatusCode)
	}
	draft := createPromo(t, senv, fmt.Sprintf(`{"code":"%s","name":"Draft","kind":"fixed","value_minor":500}`, "draft-"+promoSuffix()), http.StatusCreated)

	// Buyer sees only the published one, flagged redeemable.
	resp, rbody := promoDo(t, buyerEnv, http.MethodGet, "/api/v1/promotions/active", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("active: %d: %s", resp.StatusCode, string(rbody))
	}
	var active []promotions_schema.PromotionResponse
	_ = json.Unmarshal(rbody, &active)
	seen := map[string]bool{}
	for _, p := range active {
		seen[p.Code] = true
		if p.Code == pub.Code && (p.Redeemable == nil || !*p.Redeemable) {
			t.Errorf("published promo should be redeemable: %+v", p)
		}
	}
	if !seen[pub.Code] || seen[draft.Code] {
		t.Errorf("visibility wrong: %+v", seen)
	}

	// Reports row exists with zero cost.
	resp, rbody = promoDo(t, senv, http.MethodGet, "/api/v1/admin/promotions/reports", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reports: %d: %s", resp.StatusCode, string(rbody))
	}
	var rows []promotions_schema.ReportRow
	_ = json.Unmarshal(rbody, &rows)
	found := false
	for _, r := range rows {
		if r.Code == pub.Code {
			found = true
			if r.RedeemedCount != 0 || r.TotalCostMinor != 0 {
				t.Errorf("unexpected report row: %+v", r)
			}
		}
	}
	if !found {
		t.Errorf("report missing promotion: %+v", rows)
	}

	// Buyer on admin paths → 403.
	resp, _ = promoDo(t, buyerEnv, http.MethodGet, "/api/v1/admin/promotions", "")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("buyer admin list: expected 403, got %d", resp.StatusCode)
	}
	resp, _ = promoDo(t, buyerEnv, http.MethodPost, "/api/v1/admin/promotions", `{"code":"x","name":"x","kind":"fixed","value_minor":1}`)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("buyer admin create: expected 403, got %d", resp.StatusCode)
	}
}
