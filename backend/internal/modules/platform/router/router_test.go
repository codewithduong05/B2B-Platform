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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/platform"
	"github.com/atlas-platform/backend/internal/modules/platform/router"
	"github.com/atlas-platform/backend/internal/modules/platform/schema"
)

const platformTestDBLockKey = 1946819412

func TestMain(m *testing.M) {
	ctx := context.Background()
	cfg, err := config.Load()
	if err == nil {
		rawDB, err := sql.Open("pgx", cfg.PostgresDSN())
		if err == nil && rawDB != nil {
			_, _ = rawDB.Exec("SELECT pg_advisory_lock($1)", int64(platformTestDBLockKey))
			_, _ = rawDB.Exec("DROP SCHEMA IF EXISTS catalog CASCADE; DROP SCHEMA IF EXISTS inventory CASCADE; DROP SCHEMA IF EXISTS identity CASCADE; DROP SCHEMA IF EXISTS pricing CASCADE; DROP SCHEMA IF EXISTS commerce CASCADE; DROP SCHEMA IF EXISTS payments CASCADE; DROP SCHEMA IF EXISTS promotions CASCADE; DROP SCHEMA IF EXISTS crm CASCADE; DROP SCHEMA IF EXISTS suppliers CASCADE; DROP SCHEMA IF EXISTS cms CASCADE; DROP SCHEMA IF EXISTS reports CASCADE; DROP SCHEMA IF EXISTS erp CASCADE; DROP SCHEMA IF EXISTS platform CASCADE; DROP SCHEMA IF EXISTS ai CASCADE; DROP SCHEMA IF EXISTS analytics CASCADE; DROP TABLE IF EXISTS schema_migrations CASCADE; DROP TYPE IF EXISTS catalog_handling_class_type CASCADE;")

			if err := database.RunMigrations(ctx, &cfg.Postgres, "file://../../../../migrations"); err != nil {
				fmt.Printf("TestMain migration error: %v\n", err)
			}

			code := m.Run()
			_, _ = rawDB.Exec("SELECT pg_advisory_unlock($1)", int64(platformTestDBLockKey))
			rawDB.Close()
			os.Exit(code)
		}
	}
	os.Exit(m.Run())
}

type platformEnv struct {
	db  *database.DB
	srv *httptest.Server
}

func setupPlatformEnv(t *testing.T) *platformEnv {
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
	svc := platform.NewService(db)
	rt := platform.New(svc)
	rt.RegisterRoutes(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := router.WithPrincipalID(r.Context(), 1)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		},
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
	return &platformEnv{db: db, srv: server}
}

