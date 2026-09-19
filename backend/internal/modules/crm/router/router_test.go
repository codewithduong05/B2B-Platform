package router_test

// TASK-010 CRM tests: partner registry, referral codes, lead lifecycle
// (submit → assign → contact → convert), attribution, RBAC, races.

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
	"github.com/atlas-platform/backend/internal/modules/crm"
	crm_router "github.com/atlas-platform/backend/internal/modules/crm/router"
	crm_schema "github.com/atlas-platform/backend/internal/modules/crm/schema"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const crmTestDBLockKey = 1946819412

func TestMain(m *testing.M) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err == nil {
		rawDB, err := sql.Open("pgx", cfg.PostgresDSN())
		if err == nil && rawDB != nil {
			_, _ = rawDB.Exec("SELECT pg_advisory_lock($1)", int64(crmTestDBLockKey))
			_, _ = rawDB.Exec("DROP SCHEMA IF EXISTS catalog CASCADE; DROP SCHEMA IF EXISTS inventory CASCADE; DROP SCHEMA IF EXISTS identity CASCADE; DROP SCHEMA IF EXISTS pricing CASCADE; DROP SCHEMA IF EXISTS commerce CASCADE; DROP SCHEMA IF EXISTS payments CASCADE; DROP SCHEMA IF EXISTS promotions CASCADE; DROP SCHEMA IF EXISTS crm CASCADE; DROP TABLE IF EXISTS schema_migrations CASCADE; DROP TYPE IF EXISTS catalog_handling_class_type CASCADE;")

			if err := database.RunMigrations(ctx, &cfg.Postgres, "file://../../../../migrations"); err != nil {
				fmt.Printf("TestMain migration error: %v\n", err)
			}

			code := m.Run()
			_, _ = rawDB.Exec("SELECT pg_advisory_unlock($1)", int64(crmTestDBLockKey))
			rawDB.Close()
			os.Exit(code)
		}
	}
	os.Exit(m.Run())
}

type crmEnv struct {
	db  *database.DB
	srv *httptest.Server
}

func crmNop(next http.Handler) http.Handler { return next }

func crmPrincipal(id int64, staff map[int64]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(crm_router.WithPrincipalID(r.Context(), id)))
		})
	}
}

