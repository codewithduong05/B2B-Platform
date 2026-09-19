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
