package schema

import (
	"time"
)

type CartResponse struct {
	Code        string              `json:"code"`
	BuyerID     int64               `json:"buyer_id"`
	Currency    string              `json:"currency"`
	VoucherCode *string             `json:"voucher_code,omitempty"`
	Suppliers   []SupplierCartGroup `json:"suppliers"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
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

type ApplyVoucherRequest struct {
	Code string `json:"code"`
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
	SupplierCode   string          `json:"supplier_code"`
	SupplierName   string          `json:"supplier_name"`
	Items          []CartLineQuote `json:"items"`
	SubtotalMinor  int64           `json:"subtotal_minor"`
	DiscountsMinor int64           `json:"discounts_minor,omitempty"`
	Currency       string          `json:"currency"`
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
	Code           string              `json:"code"`
	SupplierCode   string              `json:"supplier_code"`
	SupplierName   string              `json:"supplier_name"`
	Status         string              `json:"status"`
	Currency       string              `json:"currency"`
	SubtotalMinor  int64               `json:"subtotal_minor"`
	DiscountsMinor int64               `json:"discounts_minor"`
	TotalMinor     int64               `json:"total_minor"`
	Lines          []OrderLineResponse `json:"lines"`
	PlacedAt       time.Time           `json:"placed_at"`
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

type AdminOrderDetailResponse struct {
	OrderDetailResponse
	Shipments []ShipmentResponse `json:"shipments"`
}

type ShipmentResponse struct {
	Code         string                 `json:"code"`
	Status       string                 `json:"status"`
	Carrier      string                 `json:"carrier,omitempty"`
	TrackingCode string                 `json:"tracking_code,omitempty"`
	Lines        []ShipmentLineResponse `json:"lines"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type ShipmentLineResponse struct {
	OrderLineID int64  `json:"order_line_id"`
	ProductCode string `json:"product_code,omitempty"`
	ProductName string `json:"product_name,omitempty"`
	Quantity    int    `json:"quantity"`
}

type ShipmentLineInput struct {
	OrderLineID int64 `json:"order_line_id"`
	Quantity    int   `json:"quantity"`
}

type CreateShipmentRequest struct {
	OrderID      int64               `json:"order_id"`
	Lines        []ShipmentLineInput `json:"lines"`
	Carrier      *string             `json:"carrier,omitempty"`
	TrackingCode *string             `json:"tracking_code,omitempty"`
}

type UpdateShipmentRequest struct {
	Carrier      *string `json:"carrier,omitempty"`
	TrackingCode *string `json:"tracking_code,omitempty"`
	Status       string  `json:"status,omitempty"`
}

type InvoiceResponse struct {
	Code          string     `json:"code"`
	Status        string     `json:"status"`
	SubtotalMinor int64      `json:"subtotal_minor"`
	TotalMinor    int64      `json:"total_minor"`
	BalanceMinor  int64      `json:"balance_minor"`
	Currency      string     `json:"currency"`
	IssuedAt      *time.Time `json:"issued_at,omitempty"`
	ReplacesCode  *string    `json:"replaces_code,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type CreateInvoiceRequest struct {
	OrderID int64 `json:"order_id"`
}

type ReturnLineInput struct {
	OrderLineID int64 `json:"order_line_id"`
	Quantity    int   `json:"quantity"`
}

type RequestReturnRequest struct {
	Lines  []ReturnLineInput `json:"lines"`
	Reason string            `json:"reason"`
}

type ReturnLineResponse struct {
	OrderLineID int64 `json:"order_line_id"`
	Quantity    int   `json:"quantity"`
}

type ReturnResponse struct {
	Code        string               `json:"code"`
	OrderID     int64                `json:"order_id"`
	Reason      string               `json:"reason"`
	Status      string               `json:"status"`
	RequestedBy *int64               `json:"requested_by,omitempty"`
	DecidedBy   *int64               `json:"decided_by,omitempty"`
	Lines       []ReturnLineResponse `json:"lines"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

type RejectReturnRequest struct {
	Reason string `json:"reason,omitempty"`
}

type CreditNoteResponse struct {
	Code        string    `json:"code"`
	BatchCode   string    `json:"batch_code"`
	OrderID     int64     `json:"order_id"`
	InvoiceID   *int64    `json:"invoice_id,omitempty"`
	ReturnID    *int64    `json:"return_id,omitempty"`
	AmountMinor int64     `json:"amount_minor"`
	Currency    string    `json:"currency"`
	Reason      string    `json:"reason"`
	Status      string    `json:"status"`
	Actor       *int64    `json:"actor,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateCreditRequest struct {
	OrderID     int64  `json:"order_id"`
	AmountMinor int64  `json:"amount_minor"`
	Reason      string `json:"reason"`
}
