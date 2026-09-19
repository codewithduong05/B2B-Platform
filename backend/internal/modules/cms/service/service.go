package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	mathrand "math/rand/v2"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/cms/repository"
	"github.com/atlas-platform/backend/internal/modules/cms/schema"
	"github.com/jackc/pgx/v5"
)

var jsonMarshal = json.Marshal
var jsonUnmarshal = json.Unmarshal

var (
	ErrNotFound               = errors.New("cms resource not found")
	ErrInvalidInput           = errors.New("invalid cms request")
	ErrConcurrentModification = errors.New("concurrent modification detected")
)

type CMSService struct {
	db   *database.DB
	repo *repository.CMSRepository
}

func NewCMSService(db *database.DB) *CMSService {
	return &CMSService{db: db, repo: repository.NewCMSRepository(db)}
}

func newCode(prefix string) string {
	return fmt.Sprintf("%s%s%s",
		prefix,
		strconv.FormatInt(time.Now().UnixNano(), 36),
		strconv.FormatUint(uint64(mathrand.Uint32()), 36),
	)
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

// Articles
func (s *CMSService) CreateArticle(ctx context.Context, req schema.CreateArticleRequest) (*schema.ArticleResponse, error) {
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Body) == "" {
		return nil, ErrInvalidInput
	}
	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		slug = slugify(req.Title)
	}
	a, err := s.repo.CreateArticle(ctx, newCode("art_"), slug, req.Title, req.Excerpt, req.Body, req.Author)
	if err != nil {
		return nil, fmt.Errorf("create article: %w", err)
	}
	return toArticleResponse(a), nil
}

func (s *CMSService) PublishArticle(ctx context.Context, id int64) (*schema.ArticleResponse, error) {
	a, err := s.repo.PublishArticle(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("publish article: %w", err)
	}
	return toArticleResponse(a), nil
}

func (s *CMSService) GetPublishedArticleBySlug(ctx context.Context, slug string) (*schema.ArticleResponse, error) {
	a, err := s.repo.GetArticleBySlug(ctx, slug, true)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get article: %w", err)
	}
	return toArticleResponse(a), nil
}

func (s *CMSService) ListPublishedArticles(ctx context.Context, limit, offset int32) ([]schema.ArticleResponse, int, error) {
	articles, total, err := s.repo.ListArticles(ctx, "published", limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]schema.ArticleResponse, 0, len(articles))
	for _, a := range articles {
		out = append(out, *toArticleResponse(a))
	}
	return out, total, nil
}

func (s *CMSService) ListAdminArticles(ctx context.Context, limit, offset int32) ([]schema.ArticleResponse, int, error) {
	articles, total, err := s.repo.ListArticles(ctx, "", limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]schema.ArticleResponse, 0, len(articles))
	for _, a := range articles {
		out = append(out, *toArticleResponse(a))
	}
	return out, total, nil
}

// Pages
func (s *CMSService) UpsertPage(ctx context.Context, req schema.UpsertPageRequest) (*schema.PageResponse, error) {
	if strings.TrimSpace(req.Slug) == "" || strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Body) == "" {
		return nil, ErrInvalidInput
	}
	p, err := s.repo.UpsertPage(ctx, newCode("pag_"), req.Slug, req.Title, req.Body, req.MetaTitle, req.MetaDescription)
	if err != nil {
		return nil, fmt.Errorf("upsert page: %w", err)
	}
	return toPageResponse(p), nil
}

func (s *CMSService) GetPageBySlug(ctx context.Context, slug string) (*schema.PageResponse, error) {
	p, err := s.repo.GetPageBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get page: %w", err)
	}
	return toPageResponse(p), nil
}

func (s *CMSService) ListPages(ctx context.Context) ([]schema.PageResponse, error) {
	pages, err := s.repo.ListPages(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]schema.PageResponse, 0, len(pages))
	for _, p := range pages {
		out = append(out, *toPageResponse(p))
	}
	return out, nil
}

