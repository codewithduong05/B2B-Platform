package router

// Suppliers HTTP surface. POST /suppliers/apply and GET /partners/track
// pattern: public application + supplier self-service are principal-based;
// everything under /admin is staff-gated by the injected admin middleware.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	commerce_service "github.com/atlas-platform/backend/internal/modules/commerce/service"
	"github.com/atlas-platform/backend/internal/modules/suppliers/schema"
	"github.com/atlas-platform/backend/internal/modules/suppliers/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type principalKey string

const principalCtxKey principalKey = "suppliers.principal_id"

func WithPrincipalID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, principalCtxKey, id)
}

func PrincipalIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(principalCtxKey).(int64)
	return id
}

type Router struct {
	router  chi.Router
	service *service.SupplierService
}

func New(svc *service.SupplierService) *Router {
	return &Router{router: chi.NewRouter(), service: svc}
}

func (rt *Router) ChiRouter() chi.Router {
	return rt.router
}

func (rt *Router) RegisterRoutes(authMiddleware, adminMiddleware func(http.Handler) http.Handler) {
	rt.router.Route("/suppliers", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Post("/apply", rt.handleApply)
	})

	rt.router.Route("/supplier", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Get("/me", rt.handleMyProfile)
		r.Patch("/me", rt.handleUpdateMyProfile)
		r.Get("/me/orders", rt.handleMyOrders)
		r.Patch("/me/orders/{code}", rt.handleAcknowledgeOrder)
		r.Patch("/me/stock", rt.handlePushStock)
	})

	rt.router.Route("/admin/suppliers", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		if adminMiddleware != nil {
			r.Use(adminMiddleware)
		}
		r.Get("/", rt.handleAdminList)
		r.Post("/{id}/approve", rt.handleAdminApprove)
		r.Get("/{id}/contracts", rt.handleAdminGetContracts)
		r.Put("/{id}/contracts", rt.handleAdminSetContract)
	})
}

func principalID(r *http.Request) int64 {
	return PrincipalIDFromContext(r.Context())
}

func (rt *Router) handleApply(w http.ResponseWriter, r *http.Request) {
	var req schema.ApplySupplierRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	// Authenticated applicants link automatically; anonymous stay unlinked
	// until staff attaches a user at approval.
	resp, err := rt.service.Apply(r.Context(), principalID(r), req)
	if err != nil {
		if err == service.ErrInvalidSupplier {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "company name is required")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Location", "/api/v1/admin/suppliers")
	rt.writeJSON(w, http.StatusCreated, resp)
}

func (rt *Router) handleMyProfile(w http.ResponseWriter, r *http.Request) {
	resp, err := rt.service.MyProfile(r.Context(), principalID(r))
	if err != nil {
		if err == service.ErrSupplierNotFound {
			rt.writeError(w, r, http.StatusNotFound, "supplier_not_found", "supplier profile not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleUpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	var req schema.UpdateSupplierRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	resp, err := rt.service.UpdateMyProfile(r.Context(), principalID(r), req)
	if err != nil {
		switch err {
		case service.ErrSupplierNotFound:
			rt.writeError(w, r, http.StatusNotFound, "supplier_not_found", "supplier profile not found")
		case service.ErrInvalidSupplier:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid supplier fields")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func parseAdminID(w http.ResponseWriter, r *http.Request, rt *Router) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_id", "invalid numeric id")
		return 0, false
	}
	return id, true
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

func (rt *Router) handleAdminList(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	rows, total, err := rt.service.ListSuppliers(r.Context(), r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if rows == nil {
		rows = []schema.SupplierResponse{}
	}
	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": rows, "page": page, "page_size": limit,
		"total": total, "has_next": page*limit < total,
	})
}

func (rt *Router) handleAdminApprove(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	var req schema.ApproveSupplierRequest
	if r.ContentLength != 0 {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}
	} else {
		defer r.Body.Close()
	}
	resp, err := rt.service.Approve(r.Context(), principalID(r), id, req.UserID)
	if err != nil {
		switch err {
		case service.ErrSupplierNotFound:
			rt.writeError(w, r, http.StatusNotFound, "supplier_not_found", "supplier not found")
		case service.ErrSupplierState:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "supplier_state", "only applied suppliers can be approved")
		case service.ErrInvalidSupplier:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid user reference")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleAdminGetContracts(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	rows, err := rt.service.GetContracts(r.Context(), id)
	if err != nil {
		if err == service.ErrSupplierNotFound {
			rt.writeError(w, r, http.StatusNotFound, "supplier_not_found", "supplier not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if rows == nil {
		rows = []schema.ContractResponse{}
	}
	rt.writeJSON(w, http.StatusOK, rows)
}

func (rt *Router) handleAdminSetContract(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	var req schema.SetContractRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	resp, err := rt.service.SetContract(r.Context(), principalID(r), id, req)
	if err != nil {
		switch err {
		case service.ErrSupplierNotFound:
			rt.writeError(w, r, http.StatusNotFound, "supplier_not_found", "supplier not found")
		case service.ErrInvalidContract:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "terms are required with a valid window")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	w.Header().Set("Location", "/api/v1/admin/suppliers")
	rt.writeJSON(w, http.StatusCreated, resp)
}

func (rt *Router) handleMyOrders(w http.ResponseWriter, r *http.Request) {
	resp, err := rt.service.MyOrders(r.Context(), principalID(r))
	if err != nil {
		if errors.Is(err, service.ErrSupplierNotFound) || errors.Is(err, service.ErrPortalForbidden) {
			rt.writeError(w, r, http.StatusNotFound, "supplier_not_found", "supplier profile not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleAcknowledgeOrder(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "order code is required")
		return
	}
	var req schema.AcknowledgeOrderRequest
	if r.ContentLength != 0 {
		defer r.Body.Close()
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	resp, err := rt.service.AcknowledgeOrder(r.Context(), principalID(r), code, req.Note)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			rt.writeError(w, r, http.StatusNotFound, "supplier_not_found", "order not found or not owned")
		case errors.Is(err, commerce_service.ErrInvalidTransition):
			rt.writeError(w, r, http.StatusUnprocessableEntity, "invalid_transition", err.Error())
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handlePushStock(w http.ResponseWriter, r *http.Request) {
	var req schema.PushStockRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	resp, err := rt.service.PushStock(r.Context(), principalID(r), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSupplierNotFound):
			rt.writeError(w, r, http.StatusNotFound, "supplier_not_found", "product or supplier not found")
		case errors.Is(err, service.ErrInvalidStock):
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid stock parameters")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
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
