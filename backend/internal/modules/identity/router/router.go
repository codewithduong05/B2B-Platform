package router

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/atlas-platform/backend/internal/modules/identity/schema"
	"github.com/atlas-platform/backend/internal/modules/identity/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type principalKey string

const principalCtxKey principalKey = "identity.principal_id"

func WithPrincipalID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, principalCtxKey, id)
}

func PrincipalIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(principalCtxKey).(int64)
	return id
}

type Router struct {
	router  chi.Router
	service *service.AuthService
	buyerSvc *service.BuyerService
	adminSvc *service.AdminService
}

func New(authSvc *service.AuthService, buyerSvc *service.BuyerService, adminSvc *service.AdminService) *Router {
	return &Router{
		router:  chi.NewRouter(),
		service: authSvc,
		buyerSvc: buyerSvc,
		adminSvc: adminSvc,
	}
}

func (rt *Router) ChiRouter() chi.Router {
	return rt.router
}

func (rt *Router) RegisterRoutes(authMiddleware, adminMiddleware func(http.Handler) http.Handler) {
	// Public auth endpoints
	rt.router.Route("/auth", func(r chi.Router) {
		r.Post("/register", rt.handleRegister)
		r.Post("/register/verify", rt.handleRegisterVerify)
		r.Post("/login", rt.handleLogin)
		r.Post("/refresh", rt.handleRefresh)
		r.Post("/logout", rt.handleLogout)
		r.Post("/password/forgot", rt.handleForgotPassword)
		r.Post("/password/reset", rt.handleResetPassword)
	})

	// Buyer endpoints (authenticated)
	rt.router.Route("/buyer/me", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/", rt.handleGetBuyerProfile)
		r.Patch("/", rt.handleUpdateBuyerProfile)

		r.Get("/addresses", rt.handleListAddresses)
		r.Post("/addresses", rt.handleCreateAddress)
		r.Patch("/addresses/{code}", rt.handleUpdateAddress)
		r.Delete("/addresses/{code}", rt.handleDeleteAddress)

		r.Get("/verification", rt.handleGetVerification)
		r.Post("/verification", rt.handleSubmitVerification)
	})
}

func (rt *Router) RegisterAdminRoutes(adminMiddleware func(http.Handler) http.Handler) {
	rt.router.Route("/admin", func(r chi.Router) {
		if adminMiddleware != nil {
			r.Use(adminMiddleware)
		}

		// User management
		r.Get("/users", rt.handleAdminListUsers)
		r.Get("/users/{id}", rt.handleAdminGetUser)
		r.Patch("/users/{id}", rt.handleAdminUpdateUser)
		r.Post("/users/{id}/suspend", rt.handleAdminSuspendUser)

		// Verification review
		r.Get("/verifications", rt.handleAdminListVerifications)
		r.Post("/verifications/{id}/decide", rt.handleAdminDecideVerification)

		// Roles and permissions
		r.Get("/roles", rt.handleAdminListRoles)
		r.Put("/roles/{role}/permissions", rt.handleAdminSetRolePermissions)
	})
}

// Public auth handlers
func (rt *Router) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req schema.RegisterRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	resp, err := rt.service.Register(r.Context(), req)
	if err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	rt.writeJSON(w, http.StatusCreated, resp)
}