// FAQs
func (s *CMSService) CreateFaq(ctx context.Context, req schema.UpsertFaqRequest) (*schema.FaqResponse, error) {
	if strings.TrimSpace(req.Question) == "" || strings.TrimSpace(req.Answer) == "" {
		return nil, ErrInvalidInput
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	f, err := s.repo.CreateFaq(ctx, newCode("faq_"), req.Question, req.Answer, req.SortOrder, isActive)
	if err != nil {
		return nil, err
	}
	return toFaqResponse(f), nil
}

func (s *CMSService) ListFaqs(ctx context.Context, activeOnly bool) ([]schema.FaqResponse, error) {
	faqs, err := s.repo.ListFaqs(ctx, activeOnly)
	if err != nil {
		return nil, err
	}
	out := make([]schema.FaqResponse, 0, len(faqs))
	for _, f := range faqs {
		out = append(out, *toFaqResponse(f))
	}
	return out, nil
}

// Banners
func (s *CMSService) CreateBanner(ctx context.Context, req schema.UpsertBannerRequest) (*schema.BannerResponse, error) {
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.ImageUrl) == "" {
		return nil, ErrInvalidInput
	}
	position := req.Position
	if position == "" {
		position = "home_hero"
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	b, err := s.repo.CreateBanner(ctx, newCode("ban_"), req.Title, req.ImageUrl, req.LinkUrl, position, req.SortOrder, isActive)
	if err != nil {
		return nil, err
	}
	return toBannerResponse(b), nil
}

func (s *CMSService) ListBanners(ctx context.Context, position string) ([]schema.BannerResponse, error) {
	banners, err := s.repo.ListBanners(ctx, position)
	if err != nil {
		return nil, err
	}
	out := make([]schema.BannerResponse, 0, len(banners))
	for _, b := range banners {
		out = append(out, *toBannerResponse(b))
	}
	return out, nil
}

// Menus
func (s *CMSService) UpsertMenuItem(ctx context.Context, req schema.UpsertMenuItemRequest) (*schema.MenuItemResponse, error) {
	if strings.TrimSpace(req.Location) == "" || strings.TrimSpace(req.Label) == "" || strings.TrimSpace(req.Url) == "" {
		return nil, ErrInvalidInput
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	m, err := s.repo.UpsertMenuItem(ctx, newCode("mnu_"), req.Location, req.Label, req.Url, req.ParentID, req.SortOrder, isActive)
	if err != nil {
		return nil, err
	}
	return toMenuItemResponse(m), nil
}

func (s *CMSService) ListMenuItems(ctx context.Context, location string) ([]schema.MenuItemResponse, error) {
	items, err := s.repo.ListMenuItems(ctx, location)
	if err != nil {
		return nil, err
	}
	itemMap := make(map[int64]*schema.MenuItemResponse)
	var roots []schema.MenuItemResponse

	for _, mi := range items {
		resp := schema.MenuItemResponse{
			Code:      mi.Code,
			Location:  mi.Location,
			Label:     mi.Label,
			Url:       mi.Url,
			ParentID:  mi.ParentID,
			SortOrder: mi.SortOrder,
			IsActive:  mi.IsActive,
			Children:  []schema.MenuItemResponse{},
		}
		itemMap[mi.ID] = &resp
	}

	for _, mi := range items {
		resp := itemMap[mi.ID]
		if mi.ParentID != nil && itemMap[*mi.ParentID] != nil {
			parent := itemMap[*mi.ParentID]
			parent.Children = append(parent.Children, *resp)
		} else {
			roots = append(roots, *resp)
		}
	}
	if roots == nil {
		roots = []schema.MenuItemResponse{}
	}
	return roots, nil
}

// Settings
func (s *CMSService) SetSetting(ctx context.Context, key string, req schema.UpdateSettingRequest) (*schema.SettingResponse, error) {
	if strings.TrimSpace(key) == "" || strings.TrimSpace(req.Value) == "" {
		return nil, ErrInvalidInput
	}
	groupName := req.GroupName
	if groupName == "" {
		groupName = "general"
	}
	st, err := s.repo.SetSetting(ctx, key, req.Value, groupName)
	if err != nil {
		return nil, err
	}
	return toSettingResponse(st), nil
}

func (s *CMSService) ListSettings(ctx context.Context) ([]schema.SettingResponse, error) {
	settings, err := s.repo.ListSettings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]schema.SettingResponse, 0, len(settings))
	for _, st := range settings {
		out = append(out, *toSettingResponse(st))
	}
	return out, nil
}

func toArticleResponse(a repository.Article) *schema.ArticleResponse {
	return &schema.ArticleResponse{
		ID: a.ID, Code: a.Code, Slug: a.Slug, Title: a.Title, Excerpt: a.Excerpt,
		Body: a.Body, Author: a.Author, Status: a.Status, PublishedAt: a.PublishedAt,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

func toPageResponse(p repository.Page) *schema.PageResponse {
	return &schema.PageResponse{
		Code: p.Code, Slug: p.Slug, Title: p.Title, Body: p.Body,
		MetaTitle: p.MetaTitle, MetaDescription: p.MetaDescription,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toFaqResponse(f repository.Faq) *schema.FaqResponse {
	return &schema.FaqResponse{
		Code: f.Code, Question: f.Question, Answer: f.Answer,
		SortOrder: f.SortOrder, IsActive: f.IsActive, CreatedAt: f.CreatedAt,
	}
}

func toBannerResponse(b repository.Banner) *schema.BannerResponse {
	return &schema.BannerResponse{
		Code: b.Code, Title: b.Title, ImageUrl: b.ImageUrl, LinkUrl: b.LinkUrl,
		Position: b.Position, SortOrder: b.SortOrder, IsActive: b.IsActive, CreatedAt: b.CreatedAt,
	}
}

func toMenuItemResponse(m repository.MenuItem) *schema.MenuItemResponse {
	return &schema.MenuItemResponse{
		Code: m.Code, Location: m.Location, Label: m.Label, Url: m.Url,
		ParentID: m.ParentID, SortOrder: m.SortOrder, IsActive: m.IsActive, Children: []schema.MenuItemResponse{},
	}
}

func toSettingResponse(st repository.Setting) *schema.SettingResponse {
	return &schema.SettingResponse{
		Key: st.Key, Value: st.Value, GroupName: st.GroupName, UpdatedAt: st.UpdatedAt,
	}
}

// SEO Templates
func (s *CMSService) CreateSeoTemplate(ctx context.Context, req schema.CreateSeoTemplateRequest) (*schema.SeoTemplateResponse, error) {
	if strings.TrimSpace(req.Key) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.TitleTemplate) == "" {
		return nil, ErrInvalidInput
	}
	var sd []byte
	if req.StructuredData != nil {
		var err error
		sd, err = jsonMarshal(req.StructuredData)
		if err != nil {
			return nil, ErrInvalidInput
		}
	}
	t, err := s.repo.CreateSeoTemplate(ctx, req.Key, req.Name, req.TitleTemplate, req.DescriptionTemplate, sd)
	if err != nil {
		return nil, fmt.Errorf("create seo template: %w", err)
	}
	return toSeoTemplateResponse(t), nil
}

func (s *CMSService) ListSeoTemplates(ctx context.Context) ([]schema.SeoTemplateResponse, error) {
	templates, err := s.repo.ListSeoTemplates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]schema.SeoTemplateResponse, 0, len(templates))
	for _, t := range templates {
		out = append(out, *toSeoTemplateResponse(t))
	}
	return out, nil
}

func (s *CMSService) UpdateSeoTemplate(ctx context.Context, key string, req schema.UpdateSeoTemplateRequest) (*schema.SeoTemplateResponse, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.TitleTemplate) == "" {
		return nil, ErrInvalidInput
	}
	var sd []byte
	if req.StructuredData != nil {
		var err error
		sd, err = jsonMarshal(req.StructuredData)
		if err != nil {
			return nil, ErrInvalidInput
		}
	}
	t, err := s.repo.UpdateSeoTemplate(ctx, key, req.Name, req.TitleTemplate, req.DescriptionTemplate, sd)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update seo template: %w", err)
	}
	return toSeoTemplateResponse(t), nil
}

