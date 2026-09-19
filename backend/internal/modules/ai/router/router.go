package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/atlas-platform/backend/internal/modules/ai/service"
)

type Router struct {
	svc *service.AIService
}

func New(svc *service.AIService) *Router {
	return &Router{svc: svc}
}

func (rt *Router) Register(r chi.Router) {
	r.Route("/ai", func(r chi.Router) {
		r.Get("/health", rt.handleHealth)
	})
}

func (rt *Router) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","module":"ai"}`))
}
