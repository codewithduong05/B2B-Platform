package router

import (
	"encoding/json"
	"net/http"

	"github.com/atlas-platform/backend/internal/modules/reports/schema"
	"github.com/atlas-platform/backend/internal/modules/reports/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

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
	})
}

func (rt *Router) handleSales(w http.ResponseWriter, r *http.Request) {
	groupBy := r.URL.Query().Get("group_by")
	items, err := rt.service.SalesReport(r.Context(), groupBy)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, items)
}

func (rt *Router) handleBuyers(w http.ResponseWriter, r *http.Request) {
	items, err := rt.service.BuyerReport(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, items)
}

func (rt *Router) handleProducts(w http.ResponseWriter, r *http.Request) {
	items, err := rt.service.ProductReport(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, items)
}

func (rt *Router) handleSuppliers(w http.ResponseWriter, r *http.Request) {
	items, err := rt.service.SupplierReport(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, items)
}

func (rt *Router) handlePromotions(w http.ResponseWriter, r *http.Request) {
	items, err := rt.service.PromotionReport(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, items)
}

func (rt *Router) handleOperations(w http.ResponseWriter, r *http.Request) {
	resp, err := rt.service.OperationsReport(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleFinance(w http.ResponseWriter, r *http.Request) {
	resp, err := rt.service.FinanceReport(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleListExports(w http.ResponseWriter, r *http.Request) {
	jobs, err := rt.service.ListExportJobs(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, jobs)
}

func (rt *Router) handleCreateExport(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateExportRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	job, err := rt.service.CreateExportJob(r.Context(), req)
	if err != nil {
		if err == service.ErrInvalidInput {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "report_type is required")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Location", "/api/v1/admin/reports/exports")
	rt.writeJSON(w, http.StatusCreated, job)
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
