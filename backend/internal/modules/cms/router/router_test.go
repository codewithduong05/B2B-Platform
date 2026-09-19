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

func TestCMS_LegalDocuments(t *testing.T) {
	env := setupCMSEnv(t)
	_, _ = env.db.Pool.Exec(context.Background(), "TRUNCATE TABLE cms.legal_document_version RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(context.Background(), "TRUNCATE TABLE cms.legal_document CASCADE")

	docType := "terms_of_service"

	// 1. Create document type
	resp, body := cmsDo(t, env, http.MethodPost, "/api/v1/admin/cms/legal/documents", fmt.Sprintf(`{"doc_type":"%s","title":"Terms of Service"}`, docType))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create document: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var doc cms_schema.LegalDocument
	_ = json.Unmarshal(body, &doc)
	if doc.DocType != docType {
		t.Errorf("expected doc_type %q, got %q", docType, doc.DocType)
	}
	if doc.Title != "Terms of Service" {
		t.Errorf("expected title 'Terms of Service', got %q", doc.Title)
	}
	if doc.CurrentVersion != nil {
		t.Errorf("expected current_version null, got %v", doc.CurrentVersion)
	}
	if doc.HasDraft {
		t.Errorf("expected has_draft false, got true")
	}

	// 2. Create duplicate document type (should fail)
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/admin/cms/legal/documents", fmt.Sprintf(`{"doc_type":"%s","title":"Duplicate"}`, docType))
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("duplicate create: expected 409, got %d: %s", resp.StatusCode, string(body))
	}

	// 3. Create with invalid doc_type (should fail)
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/admin/cms/legal/documents", `{"doc_type":"INVALID","title":"Bad"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid doc_type: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// 4. Get document detail (empty)
	resp, body = cmsDo(t, env, http.MethodGet, fmt.Sprintf("/api/v1/admin/cms/legal/documents/%s", docType), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get detail: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var detail cms_schema.LegalDocumentDetail
	_ = json.Unmarshal(body, &detail)
	if detail.Draft != nil {
		t.Errorf("expected no draft, got one")
	}
	if len(detail.Versions) != 0 {
		t.Errorf("expected 0 versions, got %d", len(detail.Versions))
	}

	// 5. Update draft
	resp, body = cmsDo(t, env, http.MethodPut, fmt.Sprintf("/api/v1/admin/cms/legal/documents/%s/draft", docType), `{"title":"Terms of Service v1","body":"## Terms\n\nContent here","body_format":"markdown"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update draft: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var draft cms_schema.LegalDocumentDraft
	_ = json.Unmarshal(body, &draft)
	if draft.Title != "Terms of Service v1" {
		t.Errorf("expected draft title 'Terms of Service v1', got %q", draft.Title)
	}
	if draft.Body != "## Terms\n\nContent here" {
		t.Errorf("expected draft body, got %q", draft.Body)
	}
	if draft.BodyFormat != "markdown" {
		t.Errorf("expected body_format 'markdown', got %q", draft.BodyFormat)
	}

	// 6. Update draft with empty body (should fail)
	resp, body = cmsDo(t, env, http.MethodPut, fmt.Sprintf("/api/v1/admin/cms/legal/documents/%s/draft", docType), `{"title":"Test","body":""}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty body: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// 7. Update draft with invalid body_format (should fail)
	resp, body = cmsDo(t, env, http.MethodPut, fmt.Sprintf("/api/v1/admin/cms/legal/documents/%s/draft", docType), `{"title":"Test","body":"Content","body_format":"invalid"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid body_format: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// 8. Publish draft
	resp, body = cmsDo(t, env, http.MethodPost, fmt.Sprintf("/api/v1/admin/cms/legal/documents/%s/publish", docType), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("publish: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var v1 cms_schema.LegalDocumentVersion
	_ = json.Unmarshal(body, &v1)
	if v1.Version != 1 {
		t.Errorf("expected version 1, got %d", v1.Version)
	}
	if v1.Status != "published" {
		t.Errorf("expected status 'published', got %q", v1.Status)
	}
	if v1.Title != "Terms of Service v1" {
		t.Errorf("expected title 'Terms of Service v1', got %q", v1.Title)
	}

	// 9. Get current effective version (public)
	resp, body = cmsDo(t, env, http.MethodGet, fmt.Sprintf("/api/v1/cms/legal/%s", docType), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get current effective: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var current cms_schema.LegalDocumentVersion
	_ = json.Unmarshal(body, &current)
	if current.Version != 1 {
		t.Errorf("expected current version 1, got %d", current.Version)
	}

	// 10. Get specific version (public)
	resp, body = cmsDo(t, env, http.MethodGet, fmt.Sprintf("/api/v1/cms/legal/%s/versions/1", docType), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get version 1: expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	// 11. Get non-existent version (should 404)
	resp, body = cmsDo(t, env, http.MethodGet, fmt.Sprintf("/api/v1/cms/legal/%s/versions/999", docType), "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get version 999: expected 404, got %d: %s", resp.StatusCode, string(body))
	}

	// 12. Get current effective for non-existent doc_type (should 404)
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/cms/legal/nonexistent", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get nonexistent doc: expected 404, got %d: %s", resp.StatusCode, string(body))
	}

	// 13. List documents (admin)
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/admin/cms/legal/documents", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list documents: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var listResp struct {
		Items []cms_schema.LegalDocument `json:"items"`
		Total int                        `json:"total"`
	}
	_ = json.Unmarshal(body, &listResp)
	if listResp.Total != 1 {
		t.Errorf("expected 1 document, got %d", listResp.Total)
	}

	// 14. List versions (admin)
	resp, body = cmsDo(t, env, http.MethodGet, fmt.Sprintf("/api/v1/admin/cms/legal/documents/%s/versions", docType), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list versions: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var versionsResp struct {
		Items []cms_schema.LegalDocumentVersion `json:"items"`
		Total int                               `json:"total"`
	}
	_ = json.Unmarshal(body, &versionsResp)
	if versionsResp.Total != 1 {
		t.Errorf("expected 1 version, got %d", versionsResp.Total)
	}

	// 15. Create new draft and publish (version 2)
	resp, body = cmsDo(t, env, http.MethodPut, fmt.Sprintf("/api/v1/admin/cms/legal/documents/%s/draft", docType), `{"title":"Terms of Service v2","body":"## Updated Terms\n\nNew content"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update draft v2: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	resp, body = cmsDo(t, env, http.MethodPost, fmt.Sprintf("/api/v1/admin/cms/legal/documents/%s/publish", docType), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("publish v2: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var v2 cms_schema.LegalDocumentVersion
	_ = json.Unmarshal(body, &v2)
	if v2.Version != 2 {
		t.Errorf("expected version 2, got %d", v2.Version)
	}

	// 16. Verify version 1 is now superseded
	resp, body = cmsDo(t, env, http.MethodGet, fmt.Sprintf("/api/v1/cms/legal/%s/versions/1", docType), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get version 1 after supersede: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var v1After cms_schema.LegalDocumentVersion
	_ = json.Unmarshal(body, &v1After)
	if v1After.Status != "superseded" {
		t.Errorf("expected version 1 status 'superseded', got %q", v1After.Status)
	}

	// 17. Current effective should now be version 2
	resp, body = cmsDo(t, env, http.MethodGet, fmt.Sprintf("/api/v1/cms/legal/%s", docType), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get current effective after v2: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &current)
	if current.Version != 2 {
		t.Errorf("expected current version 2, got %d", current.Version)
	}

	// 18. Publish with no draft (should fail)
	resp, body = cmsDo(t, env, http.MethodPost, fmt.Sprintf("/api/v1/admin/cms/legal/documents/%s/publish", docType), "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("publish no draft: expected 422, got %d: %s", resp.StatusCode, string(body))
	}

	// 19. Test publish idempotency - create draft identical to current published
	resp, body = cmsDo(t, env, http.MethodPut, fmt.Sprintf("/api/v1/admin/cms/legal/documents/%s/draft", docType), `{"title":"Terms of Service v2","body":"## Updated Terms\n\nNew content"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update draft for idempotency: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	resp, body = cmsDo(t, env, http.MethodPost, fmt.Sprintf("/api/v1/admin/cms/legal/documents/%s/publish", docType), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("publish idempotent: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var vIdempotent cms_schema.LegalDocumentVersion
	_ = json.Unmarshal(body, &vIdempotent)
	if vIdempotent.Version != 2 {
		t.Errorf("idempotent publish: expected version 2 (no new version), got %d", vIdempotent.Version)
	}

	// 20. List documents with has_draft filter
	_, _ = cmsDo(t, env, http.MethodPut, fmt.Sprintf("/api/v1/admin/cms/legal/documents/%s/draft", docType), `{"title":"Draft","body":"Draft content"}`)
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/admin/cms/legal/documents?has_draft=true", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list with has_draft=true: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &listResp)
	if listResp.Total != 1 {
		t.Errorf("expected 1 document with draft, got %d", listResp.Total)
	}

	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/admin/cms/legal/documents?has_draft=false", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list with has_draft=false: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &listResp)
	if listResp.Total != 0 {
		t.Errorf("expected 0 documents without draft, got %d", listResp.Total)
	}

	// 21. Get document detail with draft and versions
	resp, body = cmsDo(t, env, http.MethodGet, fmt.Sprintf("/api/v1/admin/cms/legal/documents/%s", docType), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get detail with draft: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &detail)
	if detail.Draft == nil {
		t.Errorf("expected draft to exist")
	}
	if len(detail.Versions) < 2 {
		t.Errorf("expected at least 2 versions, got %d", len(detail.Versions))
	}
}

func TestCMS_ContactEnquiries(t *testing.T) {
	env := setupCMSEnv(t)
	_, _ = env.db.Pool.Exec(context.Background(), "TRUNCATE TABLE cms.contact_enquiry RESTART IDENTITY CASCADE")

	// 1. Submit contact enquiry - success
	resp, body := cmsDo(t, env, http.MethodPost, "/api/v1/cms/enquiries", `{"name":"Jane Doe","email":"jane@example.com","phone":"+84 123 456 789","company":"Acme Restaurant","subject":"Product inquiry","message":"I would like to know more about your chilled products."}`)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("submit enquiry: expected 202, got %d: %s", resp.StatusCode, string(body))
	}
	var enquiry cms_schema.ContactEnquiryResponse
	_ = json.Unmarshal(body, &enquiry)
	if enquiry.ID == "" {
		t.Errorf("expected enquiry ID to be set")
	}
	if enquiry.Name != "Jane Doe" {
		t.Errorf("expected name 'Jane Doe', got '%s'", enquiry.Name)
	}
	if enquiry.Email != "jane@example.com" {
		t.Errorf("expected email 'jane@example.com', got '%s'", enquiry.Email)
	}
	if enquiry.Status != "new" {
		t.Errorf("expected status 'new', got '%s'", enquiry.Status)
	}
	if enquiry.Source != "contact_form" {
		t.Errorf("expected source 'contact_form', got '%s'", enquiry.Source)
	}
	if enquiry.LeadID != nil {
		t.Errorf("expected lead_id to be null initially")
	}

	// 2. Submit enquiry with custom source
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/enquiries", `{"name":"John Smith","email":"john@example.com","subject":"Lead capture","message":"Interested in partnership","source":"lead_capture"}`)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("submit enquiry with source: expected 202, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &enquiry)
	if enquiry.Source != "lead_capture" {
		t.Errorf("expected source 'lead_capture', got '%s'", enquiry.Source)
	}

	// 3. Validation - missing name
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/enquiries", `{"email":"test@example.com","subject":"Test","message":"Test message"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing name: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// 4. Validation - missing email
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/enquiries", `{"name":"Test","subject":"Test","message":"Test message"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing email: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// 5. Validation - invalid email format
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/enquiries", `{"name":"Test","email":"not-an-email","subject":"Test","message":"Test message"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid email: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// 6. Validation - missing subject
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/enquiries", `{"name":"Test","email":"test@example.com","message":"Test message"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing subject: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// 7. Validation - missing message
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/enquiries", `{"name":"Test","email":"test@example.com","subject":"Test"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing message: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// 8. Validation - message too long
	longMessage := make([]byte, 5001)
	for i := range longMessage {
		longMessage[i] = 'a'
	}
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/enquiries", fmt.Sprintf(`{"name":"Test","email":"test@example.com","subject":"Test","message":"%s"}`, string(longMessage)))
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("message too long: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// 9. Validation - invalid JSON
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/enquiries", `{invalid json}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid JSON: expected 400, got %d: %s", resp.StatusCode, string(body))
	}

	// 10. Admin list enquiries
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/admin/cms/enquiries", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list enquiries: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var listResp cms_schema.ContactEnquiryListResponse
	_ = json.Unmarshal(body, &listResp)
	if listResp.Total < 2 {
		t.Errorf("expected at least 2 enquiries, got %d", listResp.Total)
	}

	// 11. Admin list with status filter
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/admin/cms/enquiries?status=new", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list with status filter: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &listResp)
	if listResp.Total < 2 {
		t.Errorf("expected at least 2 enquiries with status 'new', got %d", listResp.Total)
	}

	// 12. Admin list with source filter
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/admin/cms/enquiries?source=lead_capture", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list with source filter: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &listResp)
	if listResp.Total != 1 {
		t.Errorf("expected 1 enquiry with source 'lead_capture', got %d", listResp.Total)
	}

	// 13. Admin list with search
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/admin/cms/enquiries?q=Jane", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list with search: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &listResp)
	if listResp.Total != 1 {
		t.Errorf("expected 1 enquiry matching 'Jane', got %d", listResp.Total)
	}

	// 14. Admin get enquiry by code
	resp, body = cmsDo(t, env, http.MethodGet, fmt.Sprintf("/api/v1/admin/cms/enquiries/%s", enquiry.ID), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get enquiry by code: expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var detail cms_schema.ContactEnquiryResponse
	_ = json.Unmarshal(body, &detail)
	if detail.ID != enquiry.ID {
		t.Errorf("expected enquiry ID '%s', got '%s'", enquiry.ID, detail.ID)
	}

	// 15. Admin get non-existent enquiry
	resp, body = cmsDo(t, env, http.MethodGet, "/api/v1/admin/cms/enquiries/enq_nonexistent", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get non-existent enquiry: expected 404, got %d: %s", resp.StatusCode, string(body))
	}

	// 16. Email normalization - should be lowercased
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/enquiries", `{"name":"Test User","email":"TEST@EXAMPLE.COM","subject":"Test","message":"Test message"}`)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("email normalization: expected 202, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &enquiry)
	if enquiry.Email != "test@example.com" {
		t.Errorf("expected email to be lowercased to 'test@example.com', got '%s'", enquiry.Email)
	}

	// 17. Duplicate submissions allowed (no dedup at CMS level)
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/enquiries", `{"name":"Jane Doe","email":"jane@example.com","subject":"Another inquiry","message":"Second message"}`)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("duplicate submission: expected 202, got %d: %s", resp.StatusCode, string(body))
	}
	_ = json.Unmarshal(body, &enquiry)
	if enquiry.ID == "" {
		t.Errorf("expected new enquiry ID for duplicate submission")
	}

	// 18. Optional fields - phone and company can be null
	resp, body = cmsDo(t, env, http.MethodPost, "/api/v1/cms/enquiries", `{"name":"Minimal User","email":"minimal@example.com","subject":"Minimal","message":"Minimal message"}`)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("optional fields: expected 202, got %d: %s", resp.StatusCode, string(body))
	}
	var minimalEnquiry cms_schema.ContactEnquiryResponse
	_ = json.Unmarshal(body, &minimalEnquiry)
	if minimalEnquiry.Phone != nil {
		t.Errorf("expected phone to be null")
	}
	if minimalEnquiry.Company != nil {
		t.Errorf("expected company to be null")
	}
}
