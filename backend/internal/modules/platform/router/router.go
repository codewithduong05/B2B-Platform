package router

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/atlas-platform/backend/internal/modules/platform/repository"
	"github.com/atlas-platform/backend/internal/modules/platform/schema"
	"github.com/atlas-platform/backend/internal/modules/platform/service"
)

type principalKey string

const principalCtxKey principalKey = "platform.principal_id"

func WithPrincipalID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, principalCtxKey, id)
}

func PrincipalIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(principalCtxKey).(int64)
	return id
}

type Router struct {
	router  chi.Router
	service *service.PlatformService
}

func New(svc *service.PlatformService) *Router {
	return &Router{router: chi.NewRouter(), service: svc}
}

func (rt *Router) ChiRouter() chi.Router {
	return rt.router
}

func (rt *Router) RegisterRoutes(authMiddleware, adminMiddleware func(http.Handler) http.Handler) {
	rt.router.Route("/platform", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Get("/notifications", rt.handleBuyerNotifications)
		r.Post("/notifications/{code}/read", rt.handleMarkNotificationRead)
		r.Post("/media", rt.handleCreateMedia)
		r.Get("/reference/regions", rt.handleListRegions)
		r.Get("/feature-flags", rt.handleListFeatureFlags)
	})

	rt.router.Route("/admin/platform", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		if adminMiddleware != nil {
			r.Use(adminMiddleware)
		}
		r.Post("/notification-templates", rt.handleCreateTemplate)
		r.Get("/notification-templates", rt.handleListTemplates)
		r.Patch("/notification-templates/{id}", rt.handleUpdateTemplate)
		r.Post("/notifications/send", rt.handleSendNotification)
		r.Get("/notifications", rt.handleAdminListNotifications)
		r.Get("/notifications/{code}", rt.handleAdminGetNotification)
		r.Get("/notifications/{code}/events", rt.handleAdminGetNotificationEvents)
		r.Get("/audit-log", rt.handleListAuditLog)
		r.Get("/integration-traffic", rt.handleListIntegrationTraffic)
	})

	rt.router.Route("/admin", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		if adminMiddleware != nil {
			r.Use(adminMiddleware)
		}
		r.Get("/feature-flags", rt.handleAdminListFeatureFlags)
		r.Put("/feature-flags/{key}", rt.handleUpsertFeatureFlag)
	})
}

func (rt *Router) handleBuyerNotifications(w http.ResponseWriter, r *http.Request) {
	buyerID := PrincipalIDFromContext(r.Context())
	if buyerID == 0 {
		rt.writeError(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	page, limit, offset := parsePage(r)
	status := r.URL.Query().Get("status")

	notifications, total, err := rt.service.ListBuyerNotifications(r.Context(), buyerID, status, limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	items := make([]schema.BuyerNotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		items = append(items, schema.BuyerNotificationResponse{
			ID:        n.ID,
			Code:      n.Code,
			Subject:   n.Subject,
			Body:      n.Body,
			Status:    n.Status,
			CreatedAt: n.CreatedAt,
		})
	}

	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     items,
		"page":      page,
		"page_size": limit,
		"total":     total,
		"has_next":  page*limit < total,
	})
}

func (rt *Router) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "notification code is required")
		return
	}

	buyerID := PrincipalIDFromContext(r.Context())
	if buyerID == 0 {
		rt.writeError(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	notification, err := rt.service.GetNotificationByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			rt.writeError(w, r, http.StatusNotFound, "not_found", "notification not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	if notification.RecipientType != "buyer" || notification.RecipientID != buyerID {
		rt.writeError(w, r, http.StatusNotFound, "not_found", "notification not found")
		return
	}

	rt.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (rt *Router) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateTemplateRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	template, err := rt.service.CreateTemplate(r.Context(), req.Code, req.Name, req.Channel, req.Subject, req.Body, req.Variables, nil)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid template data")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	resp := toTemplateResponse(template)
	w.Header().Set("Location", "/api/v1/admin/platform/notification-templates")
	rt.writeJSON(w, http.StatusCreated, resp)
}

func (rt *Router) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	q := r.URL.Query()
	channel := q.Get("channel")
	status := q.Get("status")

	templates, total, err := rt.service.ListTemplates(r.Context(), channel, status, limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	items := make([]schema.NotificationTemplateResponse, 0, len(templates))
	for _, t := range templates {
		items = append(items, toTemplateResponse(&t))
	}

	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     items,
		"page":      page,
		"page_size": limit,
		"total":     total,
		"has_next":  page*limit < total,
	})
}

func (rt *Router) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_id", "invalid template id")
		return
	}

	var req schema.UpdateTemplateRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	template, err := rt.service.UpdateTemplate(r.Context(), id, req.Name, req.Subject, req.Body, req.Variables, req.Status)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			rt.writeError(w, r, http.StatusNotFound, "not_found", "template not found")
			return
		}
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid template data")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, toTemplateResponse(template))
}

