package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/atlas-platform/backend/internal/modules/erp/repository"
	"github.com/atlas-platform/backend/internal/modules/erp/schema"
	"github.com/atlas-platform/backend/internal/modules/erp/service"
)

// Router handles ERP HTTP endpoints.
type Router struct {
	r   chi.Router
	svc *service.ERPService
}

// New creates a new ERP router.
func New(svc *service.ERPService) *Router {
	return &Router{
		r:   chi.NewRouter(),
		svc: svc,
	}
}

// ChiRouter returns the underlying chi router.
func (rt *Router) ChiRouter() chi.Router {
	return rt.r
}

// RegisterRoutes registers all ERP routes with middleware.
func (rt *Router) RegisterRoutes(authMiddleware, adminMiddleware func(http.Handler) http.Handler) {
	// Admin routes (auth + admin required)
	rt.r.Route("/admin/erp", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(adminMiddleware)

		r.Post("/sync/catalog", rt.triggerCatalogSync)
		r.Post("/sync/stock", rt.triggerStockSync)
		r.Get("/sync/{job_id}", rt.getSyncJob)
		r.Post("/orders/{order_id}/dispatch", rt.dispatchOrder)
	})
}

// WebhookRouter returns a separate router for webhook endpoints (mounted at root).
func (rt *Router) WebhookRouter() chi.Router {
	r := chi.NewRouter()
	r.Post("/webhooks/erp/{topic}", rt.handleWebhook)
	return r
}

// --- Webhook handler ---

func (rt *Router) handleWebhook(w http.ResponseWriter, r *http.Request) {
	topic := chi.URLParam(r, "topic")
	if topic == "" {
		writeError(w, r, http.StatusBadRequest, "missing_topic", "topic is required")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_body", "failed to read request body")
		return
	}

	timestamp := r.Header.Get("X-Webhook-Timestamp")
	signature := r.Header.Get("X-Webhook-Signature")

	if timestamp == "" || signature == "" {
		writeError(w, r, http.StatusUnauthorized, "missing_headers", "timestamp and signature required")
		return
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "invalid_timestamp", "timestamp must be unix epoch")
		return
	}

	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "default"
	}

	event, err := rt.svc.VerifyAndProcessWebhook(r.Context(), provider, topic, ts, body, signature)
	if err != nil {
		if errors.Is(err, service.ErrWebhookUnauthorized) || errors.Is(err, service.ErrWebhookStale) {
			writeError(w, r, http.StatusUnauthorized, "webhook_unauthorized", "signature verification failed")
			return
		}
		if errors.Is(err, service.ErrWebhookInvalid) {
			writeError(w, r, http.StatusBadRequest, "webhook_invalid", "invalid webhook payload")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "webhook_error", "failed to process webhook")
		return
	}

	writeJSON(w, http.StatusOK, toWebhookResponse(event))
}

// --- Admin handlers ---

func (rt *Router) triggerCatalogSync(w http.ResponseWriter, r *http.Request) {
	var req schema.TriggerSyncRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_json", "failed to parse request body")
			return
		}
	}

	job, err := rt.svc.TriggerCatalogSync(r.Context(), nil, req.Force)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "sync_failed", "failed to trigger catalog sync")
		return
	}

	resp := toSyncJobDetailResponse(job)
	w.Header().Set("Location", fmt.Sprintf("/api/v1/admin/erp/sync/%s", job.Code))
	writeJSON(w, http.StatusAccepted, resp)
}

func (rt *Router) triggerStockSync(w http.ResponseWriter, r *http.Request) {
	var req schema.TriggerSyncRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_json", "failed to parse request body")
			return
		}
	}

	job, err := rt.svc.TriggerStockSync(r.Context(), nil, req.Force)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "sync_failed", "failed to trigger stock sync")
		return
	}

	resp := toSyncJobDetailResponse(job)
	w.Header().Set("Location", fmt.Sprintf("/api/v1/admin/erp/sync/%s", job.Code))
	writeJSON(w, http.StatusAccepted, resp)
}

func (rt *Router) getSyncJob(w http.ResponseWriter, r *http.Request) {
	jobParam := chi.URLParam(r, "job_id")

	// Try parsing as ID first, then fall back to code
	var job *repository.SyncJob
	var drifts []repository.SyncDrift
	var err error

	if id, parseErr := strconv.ParseInt(jobParam, 10, 64); parseErr == nil {
		job, drifts, err = rt.svc.GetSyncJob(r.Context(), id)
	} else {
		job, drifts, err = rt.svc.GetSyncJobByCode(r.Context(), jobParam)
	}

	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "sync job not found")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "fetch_failed", "failed to fetch sync job")
		return
	}

	writeJSON(w, http.StatusOK, toSyncJobResponse(job, drifts))
}

