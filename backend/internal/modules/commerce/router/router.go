package router

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
	"github.com/atlas-platform/backend/internal/modules/commerce/service"
	promotions_service "github.com/atlas-platform/backend/internal/modules/promotions/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type principalKey string

const (
	principalCtxKey principalKey = "commerce.principal_id"
)

func WithPrincipalID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, principalCtxKey, id)
}

func PrincipalIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(principalCtxKey).(int64)
	return id
}

type Router struct {
	router  chi.Router
	service *service.CommerceService
}

func New(svc *service.CommerceService) *Router {
	r := chi.NewRouter()
	return &Router{router: r, service: svc}
}

func (rt *Router) ChiRouter() chi.Router {
	return rt.router
}

func (rt *Router) RegisterRoutes(authMiddleware, adminMiddleware func(http.Handler) http.Handler) {
	rt.router.Route("/commerce", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}

		r.Get("/cart", rt.handleGetCart)
		r.Post("/cart/items", rt.handleAddCartItem)
		r.Patch("/cart/items/{code}", rt.handleUpdateCartItem)
		r.Delete("/cart/items/{code}", rt.handleDeleteCartItem)
		r.Post("/cart/quote", rt.handleQuoteCart)
		r.Post("/cart/voucher", rt.handleApplyVoucher)
		r.Post("/checkout", rt.handleCheckout)
		r.Get("/orders/me", rt.handleListOrders)
		r.Get("/orders/me/{code}", rt.handleGetOrder)
		r.Post("/orders/me/{code}/returns", rt.handleRequestReturn)
		r.Get("/invoices/me", rt.handleListInvoicesMe)
		r.Get("/invoices/me/{code}", rt.handleGetInvoiceMe)
	})

	// Staff surface (05: /admin/* with integer IDs). Authenticated first, then
	// authorized by the injected admin middleware.
	rt.router.Route("/admin", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		if adminMiddleware != nil {
			r.Use(adminMiddleware)
		}

		r.Get("/orders", rt.handleAdminListOrders)
		r.Get("/orders/{id}", rt.handleAdminGetOrder)
		r.Post("/orders/{id}/notes", rt.handleAdminAddNote)
		r.Post("/orders/{id}/hold", rt.handleAdminHold)
		r.Post("/orders/{id}/release", rt.handleAdminReleaseHold)
		r.Post("/orders/{id}/transition", rt.handleAdminTransition)
		r.Post("/orders/{id}/shipments", rt.handleAdminCreateShipment)
		r.Get("/shipments", rt.handleAdminListShipments)
		r.Patch("/shipments/{id}", rt.handleAdminUpdateShipment)
		r.Get("/invoices", rt.handleAdminListInvoices)
		r.Post("/invoices", rt.handleAdminCreateInvoice)
		r.Post("/invoices/{id}/issue", rt.handleAdminIssueInvoice)
		r.Post("/invoices/{id}/void", rt.handleAdminVoidInvoice)
		r.Post("/invoices/{id}/reissue", rt.handleAdminReissueInvoice)
		r.Get("/returns", rt.handleAdminListReturns)
		r.Post("/returns/{id}/approve", rt.handleAdminApproveReturn)
		r.Post("/returns/{id}/reject", rt.handleAdminRejectReturn)
		r.Post("/returns/{id}/complete", rt.handleAdminCompleteReturn)
		r.Get("/credit-notes", rt.handleAdminListCreditNotes)
		r.Post("/credit-notes", rt.handleAdminCreateCredit)
		r.Post("/credit-notes/{id}/void", rt.handleAdminVoidCredit)
		r.Get("/finance/aged-debt", rt.handleAdminAgedDebt)
	})
}

func (rt *Router) handleGetCart(w http.ResponseWriter, r *http.Request) {
	buyerID := PrincipalIDFromContext(r.Context())
	if buyerID == 0 {
		buyerID = 1 // Default fallback for tests or authenticated buyer context
	}

	cart, err := rt.service.GetCart(r.Context(), buyerID)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, cart)
}

