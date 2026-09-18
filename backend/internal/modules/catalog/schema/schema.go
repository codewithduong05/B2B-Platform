package schema

import (
	"time"
)

type ProductSummary struct {
	Code             string     `json:"code"`
	Slug             string     `json:"slug"`
	Name             string     `json:"name"`
	ShortDescription string     `json:"short_description"`
	CategoryCode     string     `json:"category_code"`
	BrandCode        string     `json:"brand_code"`
	HandlingClass    string     `json:"handling_class"`
	BaseUnitCode     string     `json:"base_unit_code"`
	SupplierCode     string     `json:"supplier_code"`
	Status           string     `json:"status"`
	IsActive         bool       `json:"is_active"`
	IsFeatured       bool       `json:"is_featured"`
	BasePriceMinor   *int64     `json:"base_price_minor"`
	Currency         string     `json:"currency"`
	TrackInventory   bool       `json:"track_inventory"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
}

type ProductDetail struct {
	ProductSummary
	Description   string                    `json:"description"`
	WeightGrams   *int                      `json:"weight_grams,omitempty"`
	LengthMM      *int                      `json:"length_mm,omitempty"`
	WidthMM       *int                      `json:"width_mm,omitempty"`
	HeightMM      *int                      `json:"height_mm,omitempty"`
	GTIN          string                    `json:"gtin,omitempty"`
	SKU           string                    `json:"sku,omitempty"`
	Category      CategorySummary           `json:"category"`
	Brand         BrandSummary              `json:"brand,omitempty"`
	HandlingClass string                    `json:"handling_class"`
	BaseUnit      UnitSummary               `json:"base_unit"`
	Supplier      SupplierSummary           `json:"supplier,omitempty"`
	Units         []ProductUnitSummary      `json:"units"`
	Media         []ProductMediaSummary     `json:"media"`
	Attributes    []ProductAttributeSummary `json:"attributes"`
}

type ProductUnitSummary struct {
	Code             string  `json:"code"`
	UnitCode         string  `json:"unit_code"`
	UnitName         string  `json:"unit_name"`
	UnitSymbol       string  `json:"unit_symbol"`
	ConversionFactor float64 `json:"conversion_factor"`
	IsDefault        bool    `json:"is_default"`
	PriceMinor       *int64  `json:"price_minor,omitempty"`
}

type ProductMediaSummary struct {
	Code          string `json:"code"`
	URL           string `json:"url"`
	AltText       string `json:"alt_text"`
	MediaType     string `json:"media_type"`
	SortOrder     int    `json:"sort_order"`
	IsPrimary     bool   `json:"is_primary"`
	WidthPx       *int   `json:"width_px,omitempty"`
	HeightPx      *int   `json:"height_px,omitempty"`
	FileSizeBytes *int64 `json:"file_size_bytes,omitempty"`
	MimeType      string `json:"mime_type"`
}

type ProductAttributeSummary struct {
	AttributeID   string   `json:"attribute_id"`
	AttributeName string   `json:"attribute_name"`
	AttributeType string   `json:"attribute_type"`
	ValueID       string   `json:"value_id,omitempty"`
	ValueName     string   `json:"value_name,omitempty"`
	TextValue     string   `json:"text_value,omitempty"`
	NumberValue   *float64 `json:"number_value,omitempty"`
	BooleanValue  *bool    `json:"boolean_value,omitempty"`
}

type CategorySummary struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	ParentCode  string `json:"parent_code,omitempty"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
}

type CategoryDetail struct {
	CategorySummary
	Children []CategorySummary `json:"children,omitempty"`
}

type BrandSummary struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	LogoURL     string `json:"logo_url,omitempty"`
	WebsiteURL  string `json:"website_url,omitempty"`
	IsActive    bool   `json:"is_active"`
}

type UnitSummary struct {
	Code             string  `json:"code"`
	Name             string  `json:"name"`
	Symbol           string  `json:"symbol"`
	UnitType         string  `json:"unit_type"`
	BaseUnitCode     string  `json:"base_unit_code,omitempty"`
	ConversionFactor float64 `json:"conversion_factor"`
	IsActive         bool    `json:"is_active"`
}

