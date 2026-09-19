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
			_, _ = rawDB.Exec("DROP SCHEMA IF EXISTS catalog CASCADE; DROP SCHEMA IF EXISTS inventory CASCADE; DROP SCHEMA IF EXISTS identity CASCADE; DROP SCHEMA IF EXISTS pricing CASCADE; DROP SCHEMA IF EXISTS commerce CASCADE; DROP SCHEMA IF EXISTS payments CASCADE; DROP SCHEMA IF EXISTS promotions CASCADE; DROP SCHEMA IF EXISTS crm CASCADE; DROP SCHEMA IF EXISTS suppliers CASCADE; DROP SCHEMA IF EXISTS cms CASCADE; DROP SCHEMA IF EXISTS reports CASCADE; DROP SCHEMA IF EXISTS erp CASCADE; DROP SCHEMA IF EXISTS platform CASCADE; DROP SCHEMA IF EXISTS ai CASCADE; DROP TABLE IF EXISTS schema_migrations CASCADE; DROP TYPE IF EXISTS catalog_handling_class_type CASCADE;")

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
	_, _ = env.db.Pool.Exec(context.Background(), "TRUNCATE TABLE cms.article, cms.page, cms.faq, cms.banner, cms.menu_item, cms.setting, cms.seo_template, cms.seo_settings RESTART IDENTITY CASCADE")
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

