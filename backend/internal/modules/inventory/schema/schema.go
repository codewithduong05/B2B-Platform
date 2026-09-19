package schema

import (
	"time"
)

type StockLevelSummary struct {
	ID                int64     `json:"id"`
	Code              string    `json:"code"`
	ProductID         int64     `json:"product_id"`
	SupplierID        int64     `json:"supplier_id"`
	AvailableQuantity int32     `json:"available_quantity"`
	ReservedQuantity  int32     `json:"reserved_quantity"`
	TotalQuantity     int32     `json:"total_quantity"`
	SafetyStock       int32     `json:"safety_stock"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type LotSummary struct {
	ID                int64      `json:"id"`
	Code              string     `json:"code"`
	StockLevelID      int64      `json:"stock_level_id"`
	LotNumber         string     `json:"lot_number"`
	InitialQuantity   int32      `json:"initial_quantity"`
	AvailableQuantity int32      `json:"available_quantity"`
	ReservedQuantity  int32      `json:"reserved_quantity"`
	Status            string     `json:"status"`
	IsQuarantined     bool       `json:"is_quarantined"`
	ProductionDate    *time.Time `json:"production_date,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type ReservationSummary struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	LotID       int64     `json:"lot_id"`
	OrderLineID *int64    `json:"order_line_id,omitempty"`
	RequestID   string    `json:"request_id"`
	Quantity    int32     `json:"quantity"`
	Status      string    `json:"status"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ReservationResponse struct {
	RequestID   string               `json:"request_id"`
	Status      string               `json:"status"`
	Quantity    int32                `json:"quantity"`
	ExpiresAt   time.Time            `json:"expires_at"`
	Allocations []ReservationSummary `json:"allocations"`
}

type ReserveStockRequest struct {
	ProductID   int64  `json:"product_id" validate:"required"`
	SupplierID  int64  `json:"supplier_id" validate:"required"`
	Quantity    int32  `json:"quantity" validate:"required,gte=1"`
	RequestID   string `json:"request_id" validate:"required"`
	OrderLineID *int64 `json:"order_line_id,omitempty"`
	ExpiresIn   int    `json:"expires_in_minutes,omitempty"`
}

type QuarantineLotRequest struct {
	LotID  int64  `json:"lot_id" validate:"required"`
	Reason string `json:"reason" validate:"required"`
}