func (rt *Router) handleRegisterVerify(w http.ResponseWriter, r *http.Request) {
	var req schema.RegisterVerifyRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	resp, err := rt.service.VerifyRegistration(r.Context(), req)
	if err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req schema.LoginRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	resp, err := rt.service.Login(r.Context(), req)
	if err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req schema.RefreshRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	resp, err := rt.service.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	rt.writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req schema.LogoutRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if err := rt.service.Logout(r.Context(), req.RefreshToken); err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (rt *Router) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req schema.ForgotPasswordRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if err := rt.service.ForgotPassword(r.Context(), req); err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (rt *Router) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req schema.ResetPasswordRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if err := rt.service.ResetPassword(r.Context(), req); err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Buyer handlers
func (rt *Router) handleGetBuyerProfile(w http.ResponseWriter, r *http.Request) {
	userID := PrincipalIDFromContext(r.Context())
	if userID == 0 {
		rt.writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	profile, err := rt.buyerSvc.GetProfile(r.Context(), userID)
	if err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	rt.writeJSON(w, http.StatusOK, profile)
}

func (rt *Router) handleUpdateBuyerProfile(w http.ResponseWriter, r *http.Request) {
	userID := PrincipalIDFromContext(r.Context())
	if userID == 0 {
		rt.writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	var req schema.UpdateBuyerProfileRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	profile, err := rt.buyerSvc.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	rt.writeJSON(w, http.StatusOK, profile)
}

func (rt *Router) handleListAddresses(w http.ResponseWriter, r *http.Request) {
	userID := PrincipalIDFromContext(r.Context())
	if userID == 0 {
		rt.writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	addresses, err := rt.buyerSvc.ListAddresses(r.Context(), userID)
	if err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	rt.writeJSON(w, http.StatusOK, addresses)
}

func (rt *Router) handleCreateAddress(w http.ResponseWriter, r *http.Request) {
	userID := PrincipalIDFromContext(r.Context())
	if userID == 0 {
		rt.writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	var req schema.CreateAddressRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	address, err := rt.buyerSvc.CreateAddress(r.Context(), userID, req)
	if err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	rt.writeJSON(w, http.StatusCreated, address)
}

func (rt *Router) handleUpdateAddress(w http.ResponseWriter, r *http.Request) {
	userID := PrincipalIDFromContext(r.Context())
	if userID == 0 {
		rt.writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "address code is required")
		return
	}

	var req schema.UpdateAddressRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	address, err := rt.buyerSvc.UpdateAddress(r.Context(), userID, code, req)
	if err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	rt.writeJSON(w, http.StatusOK, address)
}

func (rt *Router) handleDeleteAddress(w http.ResponseWriter, r *http.Request) {
	userID := PrincipalIDFromContext(r.Context())
	if userID == 0 {
		rt.writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "address code is required")
		return
	}

	if err := rt.buyerSvc.DeleteAddress(r.Context(), userID, code); err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (rt *Router) handleGetVerification(w http.ResponseWriter, r *http.Request) {
	userID := PrincipalIDFromContext(r.Context())
	if userID == 0 {
		rt.writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	// Get buyer profile first
	_, err := rt.buyerSvc.GetProfile(r.Context(), userID)
	if err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	// TODO: Get verification from verification service
	rt.writeError(w, r, http.StatusNotImplemented, "not_implemented", "verification endpoint not fully implemented")
}

func (rt *Router) handleSubmitVerification(w http.ResponseWriter, r *http.Request) {
	userID := PrincipalIDFromContext(r.Context())
	if userID == 0 {
		rt.writeError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	var req schema.SubmitVerificationRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	// TODO: Submit verification via verification service
	rt.writeError(w, r, http.StatusNotImplemented, "not_implemented", "verification endpoint not fully implemented")
}

// Admin handlers
func (rt *Router) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	page := int32(1)
	pageSize := int32(20)
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = int32(p)
	}
	if ps, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil && ps > 0 && ps <= 200 {
		pageSize = int32(ps)
	}

	users, total, err := rt.adminSvc.ListUsers(r.Context(), page, pageSize)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": users, "page": page, "page_size": pageSize,
		"total": total, "has_next": int64(page)*int64(pageSize) < total,
	})
}

func (rt *Router) handleAdminGetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "user id is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_id", "invalid user id")
		return
	}

	user, err := rt.adminSvc.GetUserByID(r.Context(), id)
	if err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	rt.writeJSON(w, http.StatusOK, user)
}

func (rt *Router) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "user id is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_id", "invalid user id")
		return
	}

	var req schema.AdminUserUpdateRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	user, err := rt.adminSvc.UpdateUser(r.Context(), id, req)
	if err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	rt.writeJSON(w, http.StatusOK, user)
}

func (rt *Router) handleAdminSuspendUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "user id is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_id", "invalid user id")
		return
	}

	var req schema.AdminSuspendRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if err := rt.adminSvc.SuspendUser(r.Context(), id, req); err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (rt *Router) handleAdminListVerifications(w http.ResponseWriter, r *http.Request) {
	page := int32(1)
	pageSize := int32(20)
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = int32(p)
	}
	if ps, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil && ps > 0 && ps <= 200 {
		pageSize = int32(ps)
	}

	var status *string
	if s := r.URL.Query().Get("status"); s != "" {
		status = &s
	}

	apps, total, err := rt.adminSvc.ListVerifications(r.Context(), status, page, pageSize)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": apps, "page": page, "page_size": pageSize,
		"total": total, "has_next": int64(page)*int64(pageSize) < total,
	})
}