func crmStaff(staff map[int64]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !staff[crm_router.PrincipalIDFromContext(r.Context())] {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, `{"detail":"staff only","code":"forbidden"}`)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func setupCrmEnv(t *testing.T, id int64, staff map[int64]bool) *crmEnv {
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
	svc := crm.NewService(db)
	rt := crm.New(svc)
	rt.RegisterRoutes(crmPrincipal(id, staff), crmStaff(staff))

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Mount("/api/v1", rt.ChiRouter())
	server := httptest.NewServer(mux)

	t.Cleanup(func() {
		server.Close()
		db.Close()
	})
	return &crmEnv{db: db, srv: server}
}

var crmCounter int64

func crmSuffix() string {
	c := atomic.AddInt64(&crmCounter, 1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano()%1000000, c)
}

func crmDo(t *testing.T, env *crmEnv, method, path, body string) (*http.Response, []byte) {
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

func crmBuyer(t *testing.T, env *crmEnv) int64 {
	t.Helper()
	suffix := crmSuffix()
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

func mkPartner(t *testing.T, env *crmEnv, name string) (int64, string) {
	t.Helper()
	resp, body := crmDo(t, env, http.MethodPost, "/api/v1/admin/partners",
		fmt.Sprintf(`{"name":%q}`, name))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("partner: %d: %s", resp.StatusCode, string(body))
	}
	var out crm_schema.PartnerResponse
	_ = json.Unmarshal(body, &out)
	var id int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT id FROM crm.partner WHERE code=$1`, out.Code).Scan(&id)
	return id, out.Slug
}

func mkReferral(t *testing.T, env *crmEnv, partnerID int64, code string) string {
	t.Helper()
	body := fmt.Sprintf(`{"partner_id":%d}`, partnerID)
	if code != "" {
		body = fmt.Sprintf(`{"partner_id":%d,"code":%q}`, partnerID, code)
	}
	resp, rbody := crmDo(t, env, http.MethodPost, "/api/v1/admin/referrals", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("referral: %d: %s", resp.StatusCode, string(rbody))
	}
	var out crm_schema.ReferralResponse
	_ = json.Unmarshal(rbody, &out)
	return out.Code
}

func TestCRM_PartnerReferral(t *testing.T) {
	env := setupCrmEnv(t, 0, map[int64]bool{})
	staffID := crmBuyer(t, env)
	staff := map[int64]bool{staffID: true}
	senv := setupCrmEnv(t, staffID, staff)

	// Create partner (slug auto-derived) + duplicate name gets suffixed slug.
	pid, slug := mkPartner(t, senv, "Acme Foods")
	if slug == "" {
		t.Fatalf("empty slug")
	}
	pid2, slug2 := mkPartner(t, senv, "Acme Foods")
	if pid == pid2 || slug == slug2 {
		t.Errorf("expected distinct partners/slugs, got %d/%d %q/%q", pid, pid2, slug, slug2)
	}

	// Empty name → 400.
	resp, body := crmDo(t, senv, http.MethodPost, "/api/v1/admin/partners", `{"name":""}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty name: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// Referral with explicit code; duplicate → 409.
	rc := mkReferral(t, senv, pid, "ACME-"+crmSuffix())
	resp, _ = crmDo(t, senv, http.MethodPost, "/api/v1/admin/referrals",
		fmt.Sprintf(`{"partner_id":%d,"code":%q}`, pid, rc))
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("dup referral: expected 409, got %d", resp.StatusCode)
	}
	// Referral for missing partner → 404.
	resp, _ = crmDo(t, senv, http.MethodPost, "/api/v1/admin/referrals", `{"partner_id":999999999}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing partner: expected 404, got %d", resp.StatusCode)
	}

	// Track resolves code and slug; unknown → 404.
	for path, want := range map[string]int{
		"/api/v1/partners/track?ref=" + rc:   http.StatusOK,
		"/api/v1/partners/track?ref=" + slug: http.StatusOK,
		"/api/v1/partners/track?ref=nope":    http.StatusNotFound,
		"/api/v1/partners/track":             http.StatusBadRequest,
	} {
		resp, _ := crmDo(t, senv, http.MethodGet, path, "")
		if resp.StatusCode != want {
			t.Errorf("GET %s: expected %d, got %d", path, want, resp.StatusCode)
		}
	}

	// Referral list shows uses (0 so far).
	resp, body = crmDo(t, senv, http.MethodGet, "/api/v1/admin/referrals", "")
	var envelope struct {
		Items []crm_schema.ReferralResponse `json:"items"`
		Total int                           `json:"total"`
	}
	_ = json.Unmarshal(body, &envelope)
	if envelope.Total < 1 {
		t.Errorf("expected referrals: %+v", envelope)
	}
	_ = resp
}

func TestCRM_LeadLifecycle(t *testing.T) {
	env := setupCrmEnv(t, 0, map[int64]bool{})
	staffID := crmBuyer(t, env)
	buyerID := crmBuyer(t, env)
	staff := map[int64]bool{staffID: true}
	senv := setupCrmEnv(t, staffID, staff)
	pid, _ := mkPartner(t, senv, "Lead Source")
	rc := mkReferral(t, senv, pid, "")

	submit := func(body string, want int) crm_schema.LeadResponse {
		t.Helper()
		resp, rbody := crmDo(t, senv, http.MethodPost, "/api/v1/leads", body)
		if resp.StatusCode != want {
			t.Fatalf("submit: expected %d, got %d: %s", want, resp.StatusCode, string(rbody))
		}
		if want != http.StatusCreated {
			return crm_schema.LeadResponse{}
		}
		var out crm_schema.LeadResponse
		_ = json.Unmarshal(rbody, &out)
		if loc := resp.Header.Get("Location"); loc == "" {
			t.Errorf("expected Location on 201")
		}
		return out
	}

	// Attributed submit.
	lead := submit(fmt.Sprintf(
		`{"contact_name":"Chef","business_name":"Bistro","email":"c@b.local","partner_ref":%q}`, rc), http.StatusCreated)
	if lead.Status != "new" || lead.PartnerID == nil || *lead.PartnerID != pid || lead.ReferralID == nil {
		t.Fatalf("attribution wrong: %+v", lead)
	}

	// Uses incremented.
	resp, body := crmDo(t, senv, http.MethodGet, "/api/v1/admin/referrals", "")
	var renv struct {
		Items []crm_schema.ReferralResponse `json:"items"`
	}
	_ = json.Unmarshal(body, &renv)
	_ = resp
	uses := -1
	for _, r := range renv.Items {
		if r.Code == rc {
			uses = r.Uses
		}
	}
	if uses != 1 {
		t.Errorf("expected 1 use, got %d", uses)
	}

	// Missing names → 400. Unknown ref → 422 with reason.
	submit(`{"contact_name":"","business_name":"x"}`, http.StatusBadRequest)
	submit(`{"contact_name":"C","business_name":"B","partner_ref":"nope"}`, http.StatusUnprocessableEntity)

	var leadID int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT id FROM crm.lead WHERE code=$1`, lead.Code).Scan(&leadID)

	// Skip states: new → contacted is invalid.
	resp, _ = crmDo(t, senv, http.MethodPatch, fmt.Sprintf("/api/v1/admin/leads/%d", leadID), `{"status":"contacted"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("skip state: expected 422, got %d", resp.StatusCode)
	}

	// Assign (new → assigned) + double assign → 422.
	resp, body = crmDo(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/leads/%d/assign", leadID),
		fmt.Sprintf(`{"assignee":%d}`, staffID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("assign: %d: %s", resp.StatusCode, string(body))
	}
	var assigned crm_schema.LeadResponse
	_ = json.Unmarshal(body, &assigned)
	if assigned.Status != "assigned" || assigned.AssignedTo == nil || *assigned.AssignedTo != staffID {
		t.Errorf("bad assign: %+v", assigned)
	}
	resp, _ = crmDo(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/leads/%d/assign", leadID),
		fmt.Sprintf(`{"assignee":%d}`, staffID))
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("re-assign: expected 422, got %d", resp.StatusCode)
	}

	// Contact, then convert without buyer → 400-class; with buyer → converted.
	resp, _ = crmDo(t, senv, http.MethodPatch, fmt.Sprintf("/api/v1/admin/leads/%d", leadID), `{"status":"contacted"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("contact: %d", resp.StatusCode)
	}
	resp, _ = crmDo(t, senv, http.MethodPatch, fmt.Sprintf("/api/v1/admin/leads/%d", leadID), `{"status":"converted"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("convert w/o buyer: expected 400, got %d", resp.StatusCode)
	}
	resp, body = crmDo(t, senv, http.MethodPatch, fmt.Sprintf("/api/v1/admin/leads/%d", leadID),
		fmt.Sprintf(`{"status":"converted","buyer_id":%d}`, buyerID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("convert: %d: %s", resp.StatusCode, string(body))
	}
	var converted crm_schema.LeadResponse
	_ = json.Unmarshal(body, &converted)
	if converted.Status != "converted" || converted.BuyerID == nil || *converted.BuyerID != buyerID {
		t.Errorf("bad convert: %+v", converted)
	}

	// Terminal: further moves rejected. Convert with bogus buyer → 400.
	resp, _ = crmDo(t, senv, http.MethodPatch, fmt.Sprintf("/api/v1/admin/leads/%d", leadID), `{"status":"closed"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("terminal move: expected 422, got %d", resp.StatusCode)
	}
	lead2 := submit(`{"contact_name":"C2","business_name":"B2"}`, http.StatusCreated)
	var lead2ID int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT id FROM crm.lead WHERE code=$1`, lead2.Code).Scan(&lead2ID)
	for _, next := range []string{"assigned", "contacted"} {
		resp, _ = crmDo(t, senv, http.MethodPatch, fmt.Sprintf("/api/v1/admin/leads/%d", lead2ID),
			fmt.Sprintf(`{"status":%q}`, next))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("advance %s: %d", next, resp.StatusCode)
		}
	}
	resp, _ = crmDo(t, senv, http.MethodPatch, fmt.Sprintf("/api/v1/admin/leads/%d", lead2ID),
		`{"status":"converted","buyer_id":999999999}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bogus buyer: expected 400, got %d", resp.StatusCode)
	}

	// List + status filter.
	resp, body = crmDo(t, senv, http.MethodGet, "/api/v1/admin/leads?status=converted", "")
	var lenv struct {
		Items []crm_schema.LeadResponse `json:"items"`
		Total int                       `json:"total"`
	}
	_ = json.Unmarshal(body, &lenv)
	if lenv.Total < 1 || len(lenv.Items) != lenv.Total {
		t.Errorf("filter broken: %+v", lenv)
	}
	_ = resp
}

func TestCRM_RBAC(t *testing.T) {
	env := setupCrmEnv(t, 0, map[int64]bool{})
	otherID := crmBuyer(t, env)
	otherEnv := setupCrmEnv(t, otherID, map[int64]bool{})

	// Public endpoints reachable without staff.
	resp, _ := crmDo(t, otherEnv, http.MethodPost, "/api/v1/leads",
		`{"contact_name":"C","business_name":"B"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("public submit: expected 201, got %d", resp.StatusCode)
	}
	resp, _ = crmDo(t, otherEnv, http.MethodGet, "/api/v1/partners/track?ref=nope", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("public track: expected 404, got %d", resp.StatusCode)
	}

	// Every admin path denies non-staff with 403.
	for _, p := range [][2]string{
		{http.MethodGet, "/api/v1/admin/leads"},
		{http.MethodPost, "/api/v1/admin/leads/1/assign"},
		{http.MethodPatch, "/api/v1/admin/leads/1"},
		{http.MethodGet, "/api/v1/admin/referrals"},
		{http.MethodPost, "/api/v1/admin/referrals"},
		{http.MethodGet, "/api/v1/admin/partners"},
		{http.MethodPost, "/api/v1/admin/partners"},
	} {
		resp, _ := crmDo(t, otherEnv, p[0], p[1], `{"assignee":1}`)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s: expected 403, got %d", p[0], p[1], resp.StatusCode)
		}
	}
}

func TestCRM_ConcurrentSubmits(t *testing.T) {
	env := setupCrmEnv(t, 0, map[int64]bool{})
	staffID := crmBuyer(t, env)
	staff := map[int64]bool{staffID: true}
	senv := setupCrmEnv(t, staffID, staff)
	pid, _ := mkPartner(t, senv, "Race Partner")
	rc := mkReferral(t, senv, pid, "")

	// Concurrent attributed submits: all succeed, uses == N exactly.
	const n = 10
	var codes [n]int
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			resp, _ := crmDo(t, senv, http.MethodPost, "/api/v1/leads",
				fmt.Sprintf(`{"contact_name":"C%d","business_name":"B%d","partner_ref":%q}`, i, i, rc))
			codes[i] = resp.StatusCode
		}(i)
	}
	close(start)
	wg.Wait()
	for i, c := range codes {
		if c != http.StatusCreated {
			t.Errorf("req %d: expected 201, got %d", i, c)
		}
	}
	var uses int
	_ = env.db.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM crm.lead WHERE referral_id=(SELECT id FROM crm.referral_code WHERE code=$1)`, rc).Scan(&uses)
	if uses != n {
		t.Errorf("expected %d attributed leads, got %d", n, uses)
	}

	// Concurrent assigns on one lead: exactly one wins.
	resp, body := crmDo(t, senv, http.MethodPost, "/api/v1/leads",
		`{"contact_name":"R","business_name":"RB"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("submit: %d", resp.StatusCode)
	}
	var lead crm_schema.LeadResponse
	_ = json.Unmarshal(body, &lead)
	var leadID int64
	_ = env.db.Pool.QueryRow(context.Background(), `SELECT id FROM crm.lead WHERE code=$1`, lead.Code).Scan(&leadID)
	var results [n]int
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			resp, _ := crmDo(t, senv, http.MethodPost, fmt.Sprintf("/api/v1/admin/leads/%d/assign", leadID),
				fmt.Sprintf(`{"assignee":%d}`, staffID))
			results[i] = resp.StatusCode
		}(i)
	}
	// Note: start already closed; second wave runs immediately.
	wg.Wait()
	ok := 0
	for _, c := range results {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusUnprocessableEntity:
		default:
			t.Errorf("unexpected %d", c)
		}
	}
	if ok != 1 {
		t.Errorf("expected exactly 1 assign, got %d (%v)", ok, results)
	}
}
