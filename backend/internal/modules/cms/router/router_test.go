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
	"time"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/cms"
	cms_schema "github.com/atlas-platform/backend/internal/modules/cms/schema"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const cmsTestDBLockKey = 1946819412

func TestMain(m *testing.M) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err == nil {
		rawDB, err := sql.Open("pgx", cfg.PostgresDSN())
		if err == nil && rawDB != nil {
			_, _ = rawDB.Exec("SELECT pg_advisory_lock($1)", int64(cmsTestDBLockKey))
			_, _ = rawDB.Exec("DROP SCHEMA IF EXISTS catalog CASCADE; DROP SCHEMA IF EXISTS inventory CASCADE; DROP SCHEMA IF EXISTS identity CASCADE; DROP SCHEMA IF EXISTS pricing CASCADE; DROP SCHEMA IF EXISTS commerce CASCADE; DROP SCHEMA IF EXISTS payments CASCADE; DROP SCHEMA IF EXISTS promotions CASCADE; DROP SCHEMA IF EXISTS crm CASCADE; DROP SCHEMA IF EXISTS suppliers CASCADE; DROP SCHEMA IF EXISTS cms CASCADE; DROP SCHEMA IF EXISTS reports CASCADE; DROP SCHEMA IF EXISTS erp CASCADE; DROP TABLE IF EXISTS schema_migrations CASCADE; DROP TYPE IF EXISTS catalog_handling_class_type CASCADE;")

			if err := database.RunMigrations(ctx, &cfg.Postgres, "file://../../../../migrations"); err != nil {
				fmt.Printf("TestMain migration error: %v\n", err)
			}

			code := m.Run()
			_, _ = rawDB.Exec("SELECT pg_advisory_unlock($1)", int64(cmsTestDBLockKey))
			rawDB.Close()
			os.Exit(code)
		}
	}
	os.Exit(m.Run())
}

type cmsEnv struct {
	db  *database.DB
	srv *httptest.Server
}

func setupCMSEnv(t *testing.T) *cmsEnv {
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
	svc := cms.NewService(db)
	rt := cms.New(svc)
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
	return &cmsEnv{db: db, srv: server}
}

func cmsDo(t *testing.T, env *cmsEnv, method, path, body string) (*http.Response, []byte) {
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

func TestCMS_EndToEnd(t *testing.T) {
	env := setupCMSEnv(t)
	_, _ = env.db.Pool.Exec(context.Background(), "TRUNCATE TABLE cms.article, cms.page, cms.faq, cms.banner, cms.menu_item, cms.setting RESTART IDENTITY CASCADE")
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	// 1. Create article draft
	resp, body := cmsDo(t, env, http.MethodPost, "/api/v1/admin/cms/articles", fmt.Sprintf(`{"title":"Hello CMS %s","body":"Draft content"}`, suffix))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create article: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var art cms_schema.ArticleResponse
	_ = json.Unmarshal(body, &art)

	// Public articles should not show draft articles
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/cms/articles", "")
	var listEnvelope struct {
		Items []cms_schema.ArticleResponse `json:"items"`
		Total int                          `json:"total"`
	}
	_ = json.Unmarshal(body, &listEnvelope)
	if listEnvelope.Total != 0 {
		t.Errorf("expected 0 published articles, got %d", listEnvelope.Total)
	}

	// 2. Publish article
	resp, body = cmsDo(t, env, http.MethodPost, fmt.Sprintf("/api/v1/admin/cms/articles/%d/publish", art.ID), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("publish article: expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	// Public articles now show published article
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/cms/articles", "")
	_ = json.Unmarshal(body, &listEnvelope)
	if listEnvelope.Total != 1 {
		t.Errorf("expected 1 published article, got %d", listEnvelope.Total)
	}

	// 3. Get article by slug
	resp, body = cmsDo(t, env, http.MethodGet, fmt.Sprintf("/api/v1/cms/articles/%s", art.Slug), "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("get article by slug: expected 200, got %d", resp.StatusCode)
	}

	// 4. Upsert page & get page by slug
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/admin/cms/pages", fmt.Sprintf(`{"slug":"about-%s","title":"About Us","body":"Welcome to Atlas"}`, suffix))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upsert page: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	resp, body = cmsDo(t, env, http.MethodGet, fmt.Sprintf("/api/v1/cms/pages/about-%s", suffix), "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("get page by slug: expected 200, got %d", resp.StatusCode)
	}

	// 5. FAQs & Banners & Menus & Settings
	resp, _ = cmsDo(t, env, http.MethodPost, "/api/v1/admin/cms/banners", `{"title":"Hero","image_url":"https://img.local/hero.jpg","position":"home_hero"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("create banner: expected 201, got %d", resp.StatusCode)
	}

	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/admin/cms/banners", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("list banners: expected 200, got %d", resp.StatusCode)
	}

	resp, _ = cmsDo(t, env, http.MethodPut, fmt.Sprintf("/api/v1/admin/cms/settings?key=site_name_%s", suffix), `{"value":"Atlas B2B"}`)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("update setting: expected 200, got %d", resp.StatusCode)
	}

	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/admin/cms/settings", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("list settings: expected 200, got %d", resp.StatusCode)
	}

	resp, _ = cmsDo(t, env, http.MethodPost, "/api/v1/admin/cms/menus", fmt.Sprintf(`{"location":"footer-%s","label":"Contact","url":"/contact"}`, suffix))
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("create menu: expected 201, got %d", resp.StatusCode)
	}

	resp, body = cmsDo(t, env, http.MethodGet, fmt.Sprintf("/api/v1/cms/menus/footer-%s", suffix), "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("get menus: expected 200, got %d", resp.StatusCode)
	}
}