func platformDo(t *testing.T, env *platformEnv, method, path, body string) (*http.Response, []byte) {
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

func TestPlatform_CreateTemplate(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.notification_template RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	req := schema.CreateTemplateRequest{
		Code:      "order_confirmation",
		Name:      "Order Confirmation",
		Channel:   "email",
		Subject:   strPtr("Order {{order_code}} Confirmed"),
		Body:      "Your order {{order_code}} has been confirmed. Total: {{total}}",
		Variables: []string{"order_code", "total"},
	}

	body, _ := json.Marshal(req)
	resp, respBody := platformDo(t, env, http.MethodPost, "/api/v1/admin/platform/notification-templates", string(body))
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var template schema.NotificationTemplateResponse
	_ = json.Unmarshal(respBody, &template)
	assert.Equal(t, "order_confirmation", template.Code)
	assert.Equal(t, "Order Confirmation", template.Name)
	assert.Equal(t, "email", template.Channel)
	assert.Equal(t, "Order {{order_code}} Confirmed", *template.Subject)
	assert.Equal(t, "active", template.Status)
	assert.ElementsMatch(t, []string{"order_code", "total"}, template.Variables)
}

func TestPlatform_ListTemplates(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.notification_template RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	for i := 0; i < 3; i++ {
		_, _ = env.db.Pool.Exec(ctx, `
			INSERT INTO platform.notification_template (code, name, channel, subject, body_template, variables, status)
			VALUES ($1, $2, 'email', 'Subject', 'Body', '[]', 'active')
		`, fmt.Sprintf("template_%d", i), fmt.Sprintf("Template %d", i))
	}

	resp, respBody := platformDo(t, env, http.MethodGet, "/api/v1/admin/platform/notification-templates", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Items []schema.NotificationTemplateResponse `json:"items"`
		Total int                                   `json:"total"`
	}
	_ = json.Unmarshal(respBody, &result)
	assert.Equal(t, 3, result.Total)
	assert.Len(t, result.Items, 3)
}

func TestPlatform_UpdateTemplate(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.notification_template RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	var templateID int64
	_ = env.db.Pool.QueryRow(ctx, `
		INSERT INTO platform.notification_template (code, name, channel, subject, body_template, variables, status)
		VALUES ('test_template', 'Test Template', 'email', 'Old Subject', 'Old Body', '[]', 'active')
		RETURNING id
	`).Scan(&templateID)

	req := schema.UpdateTemplateRequest{
		Name:    strPtr("Updated Template"),
		Subject: strPtr("New Subject"),
		Status:  strPtr("inactive"),
	}

	body, _ := json.Marshal(req)
	resp, respBody := platformDo(t, env, http.MethodPatch, fmt.Sprintf("/api/v1/admin/platform/notification-templates/%d", templateID), string(body))
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var template schema.NotificationTemplateResponse
	_ = json.Unmarshal(respBody, &template)
	assert.Equal(t, "Updated Template", template.Name)
	assert.Equal(t, "New Subject", *template.Subject)
	assert.Equal(t, "inactive", template.Status)
}

func TestPlatform_SendNotification(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE identity.user_role, identity.role_permission, identity.permission, identity.role, identity.buyer_profile, identity.user RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.notification, platform.notification_event, platform.notification_template RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (100, 'usr_buy001', 'buyer@test.com', 'hash', 'buyer')`)
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity.buyer_profile (id, code, business_name, user_id) VALUES (100, 'BUYER001', 'Test Corp', 100)`)

	_, _ = env.db.Pool.Exec(ctx, `
		INSERT INTO platform.notification_template (code, name, channel, subject, body_template, variables, status)
		VALUES ('welcome', 'Welcome Email', 'email', 'Welcome {{name}}', 'Hello {{name}}, welcome!', '["name"]', 'active')
	`)

	req := schema.SendNotificationRequest{
		TemplateCode:  "welcome",
		RecipientType: "buyer",
		RecipientID:   100,
		Variables:     map[string]interface{}{"name": "John"},
	}

	body, _ := json.Marshal(req)
	resp, respBody := platformDo(t, env, http.MethodPost, "/api/v1/admin/platform/notifications/send", string(body))
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	var notification schema.NotificationResponse
	_ = json.Unmarshal(respBody, &notification)
	assert.Equal(t, "buyer", notification.RecipientType)
	assert.Equal(t, int64(100), notification.RecipientID)
	assert.Equal(t, "email", notification.Channel)
	assert.Equal(t, "buyer@test.com", notification.RecipientAddress)
	assert.Equal(t, "Welcome John", *notification.Subject)
	assert.Equal(t, "Hello John, welcome!", notification.Body)
	assert.Equal(t, "sent", notification.Status)
	assert.NotNil(t, notification.ProviderMessageID)
	assert.NotNil(t, notification.SentAt)
}

