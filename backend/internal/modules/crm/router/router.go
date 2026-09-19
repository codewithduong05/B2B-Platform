package router

// CRM HTTP surface. POST /leads and GET /partners/track are public;
// everything else is staff-gated by the injected admin middleware.

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/atlas-platform/backend/internal/modules/crm/schema"
	"github.com/atlas-platform/backend/internal/modules/crm/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type principalKey string

const principalCtxKey principalKey = "crm.principal_id"

func WithPrincipalID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, principalCtxKey, id)
}

func PrincipalIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(principalCtxKey).(int64)
	return id
}

type Router struct {
	router  chi.Router
	service *service.CRMService
}

func New(svc *service.CRMService) *Router {
	return &Router{router: chi.NewRouter(), service: svc}
}

func (rt *Router) ChiRouter() chi.Router {
	return rt.router
}

func (rt *Router) RegisterRoutes(authMiddleware, adminMiddleware func(http.Handler) http.Handler) {
	rt.router.Post("/leads", rt.handleSubmitLead)
	rt.router.Get("/partners/track", rt.handleTrack)

	rt.router.Route("/admin/leads", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		if adminMiddleware != nil {
			r.Use(adminMiddleware)
		}
		r.Get("/", rt.handleAdminListLeads)
		r.Post("/{id}/assign", rt.handleAdminAssignLead)
		r.Patch("/{id}", rt.handleAdminUpdateLead)
	})

	rt.router.Route("/admin/referrals", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		if adminMiddleware != nil {
			r.Use(adminMiddleware)
		}
		r.Get("/", rt.handleAdminListReferrals)
		r.Post("/", rt.handleAdminCreateReferral)
	})

	rt.router.Route("/admin/partners", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		if adminMiddleware != nil {
			r.Use(adminMiddleware)
		}
		r.Get("/", rt.handleAdminListPartners)
		r.Post("/", rt.handleAdminCreatePartner)
	})
}

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

func (rt *Router) handleSubmitLead(w http.ResponseWriter, r *http.Request) {
	var req schema.SubmitLeadRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	resp, err := rt.service.SubmitLead(r.Context(), req)
	if err != nil {
		switch err {
		case service.ErrInvalidLead:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "contact name and business name are required")
		case service.ErrReferralInvalid:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "invalid_partner_ref", "partner reference does not resolve")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	w.Header().Set("Location", "/api/v1/admin/leads")
	rt.writeJSON(w, http.StatusCreated, resp)
}

func (rt *Router) handleTrack(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "ref query parameter is required")
		return
	}
	resp, err := rt.service.TrackPartner(r.Context(), ref)
	if err != nil {
		if err == service.ErrReferralInvalid {
			rt.writeError(w, r, http.StatusNotFound, "partner_not_found", "partner reference does not resolve")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleAdminListLeads(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	leads, total, err := rt.service.ListLeads(r.Context(), r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeEnvelope(w, leads, page, limit, total)
}

func (rt *Router) handleAdminAssignLead(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	var req schema.AssignLeadRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	resp, err := rt.service.AssignLead(r.Context(), id, req.Assignee)
	if err != nil {
		switch err {
		case service.ErrLeadNotFound:
			rt.writeError(w, r, http.StatusNotFound, "lead_not_found", "lead not found")
		case service.ErrLeadState:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "lead_state", "only new leads can be assigned")
		case service.ErrInvalidLead:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "assignee is required")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleAdminUpdateLead(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	var req schema.UpdateLeadRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	resp, err := rt.service.UpdateLead(r.Context(), id, req)
	if err != nil {
		switch err {
		case service.ErrLeadNotFound:
			rt.writeError(w, r, http.StatusNotFound, "lead_not_found", "lead not found")
		case service.ErrLeadState:
			rt.writeError(w, r, http.StatusUnprocessableEntity, "lead_state", "invalid lead status transition")
		case service.ErrInvalidLead:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid update: conversion needs a valid buyer")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleAdminListReferrals(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	rows, total, err := rt.service.ListReferrals(r.Context(), limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeEnvelope(w, rows, page, limit, total)
}

func (rt *Router) handleAdminCreateReferral(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateReferralRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	if req.PartnerID <= 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "partner_id is required")
		return
	}
	resp, err := rt.service.CreateReferral(r.Context(), req)
	if err != nil {
		switch err {
		case service.ErrPartnerNotFound:
			rt.writeError(w, r, http.StatusNotFound, "partner_not_found", "partner not found")
		case service.ErrDuplicate:
			rt.writeError(w, r, http.StatusConflict, "duplicate_code", "referral code already exists")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	w.Header().Set("Location", "/api/v1/admin/referrals")
	rt.writeJSON(w, http.StatusCreated, resp)
}

func (rt *Router) handleAdminListPartners(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	rows, total, err := rt.service.ListPartners(r.Context(), limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeEnvelope(w, rows, page, limit, total)
}

func (rt *Router) handleAdminCreatePartner(w http.ResponseWriter, r *http.Request) {
	var req schema.CreatePartnerRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	resp, err := rt.service.CreatePartner(r.Context(), req)
	if err != nil {
		switch err {
		case service.ErrInvalidPartner:
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "partner name is required")
		case service.ErrDuplicate:
			rt.writeError(w, r, http.StatusConflict, "duplicate_code", "partner code or slug already exists")
		default:
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		}
		return
	}
	w.Header().Set("Location", "/api/v1/admin/partners")
	rt.writeJSON(w, http.StatusCreated, resp)
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