// SEO Settings
func (s *CMSService) UpsertSeoSetting(ctx context.Context, req schema.UpsertSeoSettingRequest) (*schema.SeoSettingsResponse, error) {
	if strings.TrimSpace(req.Key) == "" || strings.TrimSpace(req.Value) == "" {
		return nil, ErrInvalidInput
	}
	st, err := s.repo.UpsertSeoSetting(ctx, req.Key, req.Value)
	if err != nil {
		return nil, fmt.Errorf("upsert seo setting: %w", err)
	}
	return toSeoSettingResponse(st), nil
}

func (s *CMSService) ListSeoSettings(ctx context.Context) ([]schema.SeoSettingsResponse, error) {
	settings, err := s.repo.ListSeoSettings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]schema.SeoSettingsResponse, 0, len(settings))
	for _, st := range settings {
		out = append(out, *toSeoSettingResponse(st))
	}
	return out, nil
}

// Sitemap
func (s *CMSService) GenerateSitemap(ctx context.Context, baseURL string) (string, error) {
	entries, err := s.repo.ListSitemapEntries(ctx)
	if err != nil {
		return "", fmt.Errorf("list sitemap entries: %w", err)
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "https://example.com"
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString("\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	b.WriteString("\n")
	for _, e := range entries {
		b.WriteString("  <url>\n")
		b.WriteString("    <loc>")
		b.WriteString(xmlEscape(baseURL + e.Loc))
		b.WriteString("</loc>\n")
		b.WriteString("    <lastmod>")
		b.WriteString(e.LastMod.Format(time.RFC3339))
		b.WriteString("</lastmod>\n")
		b.WriteString("  </url>\n")
	}
	b.WriteString("</urlset>")
	return b.String(), nil
}

// Robots.txt
func (s *CMSService) GenerateRobotsTxt(ctx context.Context, baseURL string) (string, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "https://example.com"
	}
	var b strings.Builder
	b.WriteString("User-agent: *\n")
	b.WriteString("Allow: /\n")
	b.WriteString("Disallow: /admin/\n")
	b.WriteString("\n")
	b.WriteString("Sitemap: ")
	b.WriteString(baseURL)
	b.WriteString("/sitemap.xml\n")
	return b.String(), nil
}

// Product Feed
func (s *CMSService) GenerateProductFeed(ctx context.Context, baseURL string) (string, error) {
	entries, err := s.repo.ListProductFeedEntries(ctx)
	if err != nil {
		return "", fmt.Errorf("list product feed entries: %w", err)
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "https://example.com"
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString("\n")
	b.WriteString(`<feed xmlns="http://www.w3.org/2005/Atom">`)
	b.WriteString("\n")
	b.WriteString("  <title>Atlas Platform Product Feed</title>\n")
	b.WriteString("  <link href=\"")
	b.WriteString(xmlEscape(baseURL))
	b.WriteString("\"/>\n")
	b.WriteString("  <updated>")
	b.WriteString(time.Now().UTC().Format(time.RFC3339))
	b.WriteString("</updated>\n")
	for _, e := range entries {
		b.WriteString("  <entry>\n")
		b.WriteString("    <id>")
		b.WriteString(xmlEscape(e.Code))
		b.WriteString("</id>\n")
		b.WriteString("    <title>")
		b.WriteString(xmlEscape(e.Name))
		b.WriteString("</title>\n")
		b.WriteString("    <link href=\"")
		b.WriteString(xmlEscape(baseURL + "/products/" + e.Slug))
		b.WriteString("\"/>\n")
		b.WriteString("    <updated>")
		b.WriteString(e.UpdatedAt.Format(time.RFC3339))
		b.WriteString("</updated>\n")
		if e.Description != nil {
			b.WriteString("    <summary>")
			b.WriteString(xmlEscape(*e.Description))
			b.WriteString("</summary>\n")
		}
		b.WriteString("    <category term=\"")
		b.WriteString(xmlEscape(e.CategoryName))
		b.WriteString("\"/>\n")
		if e.BrandName != nil {
			b.WriteString("    <author><name>")
			b.WriteString(xmlEscape(*e.BrandName))
			b.WriteString("</name></author>\n")
		}
		if e.Gtin != nil {
			b.WriteString("    <gtin>")
			b.WriteString(xmlEscape(*e.Gtin))
			b.WriteString("</gtin>\n")
		}
		if e.Sku != nil {
			b.WriteString("    <sku>")
			b.WriteString(xmlEscape(*e.Sku))
			b.WriteString("</sku>\n")
		}
		if e.PriceMinor != nil {
			b.WriteString("    <price>")
			b.WriteString(fmt.Sprintf("%d %s", *e.PriceMinor, e.Currency))
			b.WriteString("</price>\n")
		}
		if e.ImageURL != nil {
			b.WriteString("    <link rel=\"enclosure\" href=\"")
			b.WriteString(xmlEscape(*e.ImageURL))
			b.WriteString("\"/>\n")
		}
		b.WriteString("  </entry>\n")
	}
	b.WriteString("</feed>")
	return b.String(), nil
}

// Homepage
func (s *CMSService) GetPublishedHomepage(ctx context.Context) (*schema.HomepageResponse, error) {
	h, err := s.repo.GetHomepageLayout(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &schema.HomepageResponse{
				Sections: []schema.HomepageSection{},
			}, nil
		}
		return nil, fmt.Errorf("get homepage layout: %w", err)
	}
	var published []schema.HomepageSection
	if len(h.PublishedSections) > 0 {
		if err := jsonUnmarshal(h.PublishedSections, &published); err != nil {
			return nil, fmt.Errorf("unmarshal published sections: %w", err)
		}
	}
	if published == nil {
		published = []schema.HomepageSection{}
	}
	return &schema.HomepageResponse{
		Sections:    published,
		PublishedAt: h.PublishedAt,
		UpdatedAt:   h.UpdatedAt,
	}, nil
}

func (s *CMSService) GetAdminHomepage(ctx context.Context) (*schema.HomepageResponse, error) {
	h, err := s.repo.GetHomepageLayout(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &schema.HomepageResponse{
				Sections:          []schema.HomepageSection{},
				PublishedSections: []schema.HomepageSection{},
			}, nil
		}
		return nil, fmt.Errorf("get homepage layout: %w", err)
	}
	var draft, published []schema.HomepageSection
	if len(h.DraftSections) > 0 {
		if err := jsonUnmarshal(h.DraftSections, &draft); err != nil {
			return nil, fmt.Errorf("unmarshal draft sections: %w", err)
		}
	}
	if len(h.PublishedSections) > 0 {
		if err := jsonUnmarshal(h.PublishedSections, &published); err != nil {
			return nil, fmt.Errorf("unmarshal published sections: %w", err)
		}
	}
	if draft == nil {
		draft = []schema.HomepageSection{}
	}
	if published == nil {
		published = []schema.HomepageSection{}
	}
	return &schema.HomepageResponse{
		Sections:          draft,
		PublishedSections: published,
		PublishedAt:       h.PublishedAt,
		UpdatedAt:         h.UpdatedAt,
		UpdatedBy:         h.UpdatedBy,
	}, nil
}