type HandlingClassSummary struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	TemperatureMinC *int   `json:"temperature_min_c,omitempty"`
	TemperatureMaxC *int   `json:"temperature_max_c,omitempty"`
}

type SupplierSummary struct {
	Code        string `json:"code"`
	SupplierID  int64  `json:"supplier_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	LogoURL     string `json:"logo_url,omitempty"`
	IsActive    bool   `json:"is_active"`
}

type Facet struct {
	FacetType string `json:"facet_type"`
	ValueID   string `json:"value_id"`
	ValueName string `json:"value_name"`
	Count     int64  `json:"count"`
}

type AttributeFacet struct {
	AttributeID   string `json:"attribute_id"`
	AttributeName string `json:"attribute_name"`
	AttributeType string `json:"attribute_type"`
	ValueID       string `json:"value_id"`
	ValueName     string `json:"value_name"`
	Count         int64  `json:"count"`
}

type ProductListRequest struct {
	Page          int     `form:"page,default=1" validate:"min=1"`
	PageSize      int     `form:"page_size,default=24" validate:"min=1,max=100"`
	Category      *string `form:"category"`
	Brand         *string `form:"brand"`
	HandlingClass *string `form:"handling_class"`
	Supplier      *string `form:"supplier"`
	Query         *string `form:"q"`
	Sort          *string `form:"sort"`
	Cursor        *string `form:"cursor"`
	Limit         int     `form:"limit,default=24" validate:"min=1,max=100"`
}

type ProductListResponse struct {
	Items    []ProductSummary `json:"items"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	Total    int64            `json:"total"`
	HasNext  bool             `json:"has_next"`
	Facets   []Facet          `json:"facets,omitempty"`
}

type ProductDetailResponse struct {
	Product ProductDetail `json:"product"`
}

type CategoryListRequest struct {
	Page     int     `form:"page,default=1" validate:"min=1"`
	PageSize int     `form:"page_size,default=50" validate:"min=1,max=200"`
	Parent   *string `form:"parent"`
}

type CategoryListResponse struct {
	Items    []CategorySummary `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	Total    int64             `json:"total"`
	HasNext  bool              `json:"has_next"`
}

type BrandListRequest struct {
	Page     int `form:"page,default=1" validate:"min=1"`
	PageSize int `form:"page_size,default=50" validate:"min=1,max=200"`
}

type BrandListResponse struct {
	Items    []BrandSummary `json:"items"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Total    int64          `json:"total"`
	HasNext  bool           `json:"has_next"`
}

type CreateProductRequest struct {
	Code             string `json:"code" validate:"required,len=26"`
	Slug             string `json:"slug" validate:"required"`
	Name             string `json:"name" validate:"required"`
	Description      string `json:"description"`
	ShortDescription string `json:"short_description"`
	CategoryID       int64  `json:"category_id" validate:"required"`
	BrandID          *int64 `json:"brand_id"`
	HandlingClass    string `json:"handling_class" validate:"required"`
	BaseUnitID       int64  `json:"base_unit_id" validate:"required"`
	SupplierID       *int64 `json:"supplier_id"`
	Status           string `json:"status" validate:"oneof=draft published archived"`
	BasePriceMinor   *int64 `json:"base_price_minor"`
	Currency         string `json:"currency" validate:"len=3"`
	TrackInventory   bool   `json:"track_inventory"`
	WeightGrams      *int   `json:"weight_grams"`
	LengthMM         *int   `json:"length_mm"`
	WidthMM          *int   `json:"width_mm"`
	HeightMM         *int   `json:"height_mm"`
	GTIN             string `json:"gtin"`
	SKU              string `json:"sku"`
	SEOTitle         string `json:"seo_title"`
	SEODescription   string `json:"seo_description"`
	SEOKeywords      string `json:"seo_keywords"`
}

