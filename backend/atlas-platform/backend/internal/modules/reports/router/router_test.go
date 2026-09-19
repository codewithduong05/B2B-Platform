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
	"testing"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/reports"
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
			_, _ = rawDB.Exec("DROP SCHEMA IF EXISTS catalog CASCADE; DROP SCHEMA IF EXISTS inventory CASCADE; DROP SCHEMA IF EXISTS identity CASCADE; DROP SCHEMA IF EXISTS pricing CASCADE; DROP SCHEMA IF EXISTS commerce CASCADE; DROP SCHEMA IF EXISTS payments CASCADE; DROP SCHEMA IF EXISTS promotions CASCADE; DROP SCHEMA IF EXISTS crm CASCADE; DROP SCHEMA IF EXISTS suppliers CASCADE; DROP SCHEMA IF EXISTS cms CASCADE; DROP SCHEMA IF EXISTS reports CASCADE; DROP TABLE IF EXISTS schema_migrations CASCADE; DROP TYPE IF EXISTS catalog_handling_class_type CASCADE;")

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

type reportsEnv struct {
	db  *database.DB
	srv *httptest.Server
}

func setupReportsEnv(t *testing.T) *reportsEnv {
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
		func(next http.Handler) http.Handler { return next },
		func(next http.Handler) http.Handler { return next },
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

func repDo(t *testing.T, env *reportsEnv, method, path, body string) (*http.Response, []byte) {
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

func TestReports_AllEndpoints(t *testing.T) {
	env := setupReportsEnv(t)
	_, _ = env.db.Pool.Exec(context.Background(), "TRUNCATE TABLE reports.export_job RESTART IDENTITY CASCADE")

	endpoints := []string{
		"/api/v1/admin/reports/sales",
		"/api/v1/admin/reports/sales?group_by=supplier",
		"/api/v1/admin/reports/sales?group_by=category",
		"/api/v1/admin/reports/buyers",
		"/api/v1/admin/reports/products",
		"/api/v1/admin/reports/suppliers",
		"/api/v1/admin/reports/promotions",
		"/api/v1/admin/reports/operations",
		"/api/v1/admin/reports/finance",
		"/api/v1/admin/reports/exports",
	}

	for _, ep := range endpoints {
		resp, body := repDo(t, env, http.MethodGet, ep, "")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s: expected 200, got %d: %s", ep, resp.StatusCode, string(body))
		}
	}

	// Test POST /admin/reports/exports
	resp, body := repDo(t, env, http.MethodPost, "/api/v1/admin/reports/exports", `{"report_type":"sales","format":"csv"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST exports: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var ej reports_schema.ExportJobResponse
	_ = json.Unmarshal(body, &ej)
	if ej.Status != "completed" || ej.ReportType != "sales" {
		t.Errorf("unexpected export job response: %+v", ej)
	}
}