func (s *CMSService) UpdateDraftHomepage(ctx context.Context, req schema.UpdateHomepageRequest, updatedBy *string) (*schema.HomepageResponse, error) {
	if req.Sections == nil {
		return nil, ErrInvalidInput
	}
	for _, sec := range req.Sections {
		if strings.TrimSpace(sec.Code) == "" || strings.TrimSpace(sec.SectionType) == "" {
			return nil, ErrInvalidInput
		}
	}
	for i := range req.Sections {
		req.Sections[i].SortOrder = i
	}
	draftJSON, err := jsonMarshal(req.Sections)
	if err != nil {
		return nil, fmt.Errorf("marshal draft sections: %w", err)
	}
	h, err := s.repo.UpdateDraftLayout(ctx, draftJSON, updatedBy, req.ExpectedUpdatedAt)
	if err != nil {
		if errors.Is(err, repository.ErrConcurrentModification) {
			return nil, ErrConcurrentModification
		}
		return nil, fmt.Errorf("update draft layout: %w", err)
	}
	var draft []schema.HomepageSection
	if len(h.DraftSections) > 0 {
		if err := jsonUnmarshal(h.DraftSections, &draft); err != nil {
			return nil, fmt.Errorf("unmarshal draft sections: %w", err)
		}
	}
	if draft == nil {
		draft = []schema.HomepageSection{}
	}
	var published []schema.HomepageSection
	if len(h.PublishedSections) > 0 {
		if err := jsonUnmarshal(h.PublishedSections, &published); err != nil {
			return nil, fmt.Errorf("unmarshal published sections: %w", err)
		}
	}
	if published == nil {
		published = []schema.HomepageSection{}
	}
	return &schema.HomepageResponse{
		Sections:          draft,
		PublishedSections: published,
		PublishedAt:       h.PublishedAt,
		UpdatedAt:         h.UpdatedAt,
		UpdatedBy:         h.UpdatedBy,
	}, nil
}