func (rt *Router) handleSendNotification(w http.ResponseWriter, r *http.Request) {
	var req schema.SendNotificationRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	notification, err := rt.service.SendNotification(r.Context(), req.TemplateCode, req.RecipientType, req.RecipientID, req.Variables)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			rt.writeError(w, r, http.StatusNotFound, "not_found", "template not found")
			return
		}
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid request data")
			return
		}
		if errors.Is(err, service.ErrTemplateInactive) {
			rt.writeError(w, r, http.StatusConflict, "template_inactive", "template is not active")
			return
		}
		if errors.Is(err, service.ErrRecipientNotFound) {
			rt.writeError(w, r, http.StatusNotFound, "recipient_not_found", "recipient not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusAccepted, toNotificationResponse(notification))
}

func (rt *Router) handleAdminListNotifications(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	q := r.URL.Query()
	channel := q.Get("channel")
	status := q.Get("status")

	notifications, total, err := rt.service.ListAdminNotifications(r.Context(), channel, status, limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	items := make([]schema.NotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		items = append(items, toNotificationResponse(&n))
	}

	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     items,
		"page":      page,
		"page_size": limit,
		"total":     total,
		"has_next":  page*limit < total,
	})
}

func (rt *Router) handleAdminGetNotification(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "notification code is required")
		return
	}

	notification, err := rt.service.GetNotificationByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			rt.writeError(w, r, http.StatusNotFound, "not_found", "notification not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, toNotificationResponse(notification))
}

func (rt *Router) handleAdminGetNotificationEvents(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "notification code is required")
		return
	}

	notification, err := rt.service.GetNotificationByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			rt.writeError(w, r, http.StatusNotFound, "not_found", "notification not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	events, err := rt.service.GetNotificationEvents(r.Context(), notification.ID)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	items := make([]schema.NotificationEventResponse, 0, len(events))
	for _, e := range events {
		items = append(items, schema.NotificationEventResponse{
			ID:              e.ID,
			NotificationID:  e.NotificationID,
			EventType:       e.EventType,
			ProviderEventID: e.ProviderEventID,
			Metadata:        e.Metadata,
			CreatedAt:       e.CreatedAt,
		})
	}

	rt.writeJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (rt *Router) handleCreateMedia(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	createdBy := PrincipalIDFromContext(r.Context())
	media, err := rt.service.CreateMediaUpload(r.Context(), req.Filename, req.ContentType, &createdBy)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_input", "filename and content_type are required")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusCreated, schema.MediaResponse{
		Code:        media.Code,
		Filename:    media.Filename,
		ContentType: media.ContentType,
		UploadURL:   media.UploadURL,
		DownloadURL: media.DownloadURL,
		Status:      media.Status,
		ExpiresAt:   media.ExpiresAt,
		CreatedAt:   media.CreatedAt,
	})
}

func (rt *Router) handleListRegions(w http.ResponseWriter, r *http.Request) {
	regions, err := rt.service.ListActiveRegions(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	items := make([]schema.RegionResponse, 0, len(regions))
	for _, reg := range regions {
		items = append(items, schema.RegionResponse{
			Code:      reg.Code,
			Name:      reg.Name,
			Country:   reg.Country,
			IsActive:  reg.IsActive,
			CreatedAt: reg.CreatedAt,
		})
	}

	rt.writeJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (rt *Router) handleListFeatureFlags(w http.ResponseWriter, r *http.Request) {
	var enabled *bool
	if v := r.URL.Query().Get("enabled"); v != "" {
		b := v == "true"
		enabled = &b
	}

	page, limit, offset := parsePage(r)
	flags, total, err := rt.service.ListFeatureFlags(r.Context(), enabled, limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	items := make([]schema.FeatureFlagResponse, 0, len(flags))
	for _, f := range flags {
		items = append(items, schema.FeatureFlagResponse{
			Key:         f.Key,
			Name:        f.Name,
			Description: f.Description,
			Enabled:     f.Enabled,
			UpdatedBy:   f.UpdatedBy,
			CreatedAt:   f.CreatedAt,
			UpdatedAt:   f.UpdatedAt,
		})
	}

	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     items,
		"page":      page,
		"page_size": limit,
		"total":     total,
	})
}

func (rt *Router) handleAdminListFeatureFlags(w http.ResponseWriter, r *http.Request) {
	var enabled *bool
	if v := r.URL.Query().Get("enabled"); v != "" {
		b := v == "true"
		enabled = &b
	}

	page, limit, offset := parsePage(r)
	flags, total, err := rt.service.ListFeatureFlags(r.Context(), enabled, limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	items := make([]schema.FeatureFlagResponse, 0, len(flags))
	for _, f := range flags {
		items = append(items, schema.FeatureFlagResponse{
			Key:         f.Key,
			Name:        f.Name,
			Description: f.Description,
			Enabled:     f.Enabled,
			UpdatedBy:   f.UpdatedBy,
			CreatedAt:   f.CreatedAt,
			UpdatedAt:   f.UpdatedAt,
		})
	}

	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     items,
		"page":      page,
		"page_size": limit,
		"total":     total,
	})
}

