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
