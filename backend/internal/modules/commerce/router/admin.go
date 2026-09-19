package router

// Staff order/shipment/invoice surface (05: /admin/*). All routes here run
// behind the injected admin middleware; integer IDs per Rule 4.

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/atlas-platform/backend/internal/modules/commerce/repository"
	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
	"github.com/atlas-platform/backend/internal/modules/commerce/service"
	"github.com/go-chi/chi/v5"
)

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

func staffActor(r *http.Request) int64 {
	return PrincipalIDFromContext(r.Context())
}

func (rt *Router) writeEnvelope(w http.ResponseWriter, items interface{}, page, pageSize, total int) {
	if items == nil {
		items = []interface{}{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": items, "page": page, "page_size": pageSize,
		"total": total, "has_next": page*pageSize < total,
	})
}

func (rt *Router) handleAdminListOrders(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	orders, total, err := rt.service.AdminListOrders(r.Context(), r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeEnvelope(w, orders, page, limit, total)
}

func (rt *Router) handleAdminGetOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	detail, err := rt.service.AdminGetOrder(r.Context(), id)
	if err != nil {
		if err == service.ErrOrderNotFound {
			rt.writeError(w, r, http.StatusNotFound, "order_not_found", "order not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, detail)
}

func (rt *Router) decodeAdminBody(w http.ResponseWriter, r *http.Request, dest interface{}) bool {
	if r.ContentLength == 0 {
		return true
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return false
	}
	return true
}

func (rt *Router) handleAdminAddNote(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	var req struct {
		Note string `json:"note"`
	}
	if !rt.decodeAdminBody(w, r, &req) {
		return
	}
	order, err := rt.service.AddOrderNote(r.Context(), id, staffActor(r), req.Note)
	if err != nil {
		rt.writeAdminOrderError(w, r, err)
		return
	}
	rt.writeJSON(w, http.StatusOK, map[string]string{"code": order.Code, "status": order.Status})
}

func (rt *Router) handleAdminHold(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if !rt.decodeAdminBody(w, r, &req) {
		return
	}
	order, err := rt.service.HoldOrder(r.Context(), id, staffActor(r), req.Reason)
	if err != nil {
		rt.writeAdminOrderError(w, r, err)
		return
	}
	rt.writeJSON(w, http.StatusOK, map[string]string{"code": order.Code, "status": order.Status})
}

func (rt *Router) handleAdminReleaseHold(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if !rt.decodeAdminBody(w, r, &req) {
		return
	}
	order, err := rt.service.ReleaseHoldOrder(r.Context(), id, staffActor(r), req.Reason)
	if err != nil {
		rt.writeAdminOrderError(w, r, err)
		return
	}
	rt.writeJSON(w, http.StatusOK, map[string]string{"code": order.Code, "status": order.Status})
}

func (rt *Router) handleAdminTransition(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	var req struct {
		ToStatus string `json:"to_status"`
		Reason   string `json:"reason"`
	}
	if !rt.decodeAdminBody(w, r, &req) {
		return
	}
	if req.ToStatus == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "to_status is required")
		return
	}
	// Cancellation carries compensation (reservation release) and must go
	// through CancelOrder; all other transitions use the generic engine.
	var order *repository.Order
	var err error
	if req.ToStatus == service.OrderStatusCancelled {
		order, err = rt.service.CancelOrder(r.Context(), id, staffActor(r), req.Reason)
	} else {
		order, err = rt.service.TransitionOrder(r.Context(), id, staffActor(r), req.ToStatus, req.Reason)
	}
	if err != nil {
		rt.writeAdminOrderError(w, r, err)
		return
	}
	rt.writeJSON(w, http.StatusOK, map[string]string{"code": order.Code, "status": order.Status})
}

func (rt *Router) writeAdminOrderError(w http.ResponseWriter, r *http.Request, err error) {
	switch err {
	case service.ErrOrderNotFound:
		rt.writeError(w, r, http.StatusNotFound, "order_not_found", "order not found")
	case service.ErrInvalidTransition:
		rt.writeError(w, r, http.StatusUnprocessableEntity, "invalid_transition", "invalid order status transition")
	case service.ErrOrderOnHold:
		rt.writeError(w, r, http.StatusConflict, "order_on_hold", "order is on hold")
	case service.ErrOrderConflict:
		rt.writeError(w, r, http.StatusConflict, "order_conflict", "order state changed underneath")
	case service.ErrIncompleteFulfillment:
		rt.writeError(w, r, http.StatusUnprocessableEntity, "incomplete_fulfillment", "order lines are not fully shipped")
	default:
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
	}
}

func (rt *Router) handleAdminCreateShipment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	var req schema.CreateShipmentRequest
	if !rt.decodeAdminBody(w, r, &req) {
		return
	}
	var items []service.ShipmentLineInput
	for _, l := range req.Lines {
		items = append(items, service.ShipmentLineInput{OrderLineID: l.OrderLineID, Quantity: l.Quantity})
	}
	resp, err := rt.service.CreateShipment(r.Context(), staffActor(r), id, items, req.Carrier, req.TrackingCode)
	if err != nil {
		switch err {
		case service.ErrOrderNotFound:
			rt.writeError(w, r, http.StatusNotFound, "order_not_found", "order not found")
		case service.ErrInvalidQuantity:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_quantity", "invalid shipment line quantity")
		case service.ErrOrderOnHold:
			rt.writeError(w, r, http.StatusConflict, "order_on_hold", "order is on hold")
		case service.ErrInvalidTransition:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "invalid_transition", "order cannot be shipped in its current state")
		case service.ErrCartLineNotFound:
			rt.writeError(w, r, http.StatusNotFound, "order_line_not_found", "order line not found")
		case service.ErrOrderConflict:
			rt.writeError(w, r, http.StatusConflict, "order_conflict", "order state changed underneath")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	w.Header().Set("Location", "/api/v1/admin/shipments")
	rt.writeJSON(w, http.StatusCreated, resp)
}

func (rt *Router) handleAdminListShipments(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	var orderID int64
	if v := r.URL.Query().Get("order_id"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
			orderID = parsed
		}
	}
	shipments, total, err := rt.service.AdminListShipments(r.Context(), orderID, limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeEnvelope(w, shipments, page, limit, total)
}

func (rt *Router) handleAdminUpdateShipment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	var req schema.UpdateShipmentRequest
	if !rt.decodeAdminBody(w, r, &req) {
		return
	}
	resp, err := rt.service.UpdateShipment(r.Context(), staffActor(r), id, req.Carrier, req.TrackingCode, req.Status)
	if err != nil {
		switch err {
		case service.ErrShipmentNotFound:
			rt.writeError(w, r, http.StatusNotFound, "shipment_not_found", "shipment not found")
		case service.ErrInvalidTransition:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "invalid_transition", "invalid shipment status transition")
		case service.ErrOrderConflict:
			rt.writeError(w, r, http.StatusConflict, "order_conflict", "shipment state changed underneath")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleAdminListInvoices(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	var orderID int64
	if v := r.URL.Query().Get("order_id"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
			orderID = parsed
		}
	}
	invoices, total, err := rt.service.AdminListInvoices(r.Context(), orderID, limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeEnvelope(w, invoices, page, limit, total)
}

func (rt *Router) handleAdminCreateInvoice(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateInvoiceRequest
	if !rt.decodeAdminBody(w, r, &req) {
		return
	}
	if req.OrderID <= 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "order_id is required")
		return
	}
	resp, err := rt.service.CreateInvoice(r.Context(), staffActor(r), req.OrderID)
	if err != nil {
		switch err {
		case service.ErrOrderNotFound:
			rt.writeError(w, r, http.StatusNotFound, "order_not_found", "order not found")
		case service.ErrInvalidTransition:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "invalid_transition", "order cannot be invoiced in its current state")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	w.Header().Set("Location", "/api/v1/admin/invoices")
	rt.writeJSON(w, http.StatusCreated, resp)
}

func (rt *Router) handleAdminIssueInvoice(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	resp, err := rt.service.IssueInvoice(r.Context(), staffActor(r), id)
	if err != nil {
		switch err {
		case service.ErrInvoiceNotFound:
			rt.writeError(w, r, http.StatusNotFound, "invoice_not_found", "invoice not found")
		case service.ErrInvalidTransition:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "invalid_transition", "only draft invoices can be issued")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleAdminVoidInvoice(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	resp, err := rt.service.VoidInvoice(r.Context(), staffActor(r), id)
	if err != nil {
		switch err {
		case service.ErrInvoiceNotFound:
			rt.writeError(w, r, http.StatusNotFound, "invoice_not_found", "invoice not found")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}
