package router

// Payments HTTP surface. Buyer routes under auth; staff routes grouped
// under the injected admin middleware (a global stub in production wiring;
// tests inject a staff-set check).

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/atlas-platform/backend/internal/modules/payments/schema"
	"github.com/atlas-platform/backend/internal/modules/payments/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type principalKey string

const principalCtxKey principalKey = "payments.principal_id"

func WithPrincipalID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, principalCtxKey, id)
}

func PrincipalIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(principalCtxKey).(int64)
	return id
}

type Router struct {
	router  chi.Router
	service *service.PaymentService
}

func New(svc *service.PaymentService) *Router {
	return &Router{router: chi.NewRouter(), service: svc}
}

func (rt *Router) ChiRouter() chi.Router {
	return rt.router
}

func (rt *Router) RegisterRoutes(authMiddleware, adminMiddleware func(http.Handler) http.Handler) {
	rt.router.Route("/payments", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Get("/methods", rt.handleListMethods)
		r.Post("/intents", rt.handleCreateIntent)
		r.Get("/intents/{code}", rt.handleGetIntent)
		r.Get("/statements", rt.handleBuyerStatement)
	})

	rt.router.Route("/admin/payments", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		if adminMiddleware != nil {
			r.Use(adminMiddleware)
		}
		r.Get("/", rt.handleAdminListIntents)
		r.Post("/intents/{code}/complete", rt.handleMarkComplete)
		r.Post("/intents/{code}/fail", rt.handleMarkFailed)
		r.Post("/intents/{code}/refund", rt.handleRequestRefund)
		r.Post("/refunds/{id}/approve", rt.handleApproveRefund)
		r.Post("/refunds/{id}/reject", rt.handleRejectRefund)
		r.Get("/reconciliation", rt.handleReconciliation)
		r.Get("/statements", rt.handleAdminStatements)
		r.Get("/credit/{buyer_id}", rt.handleGetCredit)
		r.Put("/credit/{buyer_id}", rt.handleSetCredit)
	})
}

// WebhookRouter serves provider callbacks outside /api/v1 (contract:
// /webhooks/payments/{provider}). No auth middleware: verification is by
// HMAC signature. Mount at "/" in main.
func (rt *Router) WebhookRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Post("/webhooks/payments/{provider}", rt.handleWebhook)
	return r
}

func principalID(r *http.Request) int64 {
	if id := PrincipalIDFromContext(r.Context()); id != 0 {
		return id
	}
	return 1
}

func (rt *Router) handleListMethods(w http.ResponseWriter, r *http.Request) {
	methods, err := rt.service.ListMethods(r.Context(), principalID(r))
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if methods == nil {
		methods = []schema.PaymentMethodResponse{}
	}
	rt.writeJSON(w, http.StatusOK, methods)
}

func (rt *Router) handleCreateIntent(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		r.Body.Close()
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	r.Body.Close()

	resp, created, err := rt.service.CreateIntent(r.Context(), principalID(r), req)
	if err != nil {
		switch err {
		case service.ErrInvalidIntent:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "order_code and idem_key are required")
		case service.ErrMethodNotFound:
			rt.writeError(w, r, http.StatusNotFound, "payment_method_not_found", "payment method not found")
		case service.ErrOrderNotFound:
			rt.writeError(w, r, http.StatusNotFound, "order_not_found", "order not found")
		case service.ErrIntentConflict:
			rt.writeError(w, r, http.StatusConflict, "intent_conflict", "idempotency key reused for a different intent")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}

	if !created {
		rt.writeJSON(w, http.StatusOK, resp)
		return
	}
	w.Header().Set("Location", "/api/v1/payments/intents/"+resp.Code)
	rt.writeJSON(w, http.StatusCreated, resp)
}