func (rt *Router) handleUpsertFeatureFlag(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "feature flag key is required")
		return
	}

	var req schema.UpdateFeatureFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	updatedBy := PrincipalIDFromContext(r.Context())
	flag, err := rt.service.UpsertFeatureFlag(r.Context(), key, req.Name, req.Description, req.Enabled, &updatedBy)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_input", "key and name are required")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, schema.FeatureFlagResponse{
		Key:         flag.Key,
		Name:        flag.Name,
		Description: flag.Description,
		Enabled:     flag.Enabled,
		UpdatedBy:   flag.UpdatedBy,
		CreatedAt:   flag.CreatedAt,
		UpdatedAt:   flag.UpdatedAt,
	})
}

func (rt *Router) handleListAuditLog(w http.ResponseWriter, r *http.Request) {
	actorType := r.URL.Query().Get("actor_type")
	action := r.URL.Query().Get("action")
	resourceType := r.URL.Query().Get("resource_type")

	var actorID, resourceID *int64
	if v := r.URL.Query().Get("actor_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			actorID = &id
		}
	}
	if v := r.URL.Query().Get("resource_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			resourceID = &id
		}
	}

	page, limit, offset := parsePage(r)
	logs, total, err := rt.service.ListAuditLog(r.Context(), actorType, action, resourceType, actorID, resourceID, limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	items := make([]schema.AuditLogResponse, 0, len(logs))
	for _, l := range logs {
		items = append(items, schema.AuditLogResponse{
			ID:           l.ID,
			ActorID:      l.ActorID,
			ActorType:    l.ActorType,
			Action:       l.Action,
			ResourceType: l.ResourceType,
			ResourceID:   l.ResourceID,
			ResourceCode: l.ResourceCode,
			Metadata:     l.Metadata,
			IPAddress:    l.IPAddress,
			UserAgent:    l.UserAgent,
			CreatedAt:    l.CreatedAt,
		})
	}

	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     items,
		"page":      page,
		"page_size": limit,
		"total":     total,
	})
}

func (rt *Router) handleListIntegrationTraffic(w http.ResponseWriter, r *http.Request) {
	direction := r.URL.Query().Get("direction")
	provider := r.URL.Query().Get("provider")

	page, limit, offset := parsePage(r)
	traffic, total, err := rt.service.ListIntegrationTraffic(r.Context(), direction, provider, limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	items := make([]schema.IntegrationTrafficResponse, 0, len(traffic))
	for _, t := range traffic {
		items = append(items, schema.IntegrationTrafficResponse{
			ID:              t.ID,
			Direction:       t.Direction,
			Provider:        t.Provider,
			Endpoint:        t.Endpoint,
			Method:          t.Method,
			StatusCode:      t.StatusCode,
			PayloadHash:     t.PayloadHash,
			RequestHeaders:  t.RequestHeaders,
			ResponseHeaders: t.ResponseHeaders,
			ErrorMessage:    t.ErrorMessage,
			DurationMS:      t.DurationMS,
			CreatedAt:       t.CreatedAt,
		})
	}

	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     items,
		"page":      page,
		"page_size": limit,
		"total":     total,
	})
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

func toTemplateResponse(t *repository.NotificationTemplate) schema.NotificationTemplateResponse {
	return schema.NotificationTemplateResponse{
		ID:        t.ID,
		Code:      t.Code,
		Name:      t.Name,
		Channel:   t.Channel,
		Subject:   t.Subject,
		Body:      t.Body,
		Variables: t.Variables,
		Status:    t.Status,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func toNotificationResponse(n *repository.Notification) schema.NotificationResponse {
	return schema.NotificationResponse{
		ID:                n.ID,
		Code:              n.Code,
		TemplateCode:      n.TemplateCode,
		RecipientType:     n.RecipientType,
		RecipientID:       n.RecipientID,
		Channel:           n.Channel,
		RecipientAddress:  n.RecipientAddress,
		Subject:           n.Subject,
		Body:              n.Body,
		Status:            n.Status,
		ProviderMessageID: n.ProviderMessageID,
		SentAt:            n.SentAt,
		DeliveredAt:       n.DeliveredAt,
		FailedAt:          n.FailedAt,
		ErrorMessage:      n.ErrorMessage,
		RetryCount:        n.RetryCount,
		CreatedAt:         n.CreatedAt,
	}
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
