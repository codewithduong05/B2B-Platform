package schema

import (
	"time"
)

type CartResponse struct {
	Code      string              `json:"code"`
	BuyerID   int64               `json:"buyer_id"`
	Currency  string              `json:"currency"`
	Suppliers []SupplierCartGroup `json:"suppliers"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

type SupplierCartGroup struct {
	SupplierCode string             `json:"supplier_code"`
	SupplierName string             `json:"supplier_name"`
	Items        []CartLineResponse `json:"items"`
}

type CartLineResponse struct {
	Code            string    `json:"code"`
	ProductCode     string    `json:"product_code"`
	ProductName     string    `json:"product_name"`
	Quantity        int       `json:"quantity"`
	UnitCode        string    `json:"unit_code"`
	SupplierCode    string    `json:"supplier_code"`
	UnitPriceMinor  *int64    `json:"unit_price_minor,omitempty"`
	TotalPriceMinor *int64    `json:"total_price_minor,omitempty"`
	Currency        string    `json:"currency,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type AddCartItemRequest struct {
	ProductCode string `json:"product_code"`
	Quantity    int    `json:"quantity"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity"`
}

type CartQuoteResponse struct {
	Suppliers      []SupplierQuoteGroup `json:"suppliers"`
	SubtotalMinor  int64                `json:"subtotal_minor"`
	DiscountsMinor int64                `json:"discounts_minor"`
	TotalMinor     int64                `json:"total_minor"`
	Currency       string               `json:"currency"`
}

type SupplierQuoteGroup struct {
	SupplierCode  string          `json:"supplier_code"`
	SupplierName  string          `json:"supplier_name"`
	Items         []CartLineQuote `json:"items"`
	SubtotalMinor int64           `json:"subtotal_minor"`
	Currency      string          `json:"currency"`
}

type CartLineQuote struct {
	Code              string `json:"code"`
	ProductCode       string `json:"product_code"`
	ProductName       string `json:"product_name"`
	Quantity          int    `json:"quantity"`
	UnitCode          string `json:"unit_code"`
	UnitPriceMinor    int64  `json:"unit_price_minor"`
	TotalPriceMinor   int64  `json:"total_price_minor"`
	Currency          string `json:"currency"`
	Available         bool   `json:"available"`
	AvailableQuantity int32  `json:"available_quantity"`
}

// CheckoutRequest carries no business fields: the cart, prices, stock and
// supplier split are all resolved server-side. Idempotency travels in the
// Idempotency-Key header. An explicit empty object is accepted.
type CheckoutRequest struct{}

// CheckoutResponse is the resulting order set: exactly one order per
// supplier. Replayed (idempotent) responses carry Replayed=true.
type CheckoutResponse struct {
	Orders   []OrderResponse `json:"orders"`
	Currency string          `json:"currency"`
	Replayed bool            `json:"replayed,omitempty"`
}

type OrderResponse struct {
	Code           string             `json:"code"`
	SupplierCode   string             `json:"supplier_code"`
	SupplierName   string             `json:"supplier_name"`
	Status         string             `json:"status"`
	Currency       string             `json:"currency"`
	SubtotalMinor  int64              `json:"subtotal_minor"`
	DiscountsMinor int64              `json:"discounts_minor"`
	TotalMinor     int64              `json:"total_minor"`
	Lines          []OrderLineResponse `json:"lines"`
	PlacedAt       time.Time          `json:"placed_at"`
}

type OrderLineResponse struct {
	Code            string `json:"code"`
	ProductCode     string `json:"product_code"`
	ProductName     string `json:"product_name"`
	Quantity        int    `json:"quantity"`
	UnitCode        string `json:"unit_code"`
	UnitPriceMinor  int64  `json:"unit_price_minor"`
	TotalPriceMinor int64  `json:"total_price_minor"`
	Currency        string `json:"currency"`
}

type OrderHistoryResponse struct {
	FromStatus *string   `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Actor      *int64    `json:"actor"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

type OrderDetailResponse struct {
	OrderResponse
	History []OrderHistoryResponse `json:"history"`
}
