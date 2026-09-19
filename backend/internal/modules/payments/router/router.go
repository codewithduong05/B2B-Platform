package router

// Payments HTTP surface. Buyer routes under auth; staff routes grouped
// under the injected admin middleware (a global stub in production wiring;
// tests inject a staff-set check).

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/atlas-platform/backend/internal/modules/payments/schema"
	"github.com/atlas-platform/backend/internal/modules/payments/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type principalKey string

const principalCtxKey principalKey = "payments.principal_id"

func WithPrincipalID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, principalCtxKey, id)
}

func PrincipalIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(principalCtxKey).(int64)
	return id
}

type Router struct {
	router  chi.Router
	service *service.PaymentService
}

func New(svc *service.PaymentService) *Router {
	return &Router{router: chi.NewRouter(), service: svc}
}

func (rt *Router) ChiRouter() chi.Router {
	return rt.router
}

func (rt *Router) RegisterRoutes(authMiddleware, adminMiddleware func(http.Handler) http.Handler) {
	rt.router.Route("/payments", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Get("/methods", rt.handleListMethods)
		r.Post("/intents", rt.handleCreateIntent)
		r.Get("/intents/{code}", rt.handleGetIntent)
	})

	rt.router.Route("/admin/payments", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		if adminMiddleware != nil {
			r.Use(adminMiddleware)
		}
		r.Get("/", rt.handleAdminListIntents)
		r.Post("/intents/{code}/complete", rt.handleMarkComplete)
		r.Post("/intents/{code}/fail", rt.handleMarkFailed)
	})
}

func principalID(r *http.Request) int64 {
	if id := PrincipalIDFromContext(r.Context()); id != 0 {
		return id
	}
	return 1
}

func (rt *Router) handleListMethods(w http.ResponseWriter, r *http.Request) {
	methods, err := rt.service.ListMethods(r.Context(), principalID(r))
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if methods == nil {
		methods = []schema.PaymentMethodResponse{}
	}
	rt.writeJSON(w, http.StatusOK, methods)
}

func (rt *Router) handleCreateIntent(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		r.Body.Close()
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	r.Body.Close()

	resp, created, err := rt.service.CreateIntent(r.Context(), principalID(r), req)
	if err != nil {
		switch err {
		case service.ErrInvalidIntent:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "order_code and idem_key are required")
		case service.ErrMethodNotFound:
			rt.writeError(w, r, http.StatusNotFound, "payment_method_not_found", "payment method not found")
		case service.ErrOrderNotFound:
			rt.writeError(w, r, http.StatusNotFound, "order_not_found", "order not found")
		case service.ErrIntentConflict:
			rt.writeError(w, r, http.StatusConflict, "intent_conflict", "idempotency key reused for a different intent")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}

	if !created {
		rt.writeJSON(w, http.StatusOK, resp)
		return
	}
	w.Header().Set("Location", "/api/v1/payments/intents/"+resp.Code)
	rt.writeJSON(w, http.StatusCreated, resp)
}

func (rt *Router) handleGetIntent(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "intent code is required")
		return
	}
	resp, err := rt.service.GetIntent(r.Context(), principalID(r), code)
	if err != nil {
		if err == service.ErrIntentNotFound {
			rt.writeError(w, r, http.StatusNotFound, "intent_not_found", "payment intent not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleAdminListIntents(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r)
	intents, err := rt.service.AdminListIntents(r.Context(), r.URL.Query().Get("status"), pageSize, (page-1)*pageSize)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if intents == nil {
		intents = []schema.IntentResponse{}
	}
	total, err := rt.service.AdminCountIntents(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": intents, "page": page, "page_size": pageSize,
		"total": total, "has_next": page*pageSize < total,
	})
}

func (rt *Router) handleMarkComplete(w http.ResponseWriter, r *http.Request) {
	rt.handleMark(w, r, true)
}

func (rt *Router) handleMarkFailed(w http.ResponseWriter, r *http.Request) {
	rt.handleMark(w, r, false)
}

func (rt *Router) handleMark(w http.ResponseWriter, r *http.Request, succeeded bool) {
	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "intent code is required")
		return
	}
	var req schema.MarkIntentRequest
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			r.Body.Close()
			rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}
	}
	r.Body.Close()

	resp, err := rt.service.MarkIntent(r.Context(), principalID(r), code, succeeded, req.Note)
	if err != nil {
		switch err {
		case service.ErrIntentNotFound:
			rt.writeError(w, r, http.StatusNotFound, "intent_not_found", "payment intent not found")
		case service.ErrIntentTerminal:
			rt.writeError(w, r, http.StatusConflict, "intent_terminal", "payment intent is already terminal")
		case service.ErrExceedsBalance:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "payment_exceeds_balance", "payment exceeds outstanding balance")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func parsePagination(r *http.Request) (page, pageSize int) {
	page, pageSize = 1, 20
	if v, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && v >= 1 {
		page = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil && v >= 1 {
		pageSize = v
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func (rt *Router) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (rt *Router) writeError(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	body := map[string]string{"detail": detail, "code": code}
	if requestID := middleware.GetReqID(r.Context()); requestID != "" {
		body["request_id"] = requestID
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