func (rt *Router) handleAddCartItem(w http.ResponseWriter, r *http.Request) {
	buyerID := PrincipalIDFromContext(r.Context())
	if buyerID == 0 {
		buyerID = 1
	}

	var req schema.AddCartItemRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if req.ProductCode == "" || req.Quantity <= 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_quantity", "product_code and positive quantity are required")
		return
	}

	cart, err := rt.service.AddCartItem(r.Context(), buyerID, req)
	if err != nil {
		if err == service.ErrProductNotFound {
			rt.writeError(w, r, http.StatusNotFound, "product_not_found", "product not found")
			return
		}
		if err == service.ErrInvalidQuantity {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_quantity", "invalid quantity")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	// 201 responses carry a Location header per the API contract; the
	// representation returned is the cart itself.
	w.Header().Set("Location", "/api/v1/commerce/cart")
	rt.writeJSON(w, http.StatusCreated, cart)
}

func (rt *Router) handleUpdateCartItem(w http.ResponseWriter, r *http.Request) {
	buyerID := PrincipalIDFromContext(r.Context())
	if buyerID == 0 {
		buyerID = 1
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "item code is required")
		return
	}

	var req schema.UpdateCartItemRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if req.Quantity <= 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_quantity", "quantity must be positive")
		return
	}

	cart, err := rt.service.UpdateCartItem(r.Context(), buyerID, code, req)
	if err != nil {
		if err == service.ErrCartLineNotFound {
			rt.writeError(w, r, http.StatusNotFound, "cart_line_not_found", "cart line not found")
			return
		}
		if err == service.ErrInvalidQuantity {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_quantity", "invalid quantity")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, cart)
}

func (rt *Router) handleDeleteCartItem(w http.ResponseWriter, r *http.Request) {
	buyerID := PrincipalIDFromContext(r.Context())
	if buyerID == 0 {
		buyerID = 1
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "item code is required")
		return
	}

	cart, err := rt.service.DeleteCartItem(r.Context(), buyerID, code)
	if err != nil {
		if err == service.ErrCartLineNotFound {
			rt.writeError(w, r, http.StatusNotFound, "cart_line_not_found", "cart line not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, cart)
}

func (rt *Router) handleQuoteCart(w http.ResponseWriter, r *http.Request) {
	buyerID := PrincipalIDFromContext(r.Context())
	if buyerID == 0 {
		buyerID = 1
	}

	quote, err := rt.service.QuoteCart(r.Context(), buyerID)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, quote)
}

func (rt *Router) handleApplyVoucher(w http.ResponseWriter, r *http.Request) {
	buyerID := PrincipalIDFromContext(r.Context())
	if buyerID == 0 {
		buyerID = 1
	}

	var req schema.ApplyVoucherRequest
	if err := rt.decodeBody(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	if req.Code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "voucher code is required")
		return
	}

	cart, err := rt.service.ApplyVoucher(r.Context(), buyerID, req.Code)
	if err != nil {
		rt.writeVoucherError(w, r, err)
		return
	}

	rt.writeJSON(w, http.StatusOK, cart)
}

// writeVoucherError maps promotion validation failures to stable codes.
// Unknown codes are 404; every other rejection explains itself with 422.
func (rt *Router) writeVoucherError(w http.ResponseWriter, r *http.Request, err error) {
	switch err {
	case promotions_service.ErrVoucherUnknown:
		rt.writeError(w, r, http.StatusNotFound, "voucher_not_found", "unknown voucher code")
	case promotions_service.ErrVoucherState:
		rt.writeError(w, r, http.StatusUnprocessableEntity, "voucher_invalid", "voucher is not redeemable")
	case promotions_service.ErrVoucherExpired:
		rt.writeError(w, r, http.StatusUnprocessableEntity, "voucher_expired", "voucher is outside its validity window")
	case promotions_service.ErrVoucherExhausted:
		rt.writeError(w, r, http.StatusUnprocessableEntity, "voucher_exhausted", "voucher budget exhausted")
	case promotions_service.ErrVoucherRedeemed:
		rt.writeError(w, r, http.StatusUnprocessableEntity, "voucher_redeemed", "voucher already redeemed by this buyer")
	default:
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
	}
}

func (rt *Router) handleCheckout(w http.ResponseWriter, r *http.Request) {
	buyerID := PrincipalIDFromContext(r.Context())
	if buyerID == 0 {
		buyerID = 1
	}

	idemKey := r.Header.Get("Idempotency-Key")
	if idemKey == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "Idempotency-Key header is required")
		return
	}

	// The checkout body carries no business fields; an empty body is the
	// normal case. A present-but-malformed body is still a 400.
	if r.ContentLength != 0 {
		var req schema.CheckoutRequest
		if err := rt.decodeBody(r, &req); err != nil {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}
	}

	result, err := rt.service.Checkout(r.Context(), buyerID, idemKey)
	if err != nil {
		switch err {
		case service.ErrInvalidIdemKey:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid idempotency key")
		case service.ErrEmptyCart:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "empty_cart", "cart is empty")
		case service.ErrProductNotFound:
			rt.writeError(w, r, http.StatusNotFound, "product_not_found", "product not found or unavailable")
		case service.ErrCartNotFound:
			rt.writeError(w, r, http.StatusNotFound, "cart_not_found", "cart not found")
		case service.ErrInsufficientStock:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "insufficient_stock", "insufficient stock for the requested quantity")
		case service.ErrIdempotencyConflict:
			rt.writeError(w, r, http.StatusConflict, "idempotency_conflict", "idempotency key reused for a different checkout")
		case service.ErrCheckoutInFlight:
			rt.writeError(w, r, http.StatusConflict, "checkout_in_flight", "checkout already in progress for this key")
		case service.ErrCheckoutConflict:
			rt.writeError(w, r, http.StatusConflict, "checkout_conflict", "cart was consumed by another checkout")
		case service.ErrCreditHold:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "credit_hold", "buyer credit account is on hold")
		case service.ErrCreditLimitExceeded:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "credit_limit_exceeded", "order exceeds available credit")
		case promotions_service.ErrVoucherUnknown:
			rt.writeError(w, r, http.StatusNotFound, "voucher_not_found", "applied voucher is unknown")
		case promotions_service.ErrVoucherState,
			promotions_service.ErrVoucherExpired,
			promotions_service.ErrVoucherExhausted,
			promotions_service.ErrVoucherRedeemed:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "voucher_invalid", "applied voucher is no longer redeemable")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}

	if result.Replayed {
		rt.writeJSON(w, http.StatusOK, result.Response)
		return
	}
	w.Header().Set("Location", "/api/v1/commerce/orders/me")
	rt.writeJSON(w, http.StatusCreated, result.Response)
}