func (rt *Router) dispatchOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(chi.URLParam(r, "order_id"), 10, 64)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_order_id", "order_id must be an integer")
		return
	}

	var req schema.DispatchOrderRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_json", "failed to parse request body")
			return
		}
	}

	dispatch, err := rt.svc.DispatchOrder(r.Context(), orderID, nil, req.Force)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			writeError(w, r, http.StatusNotFound, "order_not_found", "order not found")
			return
		}
		if errors.Is(err, service.ErrOrderNotDispatchable) {
			writeError(w, r, http.StatusConflict, "order_not_dispatchable", "order cannot be dispatched in current state")
			return
		}
		if errors.Is(err, service.ErrAlreadyDispatched) {
			writeError(w, r, http.StatusConflict, "already_dispatched", "order already dispatched")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "dispatch_failed", "failed to dispatch order")
		return
	}

	writeJSON(w, http.StatusOK, toOrderDispatchResponse(dispatch))
}

// --- Response mappers ---

func toWebhookResponse(e *repository.WebhookEvent) schema.WebhookEventResponse {
	return schema.WebhookEventResponse{
		ID:              e.ID,
		Provider:        e.Provider,
		ProviderEventID: e.ProviderEventID,
		Topic:           e.Topic,
		Status:          e.Status,
		ReceivedAt:      e.ReceivedAt,
		AppliedAt:       e.AppliedAt,
	}
}

func toSyncJobDetailResponse(j *repository.SyncJob) schema.SyncJobDetailResponse {
	return schema.SyncJobDetailResponse{
		Code:         j.Code,
		JobType:      j.JobType,
		Status:       j.Status,
		TotalItems:   j.TotalRecords,
		MatchedItems: j.MatchedRecords,
		DriftCount:   j.DriftCount,
		ErrorMessage: j.ErrorMessage,
	}
}

func toSyncJobResponse(j *repository.SyncJob, drifts []repository.SyncDrift) schema.SyncJobResponse {
	resp := schema.SyncJobResponse{
		ID:             j.ID,
		Code:           j.Code,
		JobType:        j.JobType,
		Status:         j.Status,
		TriggeredBy:    j.TriggeredBy,
		StartedAt:      j.StartedAt,
		CompletedAt:    j.CompletedAt,
		TotalRecords:   j.TotalRecords,
		MatchedRecords: j.MatchedRecords,
		CreatedRecords: j.CreatedRecords,
		UpdatedRecords: j.UpdatedRecords,
		DriftCount:     j.DriftCount,
		ErrorMessage:   j.ErrorMessage,
		CreatedAt:      j.CreatedAt,
		UpdatedAt:      j.UpdatedAt,
	}
	if drifts != nil {
		resp.Drifts = make([]schema.DriftResponse, len(drifts))
		for i, d := range drifts {
			resp.Drifts[i] = toDriftResponse(&d)
		}
	}
	return resp
}

func toDriftResponse(d *repository.SyncDrift) schema.DriftResponse {
	return schema.DriftResponse{
		ID:            d.ID,
		SyncJobID:     d.SyncJobID,
		EntityType:    d.EntityType,
		EntityID:      d.EntityID,
		EntityCode:    d.EntityCode,
		DriftType:     d.DriftType,
		LocalValue:    d.LocalValue,
		RemoteValue:   d.RemoteValue,
		FieldsChanged: d.FieldsChanged,
		DetectedAt:    d.DetectedAt,
		ResolvedAt:    d.ResolvedAt,
		Resolution:    d.Resolution,
	}
}

func toOrderDispatchResponse(d *repository.OrderDispatch) schema.OrderDispatchResponse {
	return schema.OrderDispatchResponse{
		ID:            d.ID,
		Code:          d.Code,
		OrderID:       d.OrderID,
		Status:        d.Status,
		AttemptCount:  d.AttemptCount,
		LastAttemptAt: d.LastAttemptAt,
		LastError:     d.LastError,
		NextRetryAt:   d.NextRetryAt,
		DispatchedAt:  d.DispatchedAt,
		TriggeredBy:   d.TriggeredBy,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	requestID := r.Header.Get("X-Request-ID")
	resp := map[string]string{
		"detail":     detail,
		"code":       code,
		"request_id": requestID,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func parsePage(r *http.Request) (page, limit, offset int) {
	page = 1
	limit = 20

	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil && l > 0 {
		limit = l
	}
	if limit > 200 {
		limit = 200
	}

	offset = (page - 1) * limit
	return page, limit, offset
}

var _ = time.Now