type UpdateProductRequest struct {
	Slug             *string `json:"slug"`
	Name             *string `json:"name"`
	Description      *string `json:"description"`
	ShortDescription *string `json:"short_description"`
	CategoryID       *int64  `json:"category_id"`
	BrandID          *int64  `json:"brand_id"`
	HandlingClass    *string `json:"handling_class"`
	BaseUnitID       *int64  `json:"base_unit_id"`
	SupplierID       *int64  `json:"supplier_id"`
	Status           *string `json:"status" validate:"omitempty,oneof=draft published archived"`
	IsActive         *bool   `json:"is_active"`
	IsFeatured       *bool   `json:"is_featured"`
	SortOrder        *int    `json:"sort_order"`
	BasePriceMinor   *int64  `json:"base_price_minor"`
	Currency         *string `json:"currency" validate:"omitempty,len=3"`
	TrackInventory   *bool   `json:"track_inventory"`
	WeightGrams      *int    `json:"weight_grams"`
	LengthMM         *int    `json:"length_mm"`
	WidthMM          *int    `json:"width_mm"`
	HeightMM         *int    `json:"height_mm"`
	GTIN             *string `json:"gtin"`
	SKU              *string `json:"sku"`
	SEOTitle         *string `json:"seo_title"`
	SEODescription   *string `json:"seo_description"`
	SEOKeywords      *string `json:"seo_keywords"`
}

type CreateCategoryRequest struct {
	Code           string `json:"code" validate:"required,len=26"`
	Name           string `json:"name" validate:"required"`
	Slug           string `json:"slug" validate:"required"`
	Description    string `json:"description"`
	ParentID       *int64 `json:"parent_id"`
	SortOrder      int    `json:"sort_order"`
	SEOTitle       string `json:"seo_title"`
	SEODescription string `json:"seo_description"`
}

type UpdateCategoryRequest struct {
	Name           *string `json:"name"`
	Slug           *string `json:"slug"`
	Description    *string `json:"description"`
	ParentID       *int64  `json:"parent_id"`
	SortOrder      *int    `json:"sort_order"`
	IsActive       *bool   `json:"is_active"`
	SEOTitle       *string `json:"seo_title"`
	SEODescription *string `json:"seo_description"`
}

type CreateBrandRequest struct {
	Code        string `json:"code" validate:"required,len=26"`
	Name        string `json:"name" validate:"required"`
	Slug        string `json:"slug" validate:"required"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
	WebsiteURL  string `json:"website_url"`
}

type UpdateBrandRequest struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
	LogoURL     *string `json:"logo_url"`
	WebsiteURL  *string `json:"website_url"`
	IsActive    *bool   `json:"is_active"`
}

type CreateUnitRequest struct {
	Code             string  `json:"code" validate:"required,len=26"`
	Name             string  `json:"name" validate:"required"`
	Symbol           string  `json:"symbol" validate:"required"`
	UnitType         string  `json:"unit_type" validate:"required,oneof=base derived"`
	BaseUnitID       *int64  `json:"base_unit_id"`
	ConversionFactor float64 `json:"conversion_factor" validate:"required,gt=0"`
}

type UpdateUnitRequest struct {
	Name             *string  `json:"name"`
	Symbol           *string  `json:"symbol"`
	UnitType         *string  `json:"unit_type" validate:"omitempty,oneof=base derived"`
	BaseUnitID       *int64   `json:"base_unit_id"`
	ConversionFactor *float64 `json:"conversion_factor" validate:"omitempty,gt=0"`
	IsActive         *bool    `json:"is_active"`
}

type CreateSupplierRequest struct {
	Code        string `json:"code" validate:"required,len=26"`
	SupplierID  int64  `json:"supplier_id" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Slug        string `json:"slug" validate:"required"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
}

type UpdateSupplierRequest struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
	LogoURL     *string `json:"logo_url"`
	IsActive    *bool   `json:"is_active"`
}

type FacetResponse struct {
	Facets []Facet `json:"facets"`
}

type AttributeFacetResponse struct {
	Facets []AttributeFacet `json:"facets"`
}
