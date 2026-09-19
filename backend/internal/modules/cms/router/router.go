package router

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/atlas-platform/backend/internal/modules/cms/schema"
	"github.com/atlas-platform/backend/internal/modules/cms/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Router struct {
	router  chi.Router
	service *service.CMSService
}

func New(svc *service.CMSService) *Router {
	return &Router{router: chi.NewRouter(), service: svc}
}

func (rt *Router) ChiRouter() chi.Router {
	return rt.router
}

func (rt *Router) RegisterRoutes(authMiddleware, adminMiddleware func(http.Handler) http.Handler) {
	// Public CMS routes
	rt.router.Route("/cms", func(r chi.Router) {
		r.Get("/articles", rt.handleListPublishedArticles)
		r.Get("/articles/{slug}", rt.handleGetArticleBySlug)
		r.Get("/pages/{slug}", rt.handleGetPageBySlug)
		r.Get("/faqs", rt.handleListFaqs)
		r.Get("/menus/{location}", rt.handleListMenus)
	})

	// Admin CMS routes
	rt.router.Route("/admin/cms", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		if adminMiddleware != nil {
			r.Use(adminMiddleware)
		}
		r.Get("/articles", rt.handleAdminListArticles)
		r.Post("/articles", rt.handleAdminCreateArticle)
		r.Post("/articles/{id}/publish", rt.handleAdminPublishArticle)
		r.Get("/pages", rt.handleAdminListPages)
		r.Post("/pages", rt.handleAdminUpsertPage)
		r.Get("/banners", rt.handleAdminListBanners)
		r.Post("/banners", rt.handleAdminCreateBanner)
		r.Post("/menus", rt.handleAdminUpsertMenu)
		r.Get("/settings", rt.handleAdminListSettings)
		r.Put("/settings", rt.handleAdminUpdateSetting)
	})
}

func parsePage(r *http.Request) (page, limit, offset int32) {
	p, l := 1, 20
	if v, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && v >= 1 {
		p = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil && v >= 1 {
		l = v
	}
	if l > 200 {
		l = 200
	}
	return int32(p), int32(l), int32((p - 1) * l)
}

func parseAdminID(w http.ResponseWriter, r *http.Request, rt *Router) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_id", "invalid numeric id")
		return 0, false
	}
	return id, true
}

func (rt *Router) handleListPublishedArticles(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	items, total, err := rt.service.ListPublishedArticles(r.Context(), limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if items == nil {
		items = []schema.ArticleResponse{}
	}
	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items, "page": page, "page_size": limit,
		"total": total, "has_next": int(page)*int(limit) < total,
	})
}

func (rt *Router) handleGetArticleBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	art, err := rt.service.GetPublishedArticleBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			rt.writeError(w, r, http.StatusNotFound, "not_found", "article not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, art)
}

func (rt *Router) handleGetPageBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	page, err := rt.service.GetPageBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			rt.writeError(w, r, http.StatusNotFound, "not_found", "page not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, page)
}

func (rt *Router) handleListFaqs(w http.ResponseWriter, r *http.Request) {
	faqs, err := rt.service.ListFaqs(r.Context(), true)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if faqs == nil {
		faqs = []schema.FaqResponse{}
	}
	rt.writeJSON(w, http.StatusOK, faqs)
}

func (rt *Router) handleListMenus(w http.ResponseWriter, r *http.Request) {
	location := chi.URLParam(r, "location")
	menus, err := rt.service.ListMenuItems(r.Context(), location)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if menus == nil {
		menus = []schema.MenuItemResponse{}
	}
	rt.writeJSON(w, http.StatusOK, menus)
}

func (rt *Router) handleAdminListArticles(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	items, total, err := rt.service.ListAdminArticles(r.Context(), limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if items == nil {
		items = []schema.ArticleResponse{}
	}
	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items, "page": page, "page_size": limit,
		"total": total, "has_next": int(page)*int(limit) < total,
	})
}

func (rt *Router) handleAdminCreateArticle(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateArticleRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	art, err := rt.service.CreateArticle(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "title and body are required")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Location", "/api/v1/admin/cms/articles")
	rt.writeJSON(w, http.StatusCreated, art)
}

func (rt *Router) handleAdminPublishArticle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(w, r, rt)
	if !ok {
		return
	}
	art, err := rt.service.PublishArticle(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			rt.writeError(w, r, http.StatusNotFound, "not_found", "article not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, art)
}

func (rt *Router) handleAdminListPages(w http.ResponseWriter, r *http.Request) {
	pages, err := rt.service.ListPages(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if pages == nil {
		pages = []schema.PageResponse{}
	}
	rt.writeJSON(w, http.StatusOK, pages)
}

func (rt *Router) handleAdminUpsertPage(w http.ResponseWriter, r *http.Request) {
	var req schema.UpsertPageRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	page, err := rt.service.UpsertPage(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "slug, title and body are required")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, page)
}

func (rt *Router) handleAdminListBanners(w http.ResponseWriter, r *http.Request) {
	position := r.URL.Query().Get("position")
	banners, err := rt.service.ListBanners(r.Context(), position)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if banners == nil {
		banners = []schema.BannerResponse{}
	}
	rt.writeJSON(w, http.StatusOK, banners)
}

func (rt *Router) handleAdminCreateBanner(w http.ResponseWriter, r *http.Request) {
	var req schema.UpsertBannerRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	ban, err := rt.service.CreateBanner(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "title and image_url are required")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Location", "/api/v1/admin/cms/banners")
	rt.writeJSON(w, http.StatusCreated, ban)
}

func (rt *Router) handleAdminUpsertMenu(w http.ResponseWriter, r *http.Request) {
	var req schema.UpsertMenuItemRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	item, err := rt.service.UpsertMenuItem(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "location, label and url are required")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Location", "/api/v1/admin/cms/menus")
	rt.writeJSON(w, http.StatusCreated, item)
}

func (rt *Router) handleAdminListSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := rt.service.ListSettings(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if settings == nil {
		settings = []schema.SettingResponse{}
	}
	rt.writeJSON(w, http.StatusOK, settings)
}

func (rt *Router) handleAdminUpdateSetting(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		var bodyReq struct {
			Key       string `json:"key"`
			Value     string `json:"value"`
			GroupName string `json:"group_name,omitempty"`
		}
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&bodyReq); err == nil && bodyReq.Key != "" {
			key = bodyReq.Key
			st, err := rt.service.SetSetting(r.Context(), key, schema.UpdateSettingRequest{Value: bodyReq.Value, GroupName: bodyReq.GroupName})
			if err != nil {
				rt.writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
				return
			}
			rt.writeJSON(w, http.StatusOK, st)
			return
		}
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "setting key is required")
		return
	}
	var req schema.UpdateSettingRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	st, err := rt.service.SetSetting(r.Context(), key, req)
	if err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, st)
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
