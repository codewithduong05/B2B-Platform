package router

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

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
	rt.router.Route("/cms", func(r chi.Router) {
		r.Get("/articles", rt.handleListPublishedArticles)
		r.Get("/articles/{slug}", rt.handleGetArticleBySlug)
		r.Get("/pages/{slug}", rt.handleGetPageBySlug)
		r.Get("/faqs", rt.handleListFaqs)
		r.Get("/menus/{location}", rt.handleListMenus)
		r.Get("/homepage", rt.handleGetPublishedHomepage)

		r.Post("/newsletter/subscribe", rt.handleSubscribe)
		r.Post("/newsletter/confirm", rt.handleConfirmSubscription)
		r.Post("/newsletter/unsubscribe", rt.handleUnsubscribe)
	})

	rt.router.Get("/sitemap.xml", rt.handleSitemap)
	rt.router.Get("/robots.txt", rt.handleRobotsTxt)
	rt.router.Get("/feed/products.xml", rt.handleProductFeed)

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

		r.Get("/homepage", rt.handleGetAdminHomepage)
		r.Put("/homepage", rt.handleUpdateDraftHomepage)
		r.Post("/homepage/publish", rt.handlePublishHomepage)

		r.Route("/seo", func(r chi.Router) {
			r.Get("/templates", rt.handleAdminListSeoTemplates)
			r.Post("/templates", rt.handleAdminCreateSeoTemplate)
			r.Put("/templates/{key}", rt.handleAdminUpdateSeoTemplate)
			r.Get("/settings", rt.handleAdminListSeoSettings)
			r.Put("/settings", rt.handleAdminUpsertSeoSetting)
		})

		r.Route("/newsletter", func(r chi.Router) {
			r.Get("/subscribers", rt.handleAdminListSubscribers)
			r.Get("/subscribers/export", rt.handleAdminExportSubscribers)
			r.Get("/subscribers/{code}", rt.handleAdminGetSubscriber)
			r.Delete("/subscribers/{code}", rt.handleAdminDeleteSubscriber)
			r.Get("/stats", rt.handleAdminNewsletterStats)
		})
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

func (rt *Router) handleSitemap(w http.ResponseWriter, r *http.Request) {
	baseURL := r.URL.Query().Get("base_url")
	xmlData, err := rt.service.GenerateSitemap(r.Context(), baseURL)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(xmlData))
}

func (rt *Router) handleRobotsTxt(w http.ResponseWriter, r *http.Request) {
	baseURL := r.URL.Query().Get("base_url")
	txt, err := rt.service.GenerateRobotsTxt(r.Context(), baseURL)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(txt))
}

func (rt *Router) handleProductFeed(w http.ResponseWriter, r *http.Request) {
	baseURL := r.URL.Query().Get("base_url")
	xmlData, err := rt.service.GenerateProductFeed(r.Context(), baseURL)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(xmlData))
}

func (rt *Router) handleAdminListSeoTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := rt.service.ListSeoTemplates(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if templates == nil {
		templates = []schema.SeoTemplateResponse{}
	}
	rt.writeJSON(w, http.StatusOK, templates)
}

func (rt *Router) handleAdminCreateSeoTemplate(w http.ResponseWriter, r *http.Request) {
	var req schema.CreateSeoTemplateRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	t, err := rt.service.CreateSeoTemplate(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "key, name, and title_template are required")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusCreated, t)
}

func (rt *Router) handleAdminUpdateSeoTemplate(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "template key is required")
		return
	}
	var req schema.UpdateSeoTemplateRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	t, err := rt.service.UpdateSeoTemplate(r.Context(), key, req)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			rt.writeError(w, r, http.StatusNotFound, "not_found", "seo template not found")
			return
		}
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "name and title_template are required")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, t)
}

func (rt *Router) handleAdminListSeoSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := rt.service.ListSeoSettings(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if settings == nil {
		settings = []schema.SeoSettingsResponse{}
	}
	rt.writeJSON(w, http.StatusOK, settings)
}

func (rt *Router) handleAdminUpsertSeoSetting(w http.ResponseWriter, r *http.Request) {
	var req schema.UpsertSeoSettingRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	st, err := rt.service.UpsertSeoSetting(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "key and value are required")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, st)
}

func (rt *Router) handleGetPublishedHomepage(w http.ResponseWriter, r *http.Request) {
	homepage, err := rt.service.GetPublishedHomepage(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, homepage)
}

func (rt *Router) handleGetAdminHomepage(w http.ResponseWriter, r *http.Request) {
	homepage, err := rt.service.GetAdminHomepage(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, homepage)
}

func (rt *Router) handleUpdateDraftHomepage(w http.ResponseWriter, r *http.Request) {
	var req schema.UpdateHomepageRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	if req.Sections == nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "sections array is required")
		return
	}
	for _, sec := range req.Sections {
		if sec.Code == "" || sec.SectionType == "" {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "section code and section_type are required")
			return
		}
	}
	updatedBy := r.Context().Value("user_id")
	var updatedByPtr *string
	if uid, ok := updatedBy.(string); ok && uid != "" {
		updatedByPtr = &uid
	}
	homepage, err := rt.service.UpdateDraftHomepage(r.Context(), req, updatedByPtr)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid section data")
			return
		}
		if errors.Is(err, service.ErrConcurrentModification) {
			rt.writeError(w, r, http.StatusConflict, "concurrent_modification", "the layout was modified by another request; re-fetch and retry")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, homepage)
}

