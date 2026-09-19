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
