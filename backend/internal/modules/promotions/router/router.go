package router

// Promotions HTTP surface. Buyer reads under auth; staff management grouped
// under the injected admin middleware. The cart voucher endpoint lives on
// the commerce router (single /commerce mount — Chi panics on duplicates)
// and delegates to the commerce service.

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/atlas-platform/backend/internal/modules/promotions/schema"
	"github.com/atlas-platform/backend/internal/modules/promotions/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type principalKey string

const principalCtxKey principalKey = "promotions.principal_id"

func WithPrincipalID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, principalCtxKey, id)
}

func PrincipalIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(principalCtxKey).(int64)
	return id
}

type Router struct {
	router  chi.Router
	service *service.PromotionService
}

func New(svc *service.PromotionService) *Router {
	return &Router{router: chi.NewRouter(), service: svc}
}

func (rt *Router) ChiRouter() chi.Router {
	return rt.router
}

func (rt *Router) RegisterRoutes(authMiddleware, adminMiddleware func(http.Handler) http.Handler) {
	rt.router.Route("/promotions", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Get("/active", rt.handleActive)
	})

	rt.router.Route("/admin/promotions", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		if adminMiddleware != nil {
			r.Use(adminMiddleware)
		}
		r.Get("/", rt.handleAdminList)
		r.Post("/", rt.handleAdminCreate)
		r.Patch("/{id}", rt.handleAdminUpdate)
		r.Post("/{id}/publish", rt.handleAdminPublish)
		r.Get("/reports", rt.handleAdminReports)
	})
}

func principalID(r *http.Request) int64 {
	if id := PrincipalIDFromContext(r.Context()); id != 0 {
		return id
	}
	return 1
}

func (rt *Router) handleActive(w http.ResponseWriter, r *http.Request) {
	promos, err := rt.service.ActivePromotions(r.Context(), principalID(r))
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if promos == nil {
		promos = []schema.PromotionResponse{}
	}
	rt.writeJSON(w, http.StatusOK, promos)
}

func (rt *Router) handleAdminList(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	promos, total, err := rt.service.ListPromotions(r.Context(), r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if promos == nil {
		promos = []schema.PromotionResponse{}
	}
	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": promos, "page": page, "page_size": limit,
		"total": total, "has_next": page*limit < total,
	})
}

func (rt *Router) handleAdminCreate(w http.ResponseWriter, r *http.Request) {
	var req schema.CreatePromotionRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	resp, err := rt.service.CreatePromotion(r.Context(), req)
	if err != nil {
		if err == service.ErrInvalidPromotion {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_promotion", "invalid promotion fields")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Location", "/api/v1/admin/promotions")
	rt.writeJSON(w, http.StatusCreated, resp)
}

func parseAdminID(w http.ResponseWriter, r *http.Request, rt *Router) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_id", "invalid numeric id")
		return 0, false
	}
	return id, true
}

func (rt *Router) handleAdminUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	var req schema.UpdatePromotionRequest
	if r.ContentLength != 0 {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}
	} else {
		defer r.Body.Close()
	}
	resp, err := rt.service.UpdatePromotion(r.Context(), id, req)
	if err != nil {
		switch err {
		case service.ErrPromotionNotFound:
			rt.writeError(w, r, http.StatusNotFound, "promotion_not_found", "promotion not found")
		case service.ErrInvalidPromotion:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_promotion", "invalid promotion fields")
		case service.ErrInvalidTransition:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "invalid_transition", "invalid promotion status transition")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleAdminPublish(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	resp, err := rt.service.PublishPromotion(r.Context(), id)
	if err != nil {
		switch err {
		case service.ErrPromotionNotFound:
			rt.writeError(w, r, http.StatusNotFound, "promotion_not_found", "promotion not found")
		case service.ErrInvalidTransition:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "invalid_transition", "only draft promotions can be published")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleAdminReports(w http.ResponseWriter, r *http.Request) {
	rows, err := rt.service.Report(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if rows == nil {
		rows = []schema.ReportRow{}
	}
	rt.writeJSON(w, http.StatusOK, rows)
}

func parsePage(r *http.Request) (page, limit, offset int) {
	page, limit = 1, 20
	if v, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && v >= 1 {
		page = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil && v >= 1 {
		limit = v
	}
	if limit > 200 {
		limit = 200
	}
	return page, limit, (page - 1) * limit
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