func (s *CMSService) PublishHomepage(ctx context.Context, updatedBy *string) (*schema.HomepageResponse, error) {
	var result *schema.HomepageResponse
	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCMSRepositoryWithTx(tx)
		h, err := txRepo.GetHomepageLayoutForUpdate(ctx)
		if err != nil {
			return fmt.Errorf("get homepage layout for update: %w", err)
		}
		var draft []schema.HomepageSection
		if len(h.DraftSections) > 0 {
			if err := jsonUnmarshal(h.DraftSections, &draft); err != nil {
				return fmt.Errorf("unmarshal draft sections: %w", err)
			}
		}
		activeSections := make([]schema.HomepageSection, 0, len(draft))
		for _, sec := range draft {
			isActive := sec.IsActive == nil || *sec.IsActive
			if isActive {
				activeSections = append(activeSections, sec)
			}
		}
		if len(activeSections) == 0 {
			return ErrInvalidInput
		}
		publishedJSON, err := jsonMarshal(activeSections)
		if err != nil {
			return fmt.Errorf("marshal published sections: %w", err)
		}
		h, err = txRepo.PublishHomepage(ctx, publishedJSON, updatedBy)
		if err != nil {
			return fmt.Errorf("publish homepage: %w", err)
		}
		var published []schema.HomepageSection
		if len(h.PublishedSections) > 0 {
			if err := jsonUnmarshal(h.PublishedSections, &published); err != nil {
				return fmt.Errorf("unmarshal published sections: %w", err)
			}
		}
		if published == nil {
			published = []schema.HomepageSection{}
		}
		var draftOut []schema.HomepageSection
		if len(h.DraftSections) > 0 {
			if err := jsonUnmarshal(h.DraftSections, &draftOut); err != nil {
				return fmt.Errorf("unmarshal draft sections: %w", err)
			}
		}
		if draftOut == nil {
			draftOut = []schema.HomepageSection{}
		}
		result = &schema.HomepageResponse{
			Sections:          draftOut,
			PublishedSections: published,
			PublishedAt:       h.PublishedAt,
			UpdatedAt:         h.UpdatedAt,
			UpdatedBy:         h.UpdatedBy,
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return nil, ErrInvalidInput
		}
		return nil, err
	}
	return result, nil
}

func toSeoTemplateResponse(t repository.SeoTemplate) *schema.SeoTemplateResponse {
	resp := &schema.SeoTemplateResponse{
		Key:                 t.Key,
		Name:                t.Name,
		TitleTemplate:       t.TitleTemplate,
		DescriptionTemplate: t.DescriptionTemplate,
		CreatedAt:           t.CreatedAt,
		UpdatedAt:           t.UpdatedAt,
	}
	if len(t.StructuredData) > 0 {
		_ = jsonUnmarshal(t.StructuredData, &resp.StructuredData)
	}
	return resp
}

func toSeoSettingResponse(s repository.SeoSetting) *schema.SeoSettingsResponse {
	return &schema.SeoSettingsResponse{
		Key: s.Key, Value: s.Value, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

var (
	ErrTokenExpired     = errors.New("token expired")
	ErrTokenAlreadyUsed = errors.New("token already used")
	ErrTokenNotFound    = errors.New("token not found")
)

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func toNewsletterSubscriberResponse(s repository.NewsletterSubscriber) *schema.NewsletterSubscriber {
	return &schema.NewsletterSubscriber{
		Code:           s.Code,
		Email:          s.Email,
		Status:         s.Status,
		FirstName:      s.FirstName,
		Source:         s.Source,
		SubscribedAt:   s.SubscribedAt,
		ConfirmedAt:    s.ConfirmedAt,
		UnsubscribedAt: s.UnsubscribedAt,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

func (s *CMSService) Subscribe(ctx context.Context, req schema.SubscribeRequest) (*schema.NewsletterSubscriber, error) {
	email := normalizeEmail(req.Email)
	if email == "" {
		return nil, ErrInvalidInput
	}
	if !isValidEmail(email) {
		return nil, ErrInvalidInput
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	tokenHash := hashToken(token)
	tokenExpiry := time.Now().Add(48 * time.Hour)

	existing, err := s.repo.GetNewsletterSubscriberByEmail(ctx, email)
	if err == nil {
		switch existing.Status {
		case "confirmed":
			return toNewsletterSubscriberResponse(existing), nil
		case "pending", "unsubscribed":
			updated, err := s.repo.UpdateNewsletterSubscriberStatus(ctx, existing.ID, "pending", &tokenHash, &tokenExpiry, nil, nil)
			if err != nil {
				return nil, fmt.Errorf("re-subscribe: %w", err)
			}
			_ = token
			return toNewsletterSubscriberResponse(updated), nil
		}
	}

	firstName := strings.TrimSpace(req.FirstName)
	var firstNamePtr *string
	if firstName != "" {
		firstNamePtr = &firstName
	}
	source := strings.TrimSpace(req.Source)
	var sourcePtr *string
	if source != "" {
		sourcePtr = &source
	}

	sub, err := s.repo.CreateNewsletterSubscriber(ctx, newCode("sub_"), email, firstNamePtr, sourcePtr, tokenHash, tokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("create subscriber: %w", err)
	}
	_ = token
	return toNewsletterSubscriberResponse(sub), nil
}

func (s *CMSService) ConfirmSubscription(ctx context.Context, token string) (*schema.NewsletterSubscriber, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrInvalidInput
	}

	tokenHash := hashToken(token)
	sub, err := s.repo.GetNewsletterSubscriberByConfirmTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("lookup token: %w", err)
	}

	if sub.Status == "confirmed" {
		return nil, ErrTokenAlreadyUsed
	}

	if sub.TokenExpiresAt != nil && sub.TokenExpiresAt.Before(time.Now()) {
		return nil, ErrTokenExpired
	}

	updated, err := s.repo.UpdateNewsletterSubscriberStatus(ctx, sub.ID, "confirmed", nil, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("confirm subscriber: %w", err)
	}
	return toNewsletterSubscriberResponse(updated), nil
}

func (s *CMSService) UnsubscribeByToken(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return ErrInvalidInput
	}

	tokenHash := hashToken(token)
	sub, err := s.repo.GetNewsletterSubscriberByUnsubTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTokenNotFound
		}
		return fmt.Errorf("lookup unsubscribe token: %w", err)
	}

	if sub.UnsubscribeTokenExpiresAt != nil && sub.UnsubscribeTokenExpiresAt.Before(time.Now()) {
		return ErrTokenExpired
	}

	if sub.Status == "unsubscribed" {
		return nil
	}

	_, err = s.repo.UpdateNewsletterSubscriberStatus(ctx, sub.ID, "unsubscribed", nil, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("unsubscribe: %w", err)
	}
	return nil
}

