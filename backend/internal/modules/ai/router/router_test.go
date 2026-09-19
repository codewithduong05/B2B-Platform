package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/atlas-platform/backend/internal/modules/ai/service"
)

func TestAI_Health(t *testing.T) {
	svc := service.NewAIService(nil)
	rt := New(svc)

	r := chi.NewRouter()
	rt.Register(r)

	req := httptest.NewRequest(http.MethodGet, "/ai/health", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	expected := `{"status":"ok","module":"ai"}`
	if body != expected {
		t.Errorf("expected body %q, got %q", expected, body)
	}
}