func TestCMS_HomepageBuilder(t *testing.T) {
	env := setupCMSEnv(t)
	_, _ = env.db.Pool.Exec(context.Background(), "TRUNCATE TABLE cms.homepage_layout RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(context.Background(), "INSERT INTO cms.homepage_layout (id) VALUES (1) ON CONFLICT DO NOTHING")

	// 1. Get published homepage (should be empty initially)
	resp, body := cmsDo(t, env, http.MethodGet, "/api/v1/cms/homepage", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get published homepage: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var pubResp cms_schema.HomepageResponse
	_ = json.Unmarshal(body, &pubResp)
	if len(pubResp.Sections) != 0 {
		t.Errorf("expected 0 published sections, got %d", len(pubResp.Sections))
	}
	if pubResp.PublishedAt != nil {
		t.Errorf("expected published_at to be nil, got %v", pubResp.PublishedAt)
	}

	// 2. Get admin homepage (should show empty draft and published)
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/admin/cms/homepage", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get admin homepage: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var adminResp cms_schema.HomepageResponse
	_ = json.Unmarshal(body, &adminResp)
	if len(adminResp.Sections) != 0 {
		t.Errorf("expected 0 draft sections, got %d", len(adminResp.Sections))
	}
	if len(adminResp.PublishedSections) != 0 {
		t.Errorf("expected 0 published sections, got %d", len(adminResp.PublishedSections))
	}

	// 3. Update draft homepage with sections
	draftBody := `{
		"sections": [
			{"code": "sec_1", "section_type": "hero_banner", "title": "Hero", "is_active": true, "config": {"banner_code": "ban_123"}},
			{"code": "sec_2", "section_type": "featured_products", "title": "Featured", "is_active": true, "config": {}},
			{"code": "sec_3", "section_type": "newsletter", "title": "Newsletter", "is_active": false, "config": {}}
		]
	}`
	resp, body = cmsDo(t, env, http.MethodPut, "/api/v1/admin/cms/homepage", draftBody)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update draft homepage: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &adminResp)
	if len(adminResp.Sections) != 3 {
		t.Errorf("expected 3 draft sections, got %d", len(adminResp.Sections))
	}
	if adminResp.Sections[0].SortOrder != 0 {
		t.Errorf("expected sort_order 0, got %d", adminResp.Sections[0].SortOrder)
	}
	if adminResp.Sections[1].SortOrder != 1 {
		t.Errorf("expected sort_order 1, got %d", adminResp.Sections[1].SortOrder)
	}
	if adminResp.Sections[2].SortOrder != 2 {
		t.Errorf("expected sort_order 2, got %d", adminResp.Sections[2].SortOrder)
	}

	// 4. Publish homepage (should filter out inactive sections)
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/admin/cms/homepage/publish", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("publish homepage: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &adminResp)
	if len(adminResp.PublishedSections) != 2 {
		t.Errorf("expected 2 published sections (inactive filtered), got %d", len(adminResp.PublishedSections))
	}
	if adminResp.PublishedAt == nil {
		t.Errorf("expected published_at to be set")
	}

	// 5. Get published homepage (should show only active sections)
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/cms/homepage", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get published homepage: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &pubResp)
	if len(pubResp.Sections) != 2 {
		t.Errorf("expected 2 published sections, got %d", len(pubResp.Sections))
	}
	if pubResp.PublishedAt == nil {
		t.Errorf("expected published_at to be set in public response")
	}
	if pubResp.UpdatedBy != nil {
		t.Errorf("expected updated_by to be nil in public response")
	}

	// 6. Update draft again and verify isolation
	draftBody2 := `{
		"sections": [
			{"code": "sec_4", "section_type": "new_hero", "title": "New Hero", "is_active": true, "config": {}}
		]
	}`
	resp, body = cmsDo(t, env, http.MethodPut, "/api/v1/admin/cms/homepage", draftBody2)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update draft homepage again: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &adminResp)
	if len(adminResp.Sections) != 1 {
		t.Errorf("expected 1 draft section after update, got %d", len(adminResp.Sections))
	}
	if len(adminResp.PublishedSections) != 2 {
		t.Errorf("expected 2 published sections (unchanged), got %d", len(adminResp.PublishedSections))
	}

	// 7. Validation: empty section code should fail
	invalidDraft := `{"sections": [{"code": "", "section_type": "hero"}]}`
	resp, body = cmsDo(t, env, http.MethodPut, "/api/v1/admin/cms/homepage", invalidDraft)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty section code, got %d: %s", resp.StatusCode, string(body))
	}

	// 8. Validation: empty section_type should fail
	invalidDraft2 := `{"sections": [{"code": "sec_x", "section_type": ""}]}`
	resp, body = cmsDo(t, env, http.MethodPut, "/api/v1/admin/cms/homepage", invalidDraft2)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty section_type, got %d: %s", resp.StatusCode, string(body))
	}

	// 9. Validation: publish with no active sections should fail (422)
	emptyDraft := `{"sections": [{"code": "sec_inactive", "section_type": "hero", "is_active": false}]}`
	resp, _ = cmsDo(t, env, http.MethodPut, "/api/v1/admin/cms/homepage", emptyDraft)
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/admin/cms/homepage/publish", "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for publish with no active sections, got %d: %s", resp.StatusCode, string(body))
	}

	// 10. Optimistic locking: stale expected_updated_at should return 409
	// First, set a known draft and capture updated_at
	setupDraft := `{"sections": [{"code": "sec_lock", "section_type": "hero", "is_active": true, "config": {}}]}`
	resp, body = cmsDo(t, env, http.MethodPut, "/api/v1/admin/cms/homepage", setupDraft)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("setup draft for lock test: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var lockResp cms_schema.HomepageResponse
	_ = json.Unmarshal(body, &lockResp)
	staleTime := lockResp.UpdatedAt.Add(-1 * time.Hour)
	staleJSON, _ := json.Marshal(map[string]interface{}{
		"sections":            []map[string]interface{}{{"code": "sec_lock2", "section_type": "hero", "is_active": true}},
		"expected_updated_at": staleTime.Format(time.RFC3339Nano),
	})
	resp, body = cmsDo(t, env, http.MethodPut, "/api/v1/admin/cms/homepage", string(staleJSON))
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 for stale expected_updated_at, got %d: %s", resp.StatusCode, string(body))
	}
	var errBody map[string]string
	_ = json.Unmarshal(body, &errBody)
	if errBody["code"] != "concurrent_modification" {
		t.Errorf("expected error code 'concurrent_modification', got %q", errBody["code"])
	}
}

