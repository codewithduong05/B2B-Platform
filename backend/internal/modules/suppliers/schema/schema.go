package schema

import (
	"time"
)

type SupplierResponse struct {
	Code        string    `json:"code"`
	CompanyName string    `json:"company_name"`
	ContactName string    `json:"contact_name,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ApplySupplierRequest struct {
	CompanyName  string `json:"company_name"`
	ContactName  string `json:"contact_name,omitempty"`
	ContactEmail string `json:"contact_email,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`
}

type UpdateSupplierRequest struct {
	CompanyName  *string `json:"company_name,omitempty"`
	ContactName  *string `json:"contact_name,omitempty"`
	ContactEmail *string `json:"contact_email,omitempty"`
	ContactPhone *string `json:"contact_phone,omitempty"`
}

type ApproveSupplierRequest struct {
	UserID *int64 `json:"user_id,omitempty"`
}

type ContractResponse struct {
	Version   int        `json:"version"`
	Terms     string     `json:"terms"`
	ValidFrom *time.Time `json:"valid_from,omitempty"`
	ValidTo   *time.Time `json:"valid_to,omitempty"`
	IsCurrent bool       `json:"is_current"`
	CreatedAt time.Time  `json:"created_at"`
}

type SetContractRequest struct {
	Terms     string  `json:"terms"`
	ValidFrom *string `json:"valid_from,omitempty"`
	ValidTo   *string `json:"valid_to,omitempty"`
}

type SupplierOrderLineResponse struct {
	ProductCode     string `json:"product_code"`
	ProductName     string `json:"product_name"`
	Quantity        int    `json:"quantity"`
	UnitCode        string `json:"unit_code"`
	UnitPriceMinor  int64  `json:"unit_price_minor"`
	TotalPriceMinor int64  `json:"total_price_minor"`
}

type SupplierOrderResponse struct {
	Code           string                      `json:"code"`
	Status         string                      `json:"status"`
	Currency       string                      `json:"currency"`
	SubtotalMinor  int64                       `json:"subtotal_minor"`
	DiscountsMinor int64                       `json:"discounts_minor"`
	TotalMinor     int64                       `json:"total_minor"`
	PlacedAt       time.Time                   `json:"placed_at"`
	Lines          []SupplierOrderLineResponse `json:"lines"`
}

type PushStockRequest struct {
	ProductCode string  `json:"product_code"`
	LotNumber   string  `json:"lot_number"`
	Quantity    int     `json:"quantity"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
}

type StockPushResponse struct {
	LotCode           string `json:"lot_code"`
	LotNumber         string `json:"lot_number"`
	AvailableQuantity int    `json:"available_quantity"`
}

type AcknowledgeOrderRequest struct {
	Note string `json:"note,omitempty"`
}