func TestPlatform_BuyerNotifications(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.notification RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (100, 'usr_buy001', 'buyer@test.com', 'hash', 'buyer')`)
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity.buyer_profile (id, code, business_name, user_id) VALUES (100, 'BUYER001', 'Test Corp', 100)`)

	for i := 0; i < 3; i++ {
		_, _ = env.db.Pool.Exec(ctx, `
			INSERT INTO platform.notification (code, template_id, recipient_type, recipient_id, channel, recipient_address, subject, body, status)
			VALUES ($1, NULL, 'buyer', 1, 'email', 'buyer@test.com', 'Subject', 'Body', 'sent')
		`, fmt.Sprintf("notif_%d", i))
	}

	resp, respBody := platformDo(t, env, http.MethodGet, "/api/v1/platform/notifications", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Items []schema.BuyerNotificationResponse `json:"items"`
		Total int                                `json:"total"`
	}
	_ = json.Unmarshal(respBody, &result)
	assert.Equal(t, 3, result.Total)
	assert.Len(t, result.Items, 3)
}

func TestPlatform_AdminListNotifications(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.notification RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	for i := 0; i < 5; i++ {
		channel := "email"
		if i%2 == 0 {
			channel = "sms"
		}
		_, _ = env.db.Pool.Exec(ctx, `
			INSERT INTO platform.notification (code, template_id, recipient_type, recipient_id, channel, recipient_address, subject, body, status)
			VALUES ($1, NULL, 'buyer', 100, $2, 'test@test.com', 'Subject', 'Body', 'sent')
		`, fmt.Sprintf("notif_%d", i), channel)
	}

	resp, respBody := platformDo(t, env, http.MethodGet, "/api/v1/admin/platform/notifications?channel=email", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Items []schema.NotificationResponse `json:"items"`
		Total int                           `json:"total"`
	}
	_ = json.Unmarshal(respBody, &result)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Items, 2)
	for _, item := range result.Items {
		assert.Equal(t, "email", item.Channel)
	}
}

func TestPlatform_SendNotification_TemplateNotFound(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.notification_template RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	req := schema.SendNotificationRequest{
		TemplateCode:  "nonexistent",
		RecipientType: "buyer",
		RecipientID:   100,
	}

	body, _ := json.Marshal(req)
	resp, _ := platformDo(t, env, http.MethodPost, "/api/v1/admin/platform/notifications/send", string(body))
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestPlatform_SendNotification_InactiveTemplate(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.notification_template RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (100, 'usr_buy001', 'buyer@test.com', 'hash', 'buyer')`)
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity.buyer_profile (id, code, business_name, user_id) VALUES (100, 'BUYER001', 'Test Corp', 100)`)

	_, _ = env.db.Pool.Exec(ctx, `
		INSERT INTO platform.notification_template (code, name, channel, subject, body_template, variables, status)
		VALUES ('inactive_template', 'Inactive', 'email', 'Subject', 'Body', '[]', 'inactive')
	`)

	req := schema.SendNotificationRequest{
		TemplateCode:  "inactive_template",
		RecipientType: "buyer",
		RecipientID:   100,
	}

	body, _ := json.Marshal(req)
	resp, _ := platformDo(t, env, http.MethodPost, "/api/v1/admin/platform/notifications/send", string(body))
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestPlatform_CreateMedia(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.media_upload RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	req := schema.CreateMediaRequest{
		Filename:    "test.jpg",
		ContentType: "image/jpeg",
	}

	body, _ := json.Marshal(req)
	resp, respBody := platformDo(t, env, http.MethodPost, "/api/v1/platform/media", string(body))
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var media schema.MediaResponse
	_ = json.Unmarshal(respBody, &media)
	assert.Equal(t, "test.jpg", media.Filename)
	assert.Equal(t, "image/jpeg", media.ContentType)
	assert.NotEmpty(t, media.Code)
	assert.NotEmpty(t, media.UploadURL)
	assert.Equal(t, "pending", media.Status)
}

func TestPlatform_CreateMedia_InvalidInput(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	req := schema.CreateMediaRequest{
		Filename:    "",
		ContentType: "image/jpeg",
	}

	body, _ := json.Marshal(req)
	resp, _ := platformDo(t, env, http.MethodPost, "/api/v1/platform/media", string(body))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPlatform_ListRegions(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.reference_region RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	_, _ = env.db.Pool.Exec(ctx, `
		INSERT INTO platform.reference_region (code, name, country, is_active)
		VALUES 
			('north', 'Miền Bắc', 'VN', true),
			('south', 'Miền Nam', 'VN', true),
			('central', 'Miền Trung', 'VN', false)
	`)

	resp, respBody := platformDo(t, env, http.MethodGet, "/api/v1/platform/reference/regions", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Items []schema.RegionResponse `json:"items"`
	}
	_ = json.Unmarshal(respBody, &result)
	assert.Len(t, result.Items, 2)
	for _, item := range result.Items {
		assert.True(t, item.IsActive)
	}
}

func TestPlatform_ListFeatureFlags(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.feature_flag RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	_, _ = env.db.Pool.Exec(ctx, `
		INSERT INTO platform.feature_flag (key, name, description, enabled)
		VALUES 
			('flag1', 'Feature 1', 'Description 1', true),
			('flag2', 'Feature 2', 'Description 2', false)
	`)

	resp, respBody := platformDo(t, env, http.MethodGet, "/api/v1/platform/feature-flags", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Items []schema.FeatureFlagResponse `json:"items"`
		Total int                          `json:"total"`
	}
	_ = json.Unmarshal(respBody, &result)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Items, 2)
}

func TestPlatform_ListFeatureFlags_FilterEnabled(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.feature_flag RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	_, _ = env.db.Pool.Exec(ctx, `
		INSERT INTO platform.feature_flag (key, name, enabled)
		VALUES 
			('flag1', 'Feature 1', true),
			('flag2', 'Feature 2', false)
	`)

	resp, respBody := platformDo(t, env, http.MethodGet, "/api/v1/platform/feature-flags?enabled=true", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Items []schema.FeatureFlagResponse `json:"items"`
		Total int                          `json:"total"`
	}
	_ = json.Unmarshal(respBody, &result)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Items, 1)
	assert.True(t, result.Items[0].Enabled)
}