func (rt *Router) handleAdminDecideVerification(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "verification id is required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_id", "invalid verification id")
		return
	}

	var req schema.AdminVerificationDecisionRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	// Get admin user ID from context (assuming auth middleware sets it)
	decidedBy := int64(0)
	if principalID := PrincipalIDFromContext(r.Context()); principalID > 0 {
		decidedBy = principalID
	}

	app, err := rt.adminSvc.DecideVerification(r.Context(), id, req, int64(decidedBy))
	if err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	rt.writeJSON(w, http.StatusOK, app)
}

func (rt *Router) handleAdminListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := rt.adminSvc.ListRoles(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	rt.writeJSON(w, http.StatusOK, roles)
}

func (rt *Router) handleAdminSetRolePermissions(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, "role")
	if role == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "role code is required")
		return
	}

	var req schema.AdminSetRolePermissionsRequest
	if err := rt.decode(r, &req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	if err := rt.adminSvc.SetRolePermissions(r.Context(), role, req); err != nil {
		rt.handleAuthError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (rt *Router) decode(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func (rt *Router) handleAuthError(w http.ResponseWriter, r *http.Request, err error) {
	switch err {
	case service.ErrUserAlreadyExists:
		rt.writeError(w, r, http.StatusConflict, "user_exists", err.Error())
	case service.ErrInvalidCredentials:
		rt.writeError(w, r, http.StatusUnauthorized, "invalid_credentials", err.Error())
	case service.ErrUserNotFound:
		rt.writeError(w, r, http.StatusNotFound, "not_found", err.Error())
	case service.ErrUserInactive:
		rt.writeError(w, r, http.StatusForbidden, "inactive", err.Error())
	case service.ErrAccountLocked:
		rt.writeError(w, r, http.StatusTooManyRequests, "locked", err.Error())
	case service.ErrInvalidOTP:
		rt.writeError(w, r, http.StatusBadRequest, "invalid_otp", err.Error())
	case service.ErrOTPMaxAttempts:
		rt.writeError(w, r, http.StatusBadRequest, "otp_max_attempts", err.Error())
	case service.ErrTokenExpired:
		rt.writeError(w, r, http.StatusUnauthorized, "token_expired", err.Error())
	case service.ErrTokenRevoked:
		rt.writeError(w, r, http.StatusUnauthorized, "token_revoked", err.Error())
	case service.ErrInvalidToken:
		rt.writeError(w, r, http.StatusUnauthorized, "invalid_token", err.Error())
	case service.ErrVerificationNotFound:
		rt.writeError(w, r, http.StatusNotFound, "not_found", err.Error())
	case service.ErrVerificationPending:
		rt.writeError(w, r, http.StatusConflict, "verification_pending", err.Error())
	case service.ErrAddressNotFound:
		rt.writeError(w, r, http.StatusNotFound, "not_found", err.Error())
	case service.ErrUnauthorized:
		rt.writeError(w, r, http.StatusUnauthorized, "unauthorized", err.Error())
	case service.ErrForbidden:
		rt.writeError(w, r, http.StatusForbidden, "forbidden", err.Error())
	case service.ErrBuyerProfileNotFound:
		rt.writeError(w, r, http.StatusNotFound, "not_found", err.Error())
	case service.ErrRoleNotFound:
		rt.writeError(w, r, http.StatusNotFound, "not_found", err.Error())
	case service.ErrPermissionNotFound:
		rt.writeError(w, r, http.StatusNotFound, "not_found", err.Error())
	case service.ErrInvalidResetToken:
		rt.writeError(w, r, http.StatusBadRequest, "invalid_token", err.Error())
	case service.ErrResetTokenUsed:
		rt.writeError(w, r, http.StatusBadRequest, "token_used", err.Error())
	case service.ErrInvalidInput:
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
	}
}

func (rt *Router) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		rt.writeError(w, nil, http.StatusInternalServerError, "internal", "failed to encode response")
	}
}

func (rt *Router) writeError(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	body := map[string]string{"detail": detail, "code": code}
	if r != nil {
		if requestID := middleware.GetReqID(r.Context()); requestID != "" {
			body["request_id"] = requestID
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}