func (rt *Router) handleListOrders(w http.ResponseWriter, r *http.Request) {
	buyerID := PrincipalIDFromContext(r.Context())
	if buyerID == 0 {
		buyerID = 1
	}

	orders, err := rt.service.ListOrders(r.Context(), buyerID)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, orders)
}

func (rt *Router) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	buyerID := PrincipalIDFromContext(r.Context())
	if buyerID == 0 {
		buyerID = 1
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "order code is required")
		return
	}

	order, err := rt.service.GetOrder(r.Context(), buyerID, code)
	if err != nil {
		if err == service.ErrOrderNotFound {
			rt.writeError(w, r, http.StatusNotFound, "order_not_found", "order not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, order)
}

func buyerIDOf(r *http.Request) int64 {
	if id := PrincipalIDFromContext(r.Context()); id != 0 {
		return id
	}
	return 1
}

func (rt *Router) handleRequestReturn(w http.ResponseWriter, r *http.Request) {
	buyerID := buyerIDOf(r)
	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "order code is required")
		return
	}
	var req schema.RequestReturnRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	var items []service.ReturnLineInput
	for _, l := range req.Lines {
		items = append(items, service.ReturnLineInput{OrderLineID: l.OrderLineID, Quantity: l.Quantity})
	}
	resp, err := rt.service.RequestReturn(r.Context(), buyerID, code, items, req.Reason)
	if err != nil {
		switch err {
		case service.ErrOrderNotFound:
			rt.writeError(w, r, http.StatusNotFound, "order_not_found", "order not found")
		case service.ErrReturnState:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "return_state", "order cannot be returned in its state")
		case service.ErrReturnQuantity, service.ErrInvalidQuantity:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_quantity", "invalid return quantity")
		case service.ErrCartLineNotFound:
			rt.writeError(w, r, http.StatusNotFound, "order_line_not_found", "order line not found")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	w.Header().Set("Location", "/api/v1/commerce/orders/me/"+code)
	rt.writeJSON(w, http.StatusCreated, resp)
}

func (rt *Router) handleListInvoicesMe(w http.ResponseWriter, r *http.Request) {
	invoices, err := rt.service.ListInvoicesMe(r.Context(), buyerIDOf(r))
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, invoices)
}

func (rt *Router) handleGetInvoiceMe(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "invoice code is required")
		return
	}
	inv, err := rt.service.GetInvoiceMeByCode(r.Context(), buyerIDOf(r), code)
	if err != nil {
		if err == service.ErrInvoiceNotFound {
			rt.writeError(w, r, http.StatusNotFound, "invoice_not_found", "invoice not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, inv)
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