func TestPlatform_AdminListFeatureFlags(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.feature_flag RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	_, _ = env.db.Pool.Exec(ctx, `
		INSERT INTO platform.feature_flag (key, name, enabled)
		VALUES ('test_flag', 'Test Flag', false)
	`)

	resp, respBody := platformDo(t, env, http.MethodGet, "/api/v1/admin/feature-flags", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Items []schema.FeatureFlagResponse `json:"items"`
		Total int                          `json:"total"`
	}
	_ = json.Unmarshal(respBody, &result)
	assert.Equal(t, 1, result.Total)
}

func TestPlatform_UpsertFeatureFlag(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.feature_flag RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	req := schema.UpdateFeatureFlagRequest{
		Name:    "New Feature",
		Enabled: true,
	}

	body, _ := json.Marshal(req)
	resp, respBody := platformDo(t, env, http.MethodPut, "/api/v1/admin/feature-flags/new_feature", string(body))
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var flag schema.FeatureFlagResponse
	_ = json.Unmarshal(respBody, &flag)
	assert.Equal(t, "new_feature", flag.Key)
	assert.Equal(t, "New Feature", flag.Name)
	assert.True(t, flag.Enabled)

	req2 := schema.UpdateFeatureFlagRequest{
		Name:    "Updated Feature",
		Enabled: false,
	}
	body2, _ := json.Marshal(req2)
	resp2, respBody2 := platformDo(t, env, http.MethodPut, "/api/v1/admin/feature-flags/new_feature", string(body2))
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var flag2 schema.FeatureFlagResponse
	_ = json.Unmarshal(respBody2, &flag2)
	assert.Equal(t, "new_feature", flag2.Key)
	assert.Equal(t, "Updated Feature", flag2.Name)
	assert.False(t, flag2.Enabled)
}