func (s *CMSService) RequestUnsubscribeByEmail(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	if email == "" {
		return ErrInvalidInput
	}

	sub, err := s.repo.GetNewsletterSubscriberByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("lookup subscriber: %w", err)
	}

	if sub.Status == "unsubscribed" {
		return nil
	}

	token, err := generateToken()
	if err != nil {
		return fmt.Errorf("generate unsubscribe token: %w", err)
	}
	tokenHash := hashToken(token)
	tokenExpiry := time.Now().Add(48 * time.Hour)

	_, err = s.repo.UpdateNewsletterSubscriberStatus(ctx, sub.ID, sub.Status, nil, nil, &tokenHash, &tokenExpiry)
	if err != nil {
		return fmt.Errorf("set unsubscribe token: %w", err)
	}
	_ = token
	return nil
}

func (s *CMSService) ListNewsletterSubscribers(ctx context.Context, status, search string, limit, offset int32) ([]schema.NewsletterSubscriber, int, error) {
	subs, total, err := s.repo.ListNewsletterSubscribers(ctx, status, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]schema.NewsletterSubscriber, 0, len(subs))
	for _, sub := range subs {
		out = append(out, *toNewsletterSubscriberResponse(sub))
	}
	return out, total, nil
}

func (s *CMSService) GetNewsletterSubscriberByCode(ctx context.Context, code string) (*schema.NewsletterSubscriber, error) {
	sub, err := s.repo.GetNewsletterSubscriberByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get subscriber: %w", err)
	}
	return toNewsletterSubscriberResponse(sub), nil
}

func (s *CMSService) DeleteNewsletterSubscriber(ctx context.Context, code string) error {
	return s.repo.DeleteNewsletterSubscriber(ctx, code)
}

func (s *CMSService) GetNewsletterStats(ctx context.Context) (*schema.SubscriberStatsResponse, error) {
	stats, err := s.repo.GetNewsletterStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}
	return &schema.SubscriberStatsResponse{
		Total:             stats.Total,
		Pending:           stats.Pending,
		Confirmed:         stats.Confirmed,
		Unsubscribed:      stats.Unsubscribed,
		SubscribedToday:   stats.SubscribedToday,
		SubscribedLast30D: stats.SubscribedLast30D,
	}, nil
}

func (s *CMSService) ExportNewsletterSubscribers(ctx context.Context) ([]schema.NewsletterSubscriber, error) {
	subs, err := s.repo.ListConfirmedSubscribersForExport(ctx)
	if err != nil {
		return nil, fmt.Errorf("list subscribers for export: %w", err)
	}
	out := make([]schema.NewsletterSubscriber, 0, len(subs))
	for _, sub := range subs {
		out = append(out, *toNewsletterSubscriberResponse(sub))
	}
	return out, nil
}

var ErrNoDraft = errors.New("no draft to publish")

var ErrDuplicateDocType = errors.New("duplicate doc_type")

