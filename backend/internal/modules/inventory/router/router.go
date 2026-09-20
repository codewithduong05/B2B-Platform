package router

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/atlas-platform/backend/internal/modules/inventory/schema"
	"github.com/atlas-platform/backend/internal/modules/inventory/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type principalKey string

const (
	principalCtxKey principalKey = "inventory.principal_id"
)

// WithPrincipalID injects the authenticated principal id into the request context.
// The auth middleware (once implemented) calls this; until then no value is set and
// handlers resolve the principal to 0 (nullable adjusted_by).
func WithPrincipalID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, principalCtxKey, id)
}

// PrincipalIDFromContext returns the authenticated principal id, or 0 when absent.
func PrincipalIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(principalCtxKey).(int64)
	return id
}

type Router struct {
	router  chi.Router
	service *service.InventoryService
}

func New(svc *service.InventoryService) *Router {
	r := chi.NewRouter()
	return &Router{router: r, service: svc}
}

func (rt *Router) ChiRouter() chi.Router {
	return rt.router
}

func (rt *Router) RegisterRoutes(authMiddleware, adminMiddleware func(http.Handler) http.Handler) {
	rt.router.Route("/inventory", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}

		r.Get("/availability", rt.handleAvailability)
		r.Post("/reservations", rt.handleCreateReservation)
		r.Get("/reservations/{id}", rt.handleGetReservation)
		r.Get("/reservations/request/{request_id}", rt.handleGetReservationByRequestID)

		if adminMiddleware != nil {
			r.With(adminMiddleware).Post("/lots/{id}/quarantine", rt.handleQuarantineLot)
			r.With(adminMiddleware).Post("/lots/{id}/release", rt.handleReleaseLot)
			r.With(adminMiddleware).Get("/low-stock", rt.handleListLowStock)
			r.With(adminMiddleware).Get("/expiring", rt.handleListExpiring)
			r.With(adminMiddleware).Post("/adjust", rt.handleAdjustStock)
		} else {
			r.Post("/lots/{id}/quarantine", rt.handleQuarantineLot)
		}
	})
}

func (rt *Router) handleAvailability(w http.ResponseWriter, r *http.Request) {
	productIDsRaw := r.URL.Query().Get("product_ids")
	if strings.TrimSpace(productIDsRaw) == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_query", "product_ids is required")
		return
	}

	var productIDs []int64
	for _, part := range strings.Split(productIDsRaw, ",") {
		part = strings.TrimSpace(part)
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_query", "product_ids must be a comma-separated list of positive integers")
			return
		}
		productIDs = append(productIDs, id)
	}

	var supplierID *int64
	if raw := r.URL.Query().Get("supplier_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_query", "supplier_id must be a positive integer")
			return
		}
		supplierID = &id
	}

	summaries, err := rt.service.GetAvailability(r.Context(), productIDs, supplierID)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", "failed to retrieve availability")
		return
	}

	rt.writeJSON(w, http.StatusOK, summaries)
}

func (rt *Router) handleCreateReservation(w http.ResponseWriter, r *http.Request) {
	var req schema.ReserveStockRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if strings.TrimSpace(req.RequestID) == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "request_id is required")
		return
	}
	if len(req.RequestID) > 100 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request_id", "request_id must be 1-100 characters")
		return
	}
	if req.ProductID <= 0 || req.SupplierID <= 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "product_id and supplier_id are required")
		return
	}
	if req.Quantity < 1 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_quantity", "quantity must be at least 1")
		return
	}
	if req.ExpiresIn < 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "expires_in_minutes must be non-negative")
		return
	}

	result, err := rt.service.ReserveStockResult(r.Context(), req)
	if err != nil {
		switch err {
		case service.ErrInsufficientStock:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "insufficient_stock", "insufficient stock for the requested quantity")
		case service.ErrDuplicateRequest:
			rt.writeError(w, r, http.StatusConflict, "duplicate_request", "duplicate request key")
		case service.ErrStockLevelNotFound:
			rt.writeError(w, r, http.StatusNotFound, "stock_level_not_found", "stock level not found for the product and supplier")
		case service.ErrLotQuarantined:
			rt.writeError(w, r, http.StatusConflict, "lot_quarantined", "lot is quarantined")
		case service.ErrInvalidQuantity:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_quantity", "invalid quantity")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", "failed to create reservation")
		}
		return
	}

	resp := buildReservationResponse(result.Reservations)
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	} else {
		w.Header().Set("Location", "/api/v1/inventory/reservations/request/"+req.RequestID)
	}

	rt.writeJSON(w, status, resp)
}