func TestPlatform_UpsertFeatureFlag_InvalidInput(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	req := schema.UpdateFeatureFlagRequest{
		Name:    "",
		Enabled: true,
	}

	body, _ := json.Marshal(req)
	resp, _ := platformDo(t, env, http.MethodPut, "/api/v1/admin/feature-flags/test", string(body))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPlatform_ListAuditLog(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.audit_log RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	_, _ = env.db.Pool.Exec(ctx, `
		INSERT INTO platform.audit_log (actor_id, actor_type, action, resource_type, resource_id)
		VALUES 
			(1, 'user', 'create', 'product', 100),
			(1, 'user', 'update', 'product', 100),
			(1, 'user', 'delete', 'order', 200)
	`)

	resp, respBody := platformDo(t, env, http.MethodGet, "/api/v1/admin/platform/audit-log", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Items []schema.AuditLogResponse `json:"items"`
		Total int                       `json:"total"`
	}
	_ = json.Unmarshal(respBody, &result)
	assert.Equal(t, 3, result.Total)
	assert.Len(t, result.Items, 3)
}

func TestPlatform_ListAuditLog_FilterAction(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.audit_log RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	_, _ = env.db.Pool.Exec(ctx, `
		INSERT INTO platform.audit_log (actor_id, actor_type, action, resource_type)
		VALUES 
			(1, 'user', 'create', 'product'),
			(1, 'user', 'update', 'product'),
			(1, 'user', 'create', 'order')
	`)

	resp, respBody := platformDo(t, env, http.MethodGet, "/api/v1/admin/platform/audit-log?action=create", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Items []schema.AuditLogResponse `json:"items"`
		Total int                       `json:"total"`
	}
	_ = json.Unmarshal(respBody, &result)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Items, 2)
	for _, item := range result.Items {
		assert.Equal(t, "create", item.Action)
	}
}

func TestPlatform_ListIntegrationTraffic(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.integration_traffic RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	_, _ = env.db.Pool.Exec(ctx, `
		INSERT INTO platform.integration_traffic (direction, provider, endpoint, method, status_code)
		VALUES 
			('inbound', 'erp', '/api/orders', 'POST', 200),
			('outbound', 'erp', '/api/products', 'GET', 200),
			('inbound', 'crm', '/api/customers', 'POST', 201)
	`)

	resp, respBody := platformDo(t, env, http.MethodGet, "/api/v1/admin/platform/integration-traffic", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Items []schema.IntegrationTrafficResponse `json:"items"`
		Total int                                 `json:"total"`
	}
	_ = json.Unmarshal(respBody, &result)
	assert.Equal(t, 3, result.Total)
	assert.Len(t, result.Items, 3)
}

func TestPlatform_ListIntegrationTraffic_FilterDirection(t *testing.T) {
	env := setupPlatformEnv(t)
	ctx := context.Background()
	_, _ = env.db.Pool.Exec(ctx, "TRUNCATE TABLE platform.integration_traffic RESTART IDENTITY CASCADE")
	_, _ = env.db.Pool.Exec(ctx, `INSERT INTO identity."user" (id, code, email, password_hash, user_type) VALUES (1, 'usr_adm001', 'admin@test.com', 'hash', 'platform')`)

	_, _ = env.db.Pool.Exec(ctx, `
		INSERT INTO platform.integration_traffic (direction, provider, endpoint)
		VALUES 
			('inbound', 'erp', '/api/orders'),
			('outbound', 'erp', '/api/products'),
			('inbound', 'crm', '/api/customers')
	`)

	resp, respBody := platformDo(t, env, http.MethodGet, "/api/v1/admin/platform/integration-traffic?direction=inbound", "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Items []schema.IntegrationTrafficResponse `json:"items"`
		Total int                                 `json:"total"`
	}
	_ = json.Unmarshal(respBody, &result)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Items, 2)
	for _, item := range result.Items {
		assert.Equal(t, "inbound", item.Direction)
	}
}

func strPtr(s string) *string {
	return &s
}