func isValidDocTypeSlug(s string) bool {
	if len(s) < 3 || len(s) > 64 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func toLegalDocumentResponse(d repository.LegalDocument) schema.LegalDocument {
	return schema.LegalDocument{
		DocType:        d.DocType,
		Title:          d.Title,
		CurrentVersion: d.CurrentVersion,
		HasDraft:       d.DraftTitle != nil,
		EffectiveAt:    d.EffectiveAt,
		UpdatedAt:      d.UpdatedAt,
		UpdatedBy:      d.UpdatedBy,
	}
}

func toLegalDocumentDraftResponse(d repository.LegalDocument) *schema.LegalDocumentDraft {
	if d.DraftTitle == nil || d.DraftBody == nil {
		return nil
	}
	draft := &schema.LegalDocumentDraft{
		DocType:     d.DocType,
		Title:       *d.DraftTitle,
		Body:        *d.DraftBody,
		BodyFormat:  d.DraftBodyFormat,
		EffectiveAt: d.DraftEffectiveAt,
		UpdatedBy:   d.DraftUpdatedBy,
	}
	if d.DraftUpdatedAt != nil {
		draft.UpdatedAt = *d.DraftUpdatedAt
	}
	return draft
}

func toLegalDocumentVersionResponse(v repository.LegalDocumentVersion) schema.LegalDocumentVersion {
	effAt := v.EffectiveAt
	pubAt := v.PublishedAt
	return schema.LegalDocumentVersion{
		DocType:     v.DocType,
		Version:     v.Version,
		Title:       v.Title,
		Body:        v.Body,
		BodyFormat:  v.BodyFormat,
		Status:      v.Status,
		EffectiveAt: &effAt,
		PublishedAt: &pubAt,
		CreatedAt:   v.CreatedAt,
		CreatedBy:   v.CreatedBy,
	}
}

func toVersionSummary(v repository.LegalDocumentVersion) schema.LegalDocumentVersionSummary {
	effAt := v.EffectiveAt
	pubAt := v.PublishedAt
	return schema.LegalDocumentVersionSummary{
		Version:     v.Version,
		Status:      v.Status,
		EffectiveAt: &effAt,
		PublishedAt: &pubAt,
		CreatedBy:   v.CreatedBy,
	}
}

func (s *CMSService) GetCurrentEffectiveVersion(ctx context.Context, docType string) (*schema.LegalDocumentVersion, error) {
	if !isValidDocTypeSlug(docType) {
		return nil, ErrInvalidInput
	}
	v, err := s.repo.GetCurrentEffectiveVersion(ctx, docType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get current effective version: %w", err)
	}
	resp := toLegalDocumentVersionResponse(v)
	return &resp, nil
}

func (s *CMSService) GetLegalDocumentVersion(ctx context.Context, docType string, version int) (*schema.LegalDocumentVersion, error) {
	if !isValidDocTypeSlug(docType) {
		return nil, ErrInvalidInput
	}
	v, err := s.repo.GetLegalDocumentVersionByNumber(ctx, docType, version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get legal document version: %w", err)
	}
	if v.Status != "published" && v.Status != "superseded" {
		return nil, ErrNotFound
	}
	resp := toLegalDocumentVersionResponse(v)
	return &resp, nil
}

func (s *CMSService) ListLegalDocuments(ctx context.Context, hasDraftFilter *bool, limit, offset int32) ([]schema.LegalDocument, int, error) {
	docs, total, err := s.repo.ListLegalDocuments(ctx, hasDraftFilter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]schema.LegalDocument, 0, len(docs))
	for _, d := range docs {
		out = append(out, toLegalDocumentResponse(d))
	}
	return out, total, nil
}

func (s *CMSService) CreateLegalDocument(ctx context.Context, req schema.CreateLegalDocumentRequest) (*schema.LegalDocument, error) {
	docType := strings.TrimSpace(req.DocType)
	title := strings.TrimSpace(req.Title)
	if !isValidDocTypeSlug(docType) {
		return nil, ErrInvalidInput
	}
	if title == "" {
		return nil, ErrInvalidInput
	}
	d, err := s.repo.CreateLegalDocument(ctx, docType, title)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrDuplicateDocType
		}
		return nil, fmt.Errorf("create legal document: %w", err)
	}
	resp := toLegalDocumentResponse(d)
	return &resp, nil
}

func (s *CMSService) GetLegalDocumentDetail(ctx context.Context, docType string) (*schema.LegalDocumentDetail, error) {
	if !isValidDocTypeSlug(docType) {
		return nil, ErrInvalidInput
	}
	d, err := s.repo.GetLegalDocument(ctx, docType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get legal document: %w", err)
	}
	versions, _, err := s.repo.ListLegalDocumentVersions(ctx, docType, "", 100, 0)
	if err != nil {
		return nil, fmt.Errorf("list legal document versions: %w", err)
	}
	summaries := make([]schema.LegalDocumentVersionSummary, 0, len(versions))
	for _, v := range versions {
		summaries = append(summaries, toVersionSummary(v))
	}
	detail := &schema.LegalDocumentDetail{
		LegalDocument: toLegalDocumentResponse(d),
		Draft:         toLegalDocumentDraftResponse(d),
		Versions:      summaries,
	}
	if detail.Versions == nil {
		detail.Versions = []schema.LegalDocumentVersionSummary{}
	}
	return detail, nil
}

func (s *CMSService) UpdateLegalDocumentDraft(ctx context.Context, docType string, req schema.UpdateLegalDocumentDraftRequest, updatedBy *string) (*schema.LegalDocumentDraft, error) {
	if !isValidDocTypeSlug(docType) {
		return nil, ErrInvalidInput
	}
	title := strings.TrimSpace(req.Title)
	body := strings.TrimSpace(req.Body)
	if title == "" || body == "" {
		return nil, ErrInvalidInput
	}
	bodyFormat := req.BodyFormat
	if bodyFormat == "" {
		bodyFormat = "markdown"
	}
	if bodyFormat != "markdown" && bodyFormat != "html" && bodyFormat != "plaintext" {
		return nil, ErrInvalidInput
	}
	d, err := s.repo.UpdateLegalDocumentDraft(ctx, docType, title, body, bodyFormat, req.EffectiveAt, updatedBy, req.ExpectedUpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if errors.Is(err, repository.ErrConcurrentModification) {
			return nil, ErrConcurrentModification
		}
		return nil, fmt.Errorf("update legal document draft: %w", err)
	}
	draft := toLegalDocumentDraftResponse(d)
	return draft, nil
}

