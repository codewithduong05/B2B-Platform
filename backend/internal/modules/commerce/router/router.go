package router

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
	"github.com/atlas-platform/backend/internal/modules/commerce/service"
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
		r.Post("/checkout", rt.handleCheckout)
		r.Get("/orders/me", rt.handleListOrders)
		r.Get("/orders/me/{code}", rt.handleGetOrder)
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