func (rt *Router) handlePublishHomepage(w http.ResponseWriter, r *http.Request) {
	updatedBy := r.Context().Value("user_id")
	var updatedByPtr *string
	if uid, ok := updatedBy.(string); ok && uid != "" {
		updatedByPtr = &uid
	}
	homepage, err := rt.service.PublishHomepage(r.Context(), updatedByPtr)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusUnprocessableEntity, "no_active_sections", "draft has no active sections to publish")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, homepage)
}

func (rt *Router) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	var req schema.SubscribeRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	if req.Email == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "email is required")
		return
	}
	sub, err := rt.service.Subscribe(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_email", "valid email is required")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusAccepted, sub)
}

func (rt *Router) handleConfirmSubscription(w http.ResponseWriter, r *http.Request) {
	var req schema.ConfirmRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	if req.Token == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_token", "token is required")
		return
	}
	sub, err := rt.service.ConfirmSubscription(r.Context(), req.Token)
	if err != nil {
		if errors.Is(err, service.ErrTokenNotFound) {
			rt.writeError(w, r, http.StatusNotFound, "token_not_found", "token not found")
			return
		}
		if errors.Is(err, service.ErrTokenExpired) {
			rt.writeError(w, r, http.StatusBadRequest, "invalid_token", "token has expired")
			return
		}
		if errors.Is(err, service.ErrTokenAlreadyUsed) {
			rt.writeError(w, r, http.StatusConflict, "token_already_used", "token has already been used")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, sub)
}

func (rt *Router) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	var req schema.UnsubscribeRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	if req.Token != "" {
		if err := rt.service.UnsubscribeByToken(r.Context(), req.Token); err != nil {
			if errors.Is(err, service.ErrTokenNotFound) {
				rt.writeError(w, r, http.StatusNotFound, "token_not_found", "token not found")
				return
			}
			if errors.Is(err, service.ErrTokenExpired) {
				rt.writeError(w, r, http.StatusBadRequest, "invalid_token", "token has expired")
				return
			}
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		rt.writeJSON(w, http.StatusOK, map[string]string{"status": "processed"})
		return
	}
	if req.Email != "" {
		if err := rt.service.RequestUnsubscribeByEmail(r.Context(), req.Email); err != nil {
			if errors.Is(err, service.ErrInvalidInput) {
				rt.writeError(w, r, http.StatusBadRequest, "invalid_email", "valid email is required")
				return
			}
			rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		rt.writeJSON(w, http.StatusOK, map[string]string{"status": "processed"})
		return
	}
	rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "token or email is required")
}

func (rt *Router) handleAdminListSubscribers(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := parsePage(r)
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("q")
	subs, total, err := rt.service.ListNewsletterSubscribers(r.Context(), status, search, limit, offset)
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if subs == nil {
		subs = []schema.NewsletterSubscriber{}
	}
	rt.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": subs, "page": page, "page_size": limit,
		"total": total, "has_next": int(page)*int(limit) < total,
	})
}

func (rt *Router) handleAdminGetSubscriber(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "subscriber code is required")
		return
	}
	sub, err := rt.service.GetNewsletterSubscriberByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			rt.writeError(w, r, http.StatusNotFound, "not_found", "subscriber not found")
			return
		}
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, sub)
}

func (rt *Router) handleAdminDeleteSubscriber(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		rt.writeError(w, r, http.StatusBadRequest, "invalid_request", "subscriber code is required")
		return
	}
	if err := rt.service.DeleteNewsletterSubscriber(r.Context(), code); err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *Router) handleAdminExportSubscribers(w http.ResponseWriter, r *http.Request) {
	subs, err := rt.service.ExportNewsletterSubscribers(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=subscribers.csv")
	w.WriteHeader(http.StatusOK)
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"email", "first_name", "status", "subscribed_at", "confirmed_at", "source"})
	for _, sub := range subs {
		firstName := ""
		if sub.FirstName != nil {
			firstName = *sub.FirstName
		}
		source := ""
		if sub.Source != nil {
			source = *sub.Source
		}
		subscribedAt := ""
		if sub.SubscribedAt != nil {
			subscribedAt = sub.SubscribedAt.Format(time.RFC3339)
		}
		confirmedAt := ""
		if sub.ConfirmedAt != nil {
			confirmedAt = sub.ConfirmedAt.Format(time.RFC3339)
		}
		_ = cw.Write([]string{sub.Email, firstName, sub.Status, subscribedAt, confirmedAt, source})
	}
	cw.Flush()
}

func (rt *Router) handleAdminNewsletterStats(w http.ResponseWriter, r *http.Request) {
	stats, err := rt.service.GetNewsletterStats(r.Context())
	if err != nil {
		rt.writeError(w, r, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	rt.writeJSON(w, http.StatusOK, stats)
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
