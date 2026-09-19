package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/cms/repository"
	"github.com/atlas-platform/backend/internal/modules/cms/schema"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFound     = errors.New("cms resource not found")
	ErrInvalidInput = errors.New("invalid cms request")
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
		strconv.FormatUint(uint64(rand.Uint32()), 36),
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
