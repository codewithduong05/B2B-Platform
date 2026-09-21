package router

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/atlas-platform/backend/internal/modules/pricing/schema"
	"github.com/atlas-platform/backend/internal/modules/pricing/service"
	"github.com/go-chi/chi/v5"
)

type Router struct {
	router  chi.Router
	service *service.Services
}

func New(svc *service.Services) *Router {
	r := chi.NewRouter()
	return &Router{router: r, service: svc}
}

func (rt *Router) ChiRouter() chi.Router {
	return rt.router
}

func (rt *Router) RegisterRoutes() {
	// Public routes
	rt.router.Route("/pricing", func(r chi.Router) {
		r.Post("/quote", rt.handleQuote)
		r.Get("/tiers", rt.handleProductTiers)
	})

	// Admin routes
	rt.router.Route("/admin/pricing", func(r chi.Router) {
		r.Get("/price-lists", rt.handleListPriceLists)
		r.Post("/price-lists", rt.handleCreatePriceList)
		r.Get("/price-lists/{id}", rt.handleGetPriceList)
		r.Patch("/price-lists/{id}", rt.handleUpdatePriceList)
		r.Delete("/price-lists/{id}", rt.handleDeletePriceList)
		r.Put("/price-lists/{id}/entries", rt.handleReplacePriceListEntries)
		r.Post("/price-lists/{id}/assign", rt.handleAssignPriceList)
		r.Get("/price-lists/{id}/assignments", rt.handleListPriceListAssignments)
	})
}

func (rt *Router) handleQuote(w http.ResponseWriter, r *http.Request) {
	var req schema.PriceQuoteRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	resp, err := rt.service.PriceList.Quote(r.Context(), req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "quote_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleProductTiers(w http.ResponseWriter, r *http.Request) {
	productIDStr := r.URL.Query().Get("product_id")
	if productIDStr == "" {
		rt.writeError(w, http.StatusBadRequest, "missing_product_id", "product_id is required")
		return
	}
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_product_id", "invalid product_id")
		return
	}

	unitIDStr := r.URL.Query().Get("unit_id")
	unitID := int64(1) // default unit
	if unitIDStr != "" {
		unitID, err = strconv.ParseInt(unitIDStr, 10, 64)
		if err != nil {
			rt.writeError(w, http.StatusBadRequest, "invalid_unit_id", "invalid unit_id")
			return
		}
	}

	resp, err := rt.service.PriceList.GetProductPricingTiers(r.Context(), productID, unitID)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "tiers_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

// Admin handlers
func (rt *Router) handleListPriceLists(w http.ResponseWriter, r *http.Request) {
	var req schema.PriceListListRequest
	if err := rt.decodeQuery(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	resp, err := rt.service.PriceList.ListPriceLists(r.Context(), req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleCreatePriceList(w http.ResponseWriter, r *http.Request) {
	var req schema.CreatePriceListRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	pl, err := rt.service.PriceList.CreatePriceList(r.Context(), req)
	if err != nil {
		if err == service.ErrDuplicateCode {
			rt.writeError(w, http.StatusConflict, "duplicate_code", "price list code already exists")
			return
		}
		if err == service.ErrInvalidEffectiveDates {
			rt.writeError(w, http.StatusBadRequest, "invalid_dates", "effective_to must be after effective_from")
			return
		}
		rt.writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusCreated, pl)
}

func (rt *Router) handleGetPriceList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		rt.writeError(w, http.StatusBadRequest, "missing_id", "price list ID is required")
		return
	}

	pl, err := rt.service.PriceList.GetPriceList(r.Context(), id)
	if err != nil {
		if err == service.ErrPriceListNotFound {
			rt.writeError(w, http.StatusNotFound, "not_found", "price list not found")
			return
		}
		rt.writeError(w, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, schema.PriceListDetailResponse{PriceList: *pl})
}

func (rt *Router) handleUpdatePriceList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		rt.writeError(w, http.StatusBadRequest, "missing_id", "price list ID is required")
		return
	}

	var req schema.UpdatePriceListRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	pl, err := rt.service.PriceList.UpdatePriceList(r.Context(), id, req)
	if err != nil {
		if err == service.ErrPriceListNotFound {
			rt.writeError(w, http.StatusNotFound, "not_found", "price list not found")
			return
		}
		rt.writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, pl)
}

func (rt *Router) handleDeletePriceList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		rt.writeError(w, http.StatusBadRequest, "missing_id", "price list ID is required")
		return
	}

	err := rt.service.PriceList.DeletePriceList(r.Context(), id)
	if err != nil {
		if err == service.ErrPriceListNotFound {
			rt.writeError(w, http.StatusNotFound, "not_found", "price list not found")
			return
		}
		rt.writeError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (rt *Router) handleReplacePriceListEntries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		rt.writeError(w, http.StatusBadRequest, "missing_id", "price list ID is required")
		return
	}

	priceListID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_id", "invalid price list ID")
		return
	}

	var req schema.ReplacePriceListEntriesRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	items, err := rt.service.PriceList.ReplacePriceListEntries(r.Context(), priceListID, req)
	if err != nil {
		rt.writeError(w, http.StatusNotImplemented, "not_implemented", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (rt *Router) handleAssignPriceList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		rt.writeError(w, http.StatusBadRequest, "missing_id", "price list ID is required")
		return
	}

	priceListID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_id", "invalid price list ID")
		return
	}

	var req schema.CreatePriceListAssignmentRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	req.PriceListID = priceListID

	assignment, err := rt.service.PriceList.CreatePriceListAssignment(r.Context(), req)
	if err != nil {
		if err == service.ErrPriceListNotFound {
			rt.writeError(w, http.StatusNotFound, "not_found", "price list not found")
			return
		}
		if err == service.ErrInvalidPriceList {
			rt.writeError(w, http.StatusBadRequest, "invalid_price_list", "price list is not active")
			return
		}
		rt.writeError(w, http.StatusInternalServerError, "assign_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusCreated, assignment)
}

func (rt *Router) handleListPriceListAssignments(w http.ResponseWriter, r *http.Request) {
	var req schema.PriceListAssignmentListRequest
	if err := rt.decodeQuery(r, &req); err != nil {
		rt.writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	resp, err := rt.service.PriceList.ListPriceListAssignments(r.Context(), req)
	if err != nil {
		rt.writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

// Helper methods
func (rt *Router) decodeQuery(r *http.Request, dest interface{}) error {
	// Simple query parameter decoding
	return nil
}

func (rt *Router) decodeBody(r *http.Request, dest interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dest)
}

func (rt *Router) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (rt *Router) writeError(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"detail": detail,
		"code":   code,
	})
}