func (rt *Router) handleGetIntent(w http.ResponseWriter, r *http.Request) {	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "intent code is required")
		return
	}
	resp, err := rt.service.GetIntent(r.Context(), principalID(r), code)
	if err != nil {
		if err == service.ErrIntentNotFound {
			rt.writeError(w, r, http.StatusNotFound, "intent_not_found", "payment intent not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleAdminListIntents(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r)
	intents, err := rt.service.AdminListIntents(r.Context(), r.URL.Query().Get("status"), pageSize, (page-1)*pageSize)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if intents == nil {
		intents = []schema.IntentResponse{}
	}
	total, err := rt.service.AdminCountIntents(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": intents, "page": page, "page_size": pageSize,
		"total": total, "has_next": page*pageSize < total,
	})
}

func (rt *Router) handleMarkComplete(w http.ResponseWriter, r *http.Request) {
	rt.handleMark(w, r, true)
}

func (rt *Router) handleMarkFailed(w http.ResponseWriter, r *http.Request) {
	rt.handleMark(w, r, false)
}

func (rt *Router) handleMark(w http.ResponseWriter, r *http.Request, succeeded bool) {
	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "intent code is required")
		return
	}
	var req schema.MarkIntentRequest
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			r.Body.Close()
			rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}
	}
	r.Body.Close()

	resp, err := rt.service.MarkIntent(r.Context(), principalID(r), code, succeeded, req.Note)
	if err != nil {
		switch err {
		case service.ErrIntentNotFound:
			rt.writeError(w, r, http.StatusNotFound, "intent_not_found", "payment intent not found")
		case service.ErrIntentTerminal:
			rt.writeError(w, r, http.StatusConflict, "intent_terminal", "payment intent is already terminal")
		case service.ErrExceedsBalance:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "payment_exceeds_balance", "payment exceeds outstanding balance")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func parsePagination(r *http.Request) (page, pageSize int) {
	page, pageSize = 1, 20
	if v, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && v >= 1 {
		page = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil && v >= 1 {
		pageSize = v
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
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

func (rt *Router) handleWebhook(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if provider == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "provider is required")
		return
	}
	defer r.Body.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "cannot read request body")
		return
	}
	body := buf.Bytes()

	outcome, err := rt.service.HandleWebhook(r.Context(), provider, body,
		r.Header.Get("X-Webhook-Timestamp"), r.Header.Get("X-Webhook-Signature"))
	if err != nil {
		switch err {
		case service.ErrWebhookUnauthorized, service.ErrWebhookStale:
			rt.writeError(w, r, http.StatusUnauthorized, "webhook_unauthorized", "invalid webhook signature or timestamp")
		case service.ErrWebhookInvalid:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid webhook payload")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	if outcome.Status == "unmatched" || outcome.Status == "terminal_conflict" {
		rt.writeJSON(w, http.StatusAccepted, outcome)
		return
	}
	rt.writeJSON(w, http.StatusOK, outcome)
}

func (rt *Router) handleRequestRefund(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "intent code is required")
		return
	}
	var req schema.RequestRefundRequest
	if r.ContentLength != 0 {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}
	} else {
		defer r.Body.Close()
	}
	resp, err := rt.service.RequestRefund(r.Context(), principalID(r), code, req.AmountMinor, req.Reason)
	if err != nil {
		switch err {
		case service.ErrIntentNotFound:
			rt.writeError(w, r, http.StatusNotFound, "intent_not_found", "payment intent not found")
		case service.ErrRefundState:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "refund_state", "intent cannot be refunded in its current state")
		case service.ErrExceedsBalance:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "refund_exceeds_captured", "refund exceeds captured amount")
		case service.ErrInvalidIntent:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "amount and reason are required")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	w.Header().Set("Location", "/api/v1/admin/payments/refunds")
	rt.writeJSON(w, http.StatusCreated, resp)
}

func (rt *Router) handleApproveRefund(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRefundID(w, r, rt)
	if !ok {
		return
	}
	resp, err := rt.service.ApproveRefund(r.Context(), principalID(r), id)
	if err != nil {
		switch err {
		case service.ErrRefundNotFound:
			rt.writeError(w, r, http.StatusNotFound, "refund_not_found", "refund not found")
		case service.ErrRefundState:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "refund_state", "refund is not actionable")
		case service.ErrRefundSelfApproval:
			rt.writeError(w, r, http.StatusForbidden, "refund_approval_required", "a second approver is required")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleRejectRefund(w http.ResponseWriter, r *http.Request) {
	id, ok := parseRefundID(w, r, rt)
	if !ok {
		return
	}
	resp, err := rt.service.RejectRefund(r.Context(), principalID(r), id)
	if err != nil {
		switch err {
		case service.ErrRefundNotFound:
			rt.writeError(w, r, http.StatusNotFound, "refund_not_found", "refund not found")
		case service.ErrRefundState:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "refund_state", "refund is not actionable")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func parseRefundID(w http.ResponseWriter, r *http.Request, rt *Router) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_id", "invalid numeric id")
		return 0, false
	}
	return id, true
}

func (rt *Router) handleReconciliation(w http.ResponseWriter, r *http.Request) {
	rows, unmatched, err := rt.service.Reconcile(r.Context(), 200)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if rows == nil {
		rows = []schema.ReconResponse{}
	}
	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"orders": rows, "unmatched_webhooks": unmatched,
	})
}

func (rt *Router) handleGetCredit(w http.ResponseWriter, r *http.Request) {
	buyerID, err := strconv.ParseInt(chi.URLParam(r, "buyer_id"), 10, 64)
	if err != nil || buyerID <= 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_id", "invalid buyer id")
		return
	}
	resp, err := rt.service.GetCredit(r.Context(), buyerID)
	if err != nil {
		if err == service.ErrCreditNotFound {
			rt.writeError(w, r, http.StatusNotFound, "credit_not_found", "credit account not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleSetCredit(w http.ResponseWriter, r *http.Request) {
	buyerID, err := strconv.ParseInt(chi.URLParam(r, "buyer_id"), 10, 64)
	if err != nil || buyerID <= 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_id", "invalid buyer id")
		return
	}
	var req schema.SetCreditRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	resp, err := rt.service.SetCredit(r.Context(), buyerID, req.CreditLimitMinor, req.Terms, req.OnHold, req.HoldReason)
	if err != nil {
		if err == service.ErrInvalidIntent {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "limit and terms are required")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleBuyerStatement(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "period YYYY-MM is required")
		return
	}
	stmt, err := rt.service.BuyerStatement(r.Context(), principalID(r), period)
	if err != nil {
		if err == service.ErrInvalidIntent {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "period must be YYYY-MM")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, stmt)
}

func (rt *Router) handleAdminStatements(w http.ResponseWriter, r *http.Request) {
	buyerID, err := strconv.ParseInt(r.URL.Query().Get("buyer_id"), 10, 64)
	if err != nil || buyerID <= 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "buyer_id is required")
		return
	}
	period := r.URL.Query().Get("period")
	if period == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "period YYYY-MM is required")
		return
	}
	stmt, err := rt.service.AdminStatement(r.Context(), buyerID, period)
	if err != nil {
		if err == service.ErrInvalidIntent {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "bad buyer_id or period")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, stmt)
}