func (s *CMSService) PublishLegalDocumentDraft(ctx context.Context, docType string, updatedBy *string) (*schema.LegalDocumentVersion, error) {
	if !isValidDocTypeSlug(docType) {
		return nil, ErrInvalidInput
	}
	var result *schema.LegalDocumentVersion
	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCMSRepositoryWithTx(tx)
		d, err := txRepo.GetLegalDocumentForUpdate(ctx, docType)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("get legal document for update: %w", err)
		}
		if d.DraftTitle == nil || d.DraftBody == nil {
			return ErrNoDraft
		}
		latestPublished, err := txRepo.GetLatestPublishedVersion(ctx, docType)
		if err == nil {
			if latestPublished.Title == *d.DraftTitle && latestPublished.Body == *d.DraftBody {
				resp := toLegalDocumentVersionResponse(latestPublished)
				result = &resp
				return nil
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("get latest published version: %w", err)
		}
		nextVersion, err := txRepo.GetNextVersion(ctx, docType)
		if err != nil {
			return fmt.Errorf("get next version: %w", err)
		}
		effectiveAt := time.Now()
		if d.DraftEffectiveAt != nil {
			effectiveAt = *d.DraftEffectiveAt
		}
		v, err := txRepo.PublishLegalDocumentVersion(ctx, docType, nextVersion, *d.DraftTitle, *d.DraftBody, d.DraftBodyFormat, effectiveAt, updatedBy)
		if err != nil {
			return fmt.Errorf("publish legal document version: %w", err)
		}
		if err := txRepo.SupersedePreviousVersion(ctx, docType, nextVersion); err != nil {
			return fmt.Errorf("supersede previous version: %w", err)
		}
		if err := txRepo.UpdateLegalDocumentCurrentVersion(ctx, docType, nextVersion, effectiveAt, updatedBy); err != nil {
			return fmt.Errorf("update current version: %w", err)
		}
		resp := toLegalDocumentVersionResponse(v)
		result = &resp
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrNoDraft) || errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrNotFound) {
			return nil, err
		}
		return nil, err
	}
	return result, nil
}

func (s *CMSService) ListLegalDocumentVersions(ctx context.Context, docType, status string, limit, offset int32) ([]schema.LegalDocumentVersion, int, error) {
	if !isValidDocTypeSlug(docType) {
		return nil, 0, ErrInvalidInput
	}
	versions, total, err := s.repo.ListLegalDocumentVersions(ctx, docType, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]schema.LegalDocumentVersion, 0, len(versions))
	for _, v := range versions {
		out = append(out, toLegalDocumentVersionResponse(v))
	}
	return out, total, nil
}

func (s *CMSService) SubmitContactEnquiry(ctx context.Context, req schema.SubmitContactEnquiryRequest) (*schema.ContactEnquiryResponse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrInvalidInput
	}
	if strings.TrimSpace(req.Email) == "" {
		return nil, ErrInvalidInput
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, ErrInvalidInput
	}
	if strings.TrimSpace(req.Subject) == "" {
		return nil, ErrInvalidInput
	}
	if strings.TrimSpace(req.Message) == "" {
		return nil, ErrInvalidInput
	}
	if len(req.Message) > 5000 {
		return nil, ErrInvalidInput
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	source := "contact_form"
	if req.Source != nil && strings.TrimSpace(*req.Source) != "" {
		source = strings.TrimSpace(*req.Source)
	}

	code := newCode("enq_")
	e, err := s.repo.CreateContactEnquiry(ctx, code, strings.TrimSpace(req.Name), email, req.Phone, req.Company, strings.TrimSpace(req.Subject), strings.TrimSpace(req.Message), source, req.IPAddress, req.UserAgent)
	if err != nil {
		return nil, fmt.Errorf("create contact enquiry: %w", err)
	}

	resp := toContactEnquiryResponse(e)
	return &resp, nil
}

func (s *CMSService) ListContactEnquiries(ctx context.Context, status, source, search string, dateFrom, dateTo *time.Time, limit, offset int32) ([]schema.ContactEnquiryResponse, int, error) {
	enquiries, total, err := s.repo.ListContactEnquiries(ctx, status, source, search, dateFrom, dateTo, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list contact enquiries: %w", err)
	}
	out := make([]schema.ContactEnquiryResponse, 0, len(enquiries))
	for _, e := range enquiries {
		out = append(out, toContactEnquiryResponse(e))
	}
	return out, total, nil
}

func (s *CMSService) GetContactEnquiryByCode(ctx context.Context, code string) (*schema.ContactEnquiryResponse, error) {
	e, err := s.repo.GetContactEnquiryByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get contact enquiry: %w", err)
	}
	resp := toContactEnquiryResponse(e)
	return &resp, nil
}

func (s *CMSService) UpdateContactEnquiryLeadID(ctx context.Context, code string, leadID int64) (*schema.ContactEnquiryResponse, error) {
	e, err := s.repo.GetContactEnquiryByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get contact enquiry: %w", err)
	}
	updated, err := s.repo.UpdateContactEnquiryLeadID(ctx, e.ID, leadID)
	if err != nil {
		return nil, fmt.Errorf("update contact enquiry lead id: %w", err)
	}
	resp := toContactEnquiryResponse(updated)
	return &resp, nil
}

func toContactEnquiryResponse(e repository.ContactEnquiry) schema.ContactEnquiryResponse {
	resp := schema.ContactEnquiryResponse{
		ID:        e.Code,
		Name:      e.Name,
		Email:     e.Email,
		Phone:     e.Phone,
		Company:   e.Company,
		Subject:   e.Subject,
		Message:   e.Message,
		Source:    e.Source,
		Status:    e.Status,
		IPAddress: e.IPAddress,
		UserAgent: e.UserAgent,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		RoutedAt:  e.RoutedAt,
	}
	if e.LeadID != nil {
		leadCode := fmt.Sprintf("lead_%d", *e.LeadID)
		resp.LeadID = &leadCode
	}
	return resp
}
