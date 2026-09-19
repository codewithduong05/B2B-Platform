package schema

import (
	"time"
)

type PriceListSummary struct {
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	Status        string     `json:"status"`
	PriceType     string     `json:"price_type"`
	Currency      string     `json:"currency"`
	EffectiveFrom time.Time  `json:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to,omitempty"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type PriceListDetail struct {
	PriceListSummary
	Items []PriceListItemSummary `json:"items"`
}

type PriceListItemSummary struct {
	ID            int64                 `json:"id"`
	ProductID     int64                 `json:"product_id"`
	UnitID        int64                 `json:"unit_id"`
	Currency      string                `json:"currency"`
	PriceMinor    int64                 `json:"price_minor"`
	MinQuantity   int                   `json:"min_quantity"`
	MaxQuantity   *int                  `json:"max_quantity,omitempty"`
	EffectiveFrom time.Time             `json:"effective_from"`
	EffectiveTo   *time.Time            `json:"effective_to,omitempty"`
	IsActive      bool                  `json:"is_active"`
	Tiers         []QuantityTierSummary `json:"tiers,omitempty"`
}

type QuantityTierSummary struct {
	ID          int64 `json:"id"`
	MinQuantity int   `json:"min_quantity"`
	MaxQuantity *int  `json:"max_quantity,omitempty"`
	PriceMinor  int64 `json:"price_minor"`
	IsActive    bool  `json:"is_active"`
}

type PriceListAssignmentSummary struct {
	Code           string     `json:"code"`
	PriceListID    int64      `json:"price_list_id"`
	BuyerProfileID int64      `json:"buyer_profile_id"`
	AssignedBy     int64      `json:"assigned_by,omitempty"`
	EffectiveFrom  time.Time  `json:"effective_from"`
	EffectiveTo    *time.Time `json:"effective_to,omitempty"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type PriceListListRequest struct {
	Page     int     `form:"page,default=1" validate:"min=1"`
	PageSize int     `form:"page_size,default=20" validate:"min=1,max=100"`
	Status   *string `form:"status"`
	IsActive *bool   `form:"is_active"`
}

type PriceListListResponse struct {
	Items    []PriceListSummary `json:"items"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int64              `json:"total"`
	HasNext  bool               `json:"has_next"`
}

type PriceListDetailResponse struct {
	PriceList PriceListDetail `json:"price_list"`
}

type CreatePriceListRequest struct {
	Code          string `json:"code" validate:"required,len=26"`
	Name          string `json:"name" validate:"required"`
	Description   string `json:"description"`
	Status        string `json:"status" validate:"oneof=draft active inactive archived"`
	PriceType     string `json:"price_type" validate:"oneof=standard contract promotional"`
	Currency      string `json:"currency" validate:"len=3"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to"`
	IsActive      bool   `json:"is_active"`
}

type UpdatePriceListRequest struct {
	Name          *string `json:"name"`
	Description   *string `json:"description"`
	Status        *string `json:"status" validate:"omitempty,oneof=draft active inactive archived"`
	PriceType     *string `json:"price_type" validate:"omitempty,oneof=standard contract promotional"`
	Currency      *string `json:"currency" validate:"omitempty,len=3"`
	EffectiveFrom *string `json:"effective_from"`
	EffectiveTo   *string `json:"effective_to"`
	IsActive      *bool   `json:"is_active"`
}

type ReplacePriceListEntriesRequest struct {
	Items []CreatePriceListItemRequest `json:"items" validate:"required,min=1,dive"`
}

type CreatePriceListItemRequest struct {
	ProductID     int64  `json:"product_id" validate:"required"`
	UnitID        int64  `json:"unit_id" validate:"required"`
	Currency      string `json:"currency" validate:"len=3"`
	PriceMinor    int64  `json:"price_minor" validate:"required,gte=0"`
	MinQuantity   int    `json:"min_quantity" validate:"required,gte=1"`
	MaxQuantity   *int   `json:"max_quantity" validate:"omitempty,gte=1"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to"`
	IsActive      bool   `json:"is_active"`
}

type PriceQuoteRequest struct {
	Lines []PriceQuoteLineRequest `json:"lines" validate:"required,min=1,dive"`
}

type PriceQuoteLineRequest struct {
	ProductID      int64  `json:"product_id" validate:"required"`
	UnitID         int64  `json:"unit_id" validate:"required"`
	Quantity       int    `json:"quantity" validate:"required,gte=1"`
	PriceListID    *int64 `json:"price_list_id,omitempty"`
	BuyerProfileID *int64 `json:"buyer_profile_id,omitempty"`
}

type PriceQuoteLineResponse struct {
	ProductID       int64  `json:"product_id"`
	UnitID          int64  `json:"unit_id"`
	Quantity        int    `json:"quantity"`
	UnitPriceMinor  int64  `json:"unit_price_minor"`
	TotalPriceMinor int64  `json:"total_price_minor"`
	Currency        string `json:"currency"`
	PriceListID     int64  `json:"price_list_id"`
	PriceListItemID int64  `json:"price_list_item_id"`
	PriceType       string `json:"price_type"`
	TierID          *int64 `json:"tier_id,omitempty"`
}

type PriceQuoteResponse struct {
	Lines []PriceQuoteLineResponse `json:"lines"`
}

type CreatePriceListAssignmentRequest struct {
	PriceListID    int64  `json:"price_list_id" validate:"required"`
	BuyerProfileID int64  `json:"buyer_profile_id" validate:"required"`
	EffectiveFrom  string `json:"effective_from"`
	EffectiveTo    string `json:"effective_to"`
	IsActive       bool   `json:"is_active"`
}

type PriceListAssignmentListRequest struct {
	Page        int    `form:"page,default=1" validate:"min=1"`
	PageSize    int    `form:"page_size,default=20" validate:"min=1,max=100"`
	PriceListID *int64 `form:"price_list_id"`
	BuyerID     *int64 `form:"buyer_id"`
	IsActive    *bool  `form:"is_active"`
}

type PriceListAssignmentListResponse struct {
	Items    []PriceListAssignmentSummary `json:"items"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
	Total    int64                        `json:"total"`
	HasNext  bool                         `json:"has_next"`
}
