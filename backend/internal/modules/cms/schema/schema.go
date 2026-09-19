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

type NewsletterSubscriber struct {
	Code           string     `json:"code"`
	Email          string     `json:"email"`
	Status         string     `json:"status"`
	FirstName      *string    `json:"first_name,omitempty"`
	Source         *string    `json:"source,omitempty"`
	SubscribedAt   *time.Time `json:"subscribed_at,omitempty"`
	ConfirmedAt    *time.Time `json:"confirmed_at,omitempty"`
	UnsubscribedAt *time.Time `json:"unsubscribed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type SubscribeRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name,omitempty"`
	Source    string `json:"source,omitempty"`
}

type ConfirmRequest struct {
	Token string `json:"token"`
}

type UnsubscribeRequest struct {
	Token string `json:"token,omitempty"`
	Email string `json:"email,omitempty"`
}

type SubscriberListResponse struct {
	Items    []NewsletterSubscriber `json:"items"`
	Page     int32                  `json:"page"`
	PageSize int32                  `json:"page_size"`
	Total    int                    `json:"total"`
	HasNext  bool                   `json:"has_next"`
}

type SubscriberStatsResponse struct {
	Total             int `json:"total"`
	Pending           int `json:"pending"`
	Confirmed         int `json:"confirmed"`
	Unsubscribed      int `json:"unsubscribed"`
	SubscribedToday   int `json:"subscribed_today"`
	SubscribedLast30D int `json:"subscribed_last_30_days"`
}

type LegalDocument struct {
	DocType        string     `json:"doc_type"`
	Title          string     `json:"title"`
	CurrentVersion *int       `json:"current_version"`
	HasDraft       bool       `json:"has_draft"`
	EffectiveAt    *time.Time `json:"effective_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
	UpdatedBy      *string    `json:"updated_by,omitempty"`
}

type LegalDocumentDraft struct {
	DocType        string     `json:"doc_type"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	BodyFormat     string     `json:"body_format"`
	EffectiveAt    *time.Time `json:"effective_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
	UpdatedBy      *string    `json:"updated_by,omitempty"`
}

type LegalDocumentVersion struct {
	DocType     string     `json:"doc_type"`
	Version     int        `json:"version"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	BodyFormat  string     `json:"body_format"`
	Status      string     `json:"status"`
	EffectiveAt *time.Time `json:"effective_at,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   *string    `json:"created_by,omitempty"`
}

type LegalDocumentDetail struct {
	LegalDocument
	Draft    *LegalDocumentDraft            `json:"draft,omitempty"`
	Versions []LegalDocumentVersionSummary  `json:"versions,omitempty"`
}

type LegalDocumentVersionSummary struct {
	Version     int        `json:"version"`
	Status      string     `json:"status"`
	EffectiveAt *time.Time `json:"effective_at,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedBy   *string    `json:"created_by,omitempty"`
}

type CreateLegalDocumentRequest struct {
	DocType string `json:"doc_type"`
	Title   string `json:"title"`
}

type UpdateLegalDocumentDraftRequest struct {
	Title             string     `json:"title"`
	Body              string     `json:"body"`
	BodyFormat        string     `json:"body_format,omitempty"`
	EffectiveAt       *time.Time `json:"effective_at,omitempty"`
	ExpectedUpdatedAt *time.Time `json:"expected_updated_at,omitempty"`
}

type LegalDocumentListResponse struct {
	Items    []LegalDocument `json:"items"`
	Page     int32           `json:"page"`
	PageSize int32           `json:"page_size"`
	Total    int             `json:"total"`
	HasNext  bool            `json:"has_next"`
}

type LegalDocumentVersionListResponse struct {
	Items    []LegalDocumentVersion `json:"items"`
	Page     int32                  `json:"page"`
	PageSize int32                  `json:"page_size"`
	Total    int                    `json:"total"`
	HasNext  bool                   `json:"has_next"`
}

type ContactEnquiryResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Phone     *string    `json:"phone,omitempty"`
	Company   *string    `json:"company,omitempty"`
	Subject   string     `json:"subject"`
	Message   string     `json:"message"`
	Source    string     `json:"source"`
	Status    string     `json:"status"`
	LeadID    *string    `json:"lead_id,omitempty"`
	IPAddress *string    `json:"ip_address,omitempty"`
	UserAgent *string    `json:"user_agent,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	RoutedAt  *time.Time `json:"routed_at,omitempty"`
}

type SubmitContactEnquiryRequest struct {
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Phone     *string `json:"phone,omitempty"`
	Company   *string `json:"company,omitempty"`
	Subject   string  `json:"subject"`
	Message   string  `json:"message"`
	Source    *string `json:"source,omitempty"`
	IPAddress *string `json:"ip_address,omitempty"`
	UserAgent *string `json:"user_agent,omitempty"`
}

type ContactEnquiryListResponse struct {
	Items    []ContactEnquiryResponse `json:"items"`
	Page     int32                    `json:"page"`
	PageSize int32                    `json:"page_size"`
	Total    int                      `json:"total"`
	HasNext  bool                     `json:"has_next"`
}