func (rt *Router) handleGetReservation(w http.ResponseWriter, r *http.Request) {
	id, ok := rt.parsePositiveID(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	res, err := rt.service.GetReservation(r.Context(), id)
	if err != nil {
		if err == service.ErrReservationNotFound {
			rt.writeError(w, r, http.StatusNotFound, "reservation_not_found", "reservation not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", "failed to get reservation")
		return
	}

	rt.writeJSON(w, http.StatusOK, res)
}

func (rt *Router) handleGetReservationByRequestID(w http.ResponseWriter, r *http.Request) {
	requestID := strings.TrimSpace(chi.URLParam(r, "request_id"))
	if requestID == "" || len(requestID) > 100 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request_id", "request_id must be 1-100 characters")
		return
	}

	reservations, err := rt.service.GetReservationsByRequestID(r.Context(), requestID)
	if err != nil {
		if err == service.ErrReservationNotFound {
			rt.writeError(w, r, http.StatusNotFound, "reservation_not_found", "reservation not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", "failed to get reservation")
		return
	}

	rt.writeJSON(w, http.StatusOK, buildReservationResponse(reservations))
}

func (rt *Router) handleQuarantineLot(w http.ResponseWriter, r *http.Request) {
	lotID, ok := rt.parsePositiveID(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	var req schema.QuarantineLotRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "reason is required")
		return
	}

	req.LotID = lotID
	userID := PrincipalIDFromContext(r.Context())

	lot, err := rt.service.QuarantineLot(r.Context(), req, userID)
	if err != nil {
		if err == service.ErrLotNotFound {
			rt.writeError(w, r, http.StatusNotFound, "lot_not_found", "lot not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", "failed to quarantine lot")
		return
	}

	rt.writeJSON(w, http.StatusOK, lot)
}

func buildReservationResponse(reservations []schema.ReservationSummary) schema.ReservationResponse {
	resp := schema.ReservationResponse{
		Status:      "reserved",
		Allocations: reservations,
	}

	if len(reservations) == 0 {
		return resp
	}

	resp.RequestID = reservations[0].RequestID
	resp.Status = reservations[0].Status
	resp.ExpiresAt = reservations[0].ExpiresAt

	for _, res := range reservations {
		resp.Quantity += res.Quantity
		if res.ExpiresAt.Before(resp.ExpiresAt) {
			resp.ExpiresAt = res.ExpiresAt
		}
	}

	return resp
}

func (rt *Router) parsePositiveID(w http.ResponseWriter, r *http.Request, raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_id", "ID must be a positive integer")
		return 0, false
	}
	return id, true
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

func (rt *Router) writeError(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	body := map[string]string{
		"detail": detail,
		"code":   code,
	}

	if requestID := middleware.GetReqID(r.Context()); requestID != "" {
		body["request_id"] = requestID
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// handleReleaseLot releases a quarantined lot
func (rt *Router) handleReleaseLot(w http.ResponseWriter, r *http.Request) {
	lotID, ok := rt.parsePositiveID(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	var req schema.ReleaseLotRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "reason is required")
		return
	}

	req.LotID = lotID
	lot, err := rt.service.ReleaseLot(r.Context(), req)
	if err != nil {
		if err == service.ErrLotNotFound {
			rt.writeError(w, r, http.StatusNotFound, "lot_not_found", "lot not found or not quarantined")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", "failed to release lot")
		return
	}

	rt.writeJSON(w, http.StatusOK, lot)
}

// handleListLowStock returns lots below their safety stock threshold
func (rt *Router) handleListLowStock(w http.ResponseWriter, r *http.Request) {
	summaries, err := rt.service.ListLowStockLots(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", "failed to list low stock lots")
		return
	}
	if summaries == nil {
		summaries = []schema.LowStockLotSummary{}
	}
	rt.writeJSON(w, http.StatusOK, summaries)
}

// handleListExpiring returns lots expiring within the given horizon
func (rt *Router) handleListExpiring(w http.ResponseWriter, r *http.Request) {
	var horizonDays *int32
	if raw := r.URL.Query().Get("horizon_days"); raw != "" {
		if d, err := strconv.Atoi(raw); err == nil && d > 0 {
			d32 := int32(d)
			horizonDays = &d32
		}
	}

	summaries, err := rt.service.ListExpiringLots(r.Context(), horizonDays)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", "failed to list expiring lots")
		return
	}
	if summaries == nil {
		summaries = []schema.ExpiringLotSummary{}
	}
	rt.writeJSON(w, http.StatusOK, summaries)
}

// handleAdjustStock adjusts stock quantity for a lot
func (rt *Router) handleAdjustStock(w http.ResponseWriter, r *http.Request) {
	var req schema.AdjustStockRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if req.QuantityDelta == 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_quantity", "quantity_delta must not be zero")
		return
	}
	if strings.TrimSpace(req.ReasonCode) == "" || strings.TrimSpace(req.Reason) == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "reason_code and reason are required")
		return
	}

	userID := PrincipalIDFromContext(r.Context())
	adjustment, err := rt.service.AdjustStock(r.Context(), req, userID)
	if err != nil {
		if err == service.ErrInvalidQuantity {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_quantity", "quantity_delta must not be zero")
			return
		}
		if err == service.ErrInvalidInput {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "reason_code and reason are required")
			return
		}
		if err == service.ErrLotNotFound {
			rt.writeError(w, r, http.StatusNotFound, "lot_not_found", "lot not found")
			return
		}
		if err == service.ErrLotQuarantined {
			rt.writeError(w, r, http.StatusConflict, "lot_quarantined", "lot is quarantined")
			return
		}
		if err == service.ErrInvalidInput {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "reason_code and reason are required")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", "failed to adjust stock")
		return
	}

	rt.writeJSON(w, http.StatusOK, adjustment)
}