func TestCMS_NewsletterSubscription(t *testing.T) {
	env := setupCMSEnv(t)
	_, _ = env.db.Pool.Exec(context.Background(), "TRUNCATE TABLE cms.newsletter_subscriber RESTART IDENTITY CASCADE")
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	// 1. Subscribe with valid email
	resp, body := cmsDo(t, env, http.MethodPost, "/api/v1/cms/newsletter/subscribe", fmt.Sprintf(`{"email":"test%s@example.com","first_name":"Test","source":"footer"}`, suffix))
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("subscribe: expected 202, got %d: %s", resp.StatusCode, string(body))
	}
	var sub cms_schema.NewsletterSubscriber
	_ = json.Unmarshal(body, &sub)
	if sub.Email != fmt.Sprintf("test%s@example.com", suffix) {
		t.Errorf("expected normalized email, got %q", sub.Email)
	}
	if sub.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", sub.Status)
	}
	if sub.Code == "" {
		t.Errorf("expected subscriber code to be set")
	}

	// 2. Subscribe with invalid email
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/newsletter/subscribe", `{"email":"not-an-email"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("subscribe with invalid email: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// 3. Subscribe with missing email
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/newsletter/subscribe", `{"first_name":"Test"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("subscribe with missing email: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// 4. Confirm with invalid token
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/newsletter/confirm", `{"token":"invalid-token"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("confirm with invalid token: expected 404, got %d: %s", resp.StatusCode, string(body))
	}

	// 5. Unsubscribe with invalid token
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/newsletter/unsubscribe", `{"token":"invalid-token"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unsubscribe with invalid token: expected 404, got %d: %s", resp.StatusCode, string(body))
	}

	// 6. Unsubscribe with email (should succeed even if email doesn't exist)
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/newsletter/unsubscribe", `{"email":"nonexistent@example.com"}`)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("unsubscribe with email: expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	// 7. Unsubscribe with neither token nor email
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/newsletter/unsubscribe", `{}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unsubscribe with no token or email: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// 8. Admin list subscribers
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/admin/cms/newsletter/subscribers", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("admin list subscribers: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var listEnv struct {
		Items []cms_schema.NewsletterSubscriber `json:"items"`
		Total int                               `json:"total"`
	}
	_ = json.Unmarshal(body, &listEnv)
	if listEnv.Total != 1 {
		t.Errorf("expected 1 subscriber, got %d", listEnv.Total)
	}

	// 9. Admin get subscriber by code
	resp, body = cmsDo(t, env, http.MethodGet, fmt.Sprintf("/api/v1/admin/cms/newsletter/subscribers/%s", sub.Code), "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("admin get subscriber: expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	// 10. Admin get stats
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/admin/cms/newsletter/stats", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("admin get stats: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var stats cms_schema.SubscriberStatsResponse
	_ = json.Unmarshal(body, &stats)
	if stats.Total != 1 {
		t.Errorf("expected stats total 1, got %d", stats.Total)
	}
	if stats.Pending != 1 {
		t.Errorf("expected stats pending 1, got %d", stats.Pending)
	}

	// 11. Admin export subscribers (CSV)
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/admin/cms/newsletter/subscribers/export", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("admin export subscribers: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/csv; charset=utf-8" {
		t.Errorf("expected CSV content type, got %q", ct)
	}

	// 12. Admin delete subscriber
	resp, body = cmsDo(t, env, http.MethodDelete, fmt.Sprintf("/api/v1/admin/cms/newsletter/subscribers/%s", sub.Code), "")
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("admin delete subscriber: expected 204, got %d: %s", resp.StatusCode, string(body))
	}

	// 13. Verify deletion
	resp, body = cmsDo(t, env, http.MethodGet, fmt.Sprintf("/api/v1/admin/cms/newsletter/subscribers/%s", sub.Code), "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get deleted subscriber: expected 404, got %d: %s", resp.StatusCode, string(body))
	}

	// 14. Re-subscribe after unsubscribe (test re-subscription flow)
	// First, create a new subscriber
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/newsletter/subscribe", fmt.Sprintf(`{"email":"resubscribe%s@example.com"}`, suffix))
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("re-subscribe setup: expected 202, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &sub)

	// Manually confirm via DB for testing
	_, _ = env.db.Pool.Exec(context.Background(), "UPDATE cms.newsletter_subscriber SET status = 'confirmed', confirmed_at = NOW() WHERE code = $1", sub.Code)

	// Subscribe again as confirmed → should return 202 with no state change
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/newsletter/subscribe", fmt.Sprintf(`{"email":"resubscribe%s@example.com"}`, suffix))
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("re-subscribe confirmed: expected 202, got %d: %s", resp.StatusCode, string(body))
	}
}
