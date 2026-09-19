package router

// Reports HTTP surface. All nine TASK-014 routes sit behind the injected
// auth and admin middleware; there is no buyer-facing report route.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/atlas-platform/backend/internal/modules/reports/schema"
	"github.com/atlas-platform/backend/internal/modules/reports/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type principalKey string

const principalCtxKey principalKey = "reports.principal_id"

func WithPrincipalID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, principalCtxKey, id)
}

func PrincipalIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(principalCtxKey).(int64)
	return id
}

type Router struct {
	router  chi.Router
	service *service.ReportsService
}

func New(svc *service.ReportsService) *Router {
	return &Router{router: chi.NewRouter(), service: svc}
}

func (rt *Router) ChiRouter() chi.Router {
	return rt.router
}

func (rt *Router) RegisterRoutes(authMiddleware, adminMiddleware func(http.Handler) http.Handler) {
	rt.router.Route("/admin/reports", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		if adminMiddleware != nil {
			r.Use(adminMiddleware)
		}
		r.Get("/sales", rt.handleSales)
		r.Get("/buyers", rt.handleBuyers)
		r.Get("/products", rt.handleProducts)
		r.Get("/suppliers", rt.handleSuppliers)
		r.Get("/promotions", rt.handlePromotions)
		r.Get("/operations", rt.handleOperations)
		r.Get("/finance", rt.handleFinance)
		r.Get("/exports", rt.handleListExports)
		r.Post("/exports", rt.handleCreateExport)
		r.Get("/exports/{code}/download", rt.handleDownloadExport)
	})
}

func (rt *Router) handleSales(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	q := r.URL.Query()
	items, total, err := rt.service.SalesReport(r.Context(), q.Get("group_by"), q.Get("from"), q.Get("to"), limit, offset)
	if err != nil {
		rt.writeServiceError(w, r, err)
		return
	}
	rt.writeJSON(w, http.StatusOK, envelope(items, page, limit, total))
}

func (rt *Router) handleBuyers(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	q := r.URL.Query()
	items, total, err := rt.service.BuyerReport(r.Context(), q.Get("from"), q.Get("to"), limit, offset)
	if err != nil {
		rt.writeServiceError(w, r, err)
		return
	}
	rt.writeJSON(w, http.StatusOK, envelope(items, page, limit, total))
}

func (rt *Router) handleProducts(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	q := r.URL.Query()
	items, total, err := rt.service.ProductReport(r.Context(), q.Get("from"), q.Get("to"), limit, offset)
	if err != nil {
		rt.writeServiceError(w, r, err)
		return
	}
	rt.writeJSON(w, http.StatusOK, envelope(items, page, limit, total))
}

func (rt *Router) handleSuppliers(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	q := r.URL.Query()
	items, total, err := rt.service.SupplierReport(r.Context(), q.Get("from"), q.Get("to"), limit, offset)
	if err != nil {
		rt.writeServiceError(w, r, err)
		return
	}
	rt.writeJSON(w, http.StatusOK, envelope(items, page, limit, total))
}

func (rt *Router) handlePromotions(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	q := r.URL.Query()
	items, total, err := rt.service.PromotionReport(r.Context(), q.Get("from"), q.Get("to"), limit, offset)
	if err != nil {
		rt.writeServiceError(w, r, err)
		return
	}
	rt.writeJSON(w, http.StatusOK, envelope(items, page, limit, total))
}

func (rt *Router) handleOperations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	resp, err := rt.service.OperationsReport(r.Context(), q.Get("from"), q.Get("to"))
	if err != nil {
		rt.writeServiceError(w, r, err)
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleFinance(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	resp, err := rt.service.FinanceReport(r.Context(), q.Get("from"), q.Get("to"))
	if err != nil {
		rt.writeServiceError(w, r, err)
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleListExports(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	q := r.URL.Query()
	items, total, err := rt.service.ListExportJobs(r.Context(), q.Get("status"), limit, offset)
	if err != nil {
		rt.writeServiceError(w, r, err)
		return
	}
	rt.writeJSON(w, http.StatusOK, envelope(items, page, limit, total))
}

func (rt *Router) handleCreateExport(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateExportRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	resp, err := rt.service.CreateExportJob(r.Context(), req, PrincipalIDFromContext(r.Context()))
	if err != nil {
		rt.writeServiceError(w, r, err)
		return
	}
	rt.writeJSON(w, http.StatusAccepted, resp)
}

func (rt *Router) handleDownloadExport(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "export code is required")
		return
	}
	job, err := rt.service.GetExportJobByCode(r.Context(), code)
	if err != nil {
		rt.writeError(w, r, http.StatusNotFound, "not_found", "export job not found")
		return
	}
	if job.Status != "completed" {
		rt.writeError(w, r, http.StatusConflict, "not_ready", "export job is not completed")
		return
	}
	if job.DownloadUrl == nil || *job.DownloadUrl == "" {
		rt.writeError(w, r, http.StatusNotFound, "no_file", "export file not available")
		return
	}
	rt.writeJSON(w, http.StatusOK, map[string]string{
		"code":         job.Code,
		"download_url": *job.DownloadUrl,
		"status":       job.Status,
	})
}

// parsePage follows the documented admin pagination envelope: 1-based page,
// page_size capped at 200 so no client can request an unbounded result.
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

func envelope(items interface{}, page, limit, total int) map[string]interface{} {
	return map[string]interface{}{
		"items":     items,
		"page":      page,
		"page_size": limit,
		"total":     total,
		"has_next":  page*limit < total,
	}
}

func (rt *Router) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, service.ErrInvalidInput) {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid report filters")
		return
	}
	rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
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
