package schema

import (
	"time"
)

type ArticleResponse struct {
	ID          int64
	Code        string     `json:"code"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Excerpt     *string    `json:"excerpt,omitempty"`
	Body        string     `json:"body"`
	Author      *string    `json:"author,omitempty"`
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CreateArticleRequest struct {
	Title   string  `json:"title"`
	Slug    string  `json:"slug,omitempty"`
	Excerpt *string `json:"excerpt,omitempty"`
	Body    string  `json:"body"`
	Author  *string `json:"author,omitempty"`
}

type PageResponse struct {
	Code            string    `json:"code"`
	Slug            string    `json:"slug"`
	Title           string    `json:"title"`
	Body            string    `json:"body"`
	MetaTitle       *string   `json:"meta_title,omitempty"`
	MetaDescription *string   `json:"meta_description,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UpsertPageRequest struct {
	Slug            string  `json:"slug"`
	Title           string  `json:"title"`
	Body            string  `json:"body"`
	MetaTitle       *string `json:"meta_title,omitempty"`
	MetaDescription *string `json:"meta_description,omitempty"`
}

type FaqResponse struct {
	Code      string    `json:"code"`
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	SortOrder int       `json:"sort_order"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type UpsertFaqRequest struct {
	Question  string `json:"question"`
	Answer    string `json:"answer"`
	SortOrder int    `json:"sort_order,omitempty"`
	IsActive  *bool  `json:"is_active,omitempty"`
}

type BannerResponse struct {
	Code      string    `json:"code"`
	Title     string    `json:"title"`
	ImageUrl  string    `json:"image_url"`
	LinkUrl   *string   `json:"link_url,omitempty"`
	Position  string    `json:"position"`
	SortOrder int       `json:"sort_order"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type UpsertBannerRequest struct {
	Title     string  `json:"title"`
	ImageUrl  string  `json:"image_url"`
	LinkUrl   *string `json:"link_url,omitempty"`
	Position  string  `json:"position,omitempty"`
	SortOrder int     `json:"sort_order,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
}

type MenuItemResponse struct {
	Code      string             `json:"code"`
	Location  string             `json:"location"`
	Label     string             `json:"label"`
	Url       string             `json:"url"`
	ParentID  *int64             `json:"parent_id,omitempty"`
	SortOrder int                `json:"sort_order"`
	IsActive  bool               `json:"is_active"`
	Children  []MenuItemResponse `json:"children,omitempty"`
}

type UpsertMenuItemRequest struct {
	Location  string `json:"location"`
	Label     string `json:"label"`
	Url       string `json:"url"`
	ParentID  *int64 `json:"parent_id,omitempty"`
	SortOrder int    `json:"sort_order,omitempty"`
	IsActive  *bool  `json:"is_active,omitempty"`
}

type SettingResponse struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	GroupName string    `json:"group_name"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateSettingRequest struct {
	Value     string `json:"value"`
	GroupName string `json:"group_name,omitempty"`
}

type SeoTemplateResponse struct {
	Key                 string         `json:"key"`
	Name                string         `json:"name"`
	TitleTemplate       string         `json:"title_template"`
	DescriptionTemplate *string        `json:"description_template,omitempty"`
	StructuredData      map[string]any `json:"structured_data,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

type CreateSeoTemplateRequest struct {
	Key                 string         `json:"key"`
	Name                string         `json:"name"`
	TitleTemplate       string         `json:"title_template"`
	DescriptionTemplate *string        `json:"description_template,omitempty"`
	StructuredData      map[string]any `json:"structured_data,omitempty"`
}

type UpdateSeoTemplateRequest struct {
	Name                string         `json:"name"`
	TitleTemplate       string         `json:"title_template"`
	DescriptionTemplate *string        `json:"description_template,omitempty"`
	StructuredData      map[string]any `json:"structured_data,omitempty"`
}

type SeoSettingsResponse struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpsertSeoSettingRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SitemapURL struct {
	Loc        string     `xml:"loc"`
	LastMod    *time.Time `xml:"lastmod,omitempty"`
	ChangeFreq string     `xml:"changefreq,omitempty"`
	Priority   float64    `xml:"priority,omitempty"`
}

type HomepageSection struct {
	Code        string         `json:"code"`
	SectionType string         `json:"section_type"`
	Title       *string        `json:"title,omitempty"`
	IsActive    *bool          `json:"is_active,omitempty"`
	SortOrder   int            `json:"sort_order"`
	Config      map[string]any `json:"config,omitempty"`
}

type UpdateHomepageRequest struct {
	Sections          []HomepageSection `json:"sections"`
	ExpectedUpdatedAt *time.Time        `json:"expected_updated_at,omitempty"`
}

type HomepageResponse struct {
	Sections          []HomepageSection `json:"sections"`
	PublishedSections []HomepageSection `json:"published_sections,omitempty"`
	PublishedAt       *time.Time        `json:"published_at,omitempty"`
	UpdatedAt         time.Time         `json:"updated_at"`
	UpdatedBy         *string           `json:"updated_by,omitempty"`
}
