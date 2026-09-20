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

type ReleaseLotRequest struct {
	LotID  int64  `json:"lot_id" validate:"required"`
	Reason string `json:"reason" validate:"required"`
}

type LowStockLotSummary struct {
	ID                int64     `json:"id"`
	Code              string    `json:"code"`
	StockLevelID      int64     `json:"stock_level_id"`
	LotNumber         string    `json:"lot_number"`
	ProductID         int64     `json:"product_id"`
	ProductCode       string    `json:"product_code"`
	ProductName       string    `json:"product_name"`
	SupplierID        int64     `json:"supplier_id"`
	SupplierCode      string    `json:"supplier_code"`
	AvailableQuantity int32     `json:"available_quantity"`
	ReservedQuantity  int32     `json:"reserved_quantity"`
	SafetyStock       int32     `json:"safety_stock"`
	Threshold         int32     `json:"threshold"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ExpiringLotSummary struct {
	ID                int64      `json:"id"`
	Code              string     `json:"code"`
	StockLevelID      int64      `json:"stock_level_id"`
	LotNumber         string     `json:"lot_number"`
	ProductID         int64      `json:"product_id"`
	ProductCode       string     `json:"product_code"`
	ProductName       string     `json:"product_name"`
	SupplierID        int64      `json:"supplier_id"`
	SupplierCode      string     `json:"supplier_code"`
	AvailableQuantity int32      `json:"available_quantity"`
	ReservedQuantity  int32      `json:"reserved_quantity"`
	ExpiresAt         time.Time  `json:"expires_at"`
	DaysUntilExpiry   int        `json:"days_until_expiry"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type AdjustStockRequest struct {
	LotID         int64  `json:"lot_id" validate:"required"`
	QuantityDelta int32  `json:"quantity_delta" validate:"required"`
	ReasonCode    string `json:"reason_code" validate:"required"`
	Reason        string `json:"reason" validate:"required"`
}

type StockAdjustmentSummary struct {
	ID               int64      `json:"id"`
	LotID            int64      `json:"lot_id"`
	LotCode          string     `json:"lot_code"`
	LotNumber        string     `json:"lot_number"`
	QuantityDelta    int32      `json:"quantity_delta"`
	PreviousQuantity int32      `json:"previous_quantity"`
	NewQuantity      int32      `json:"new_quantity"`
	ReasonCode       string     `json:"reason_code"`
	Reason           string     `json:"reason"`
	AdjustedBy       int64      `json:"adjusted_by"`
	AdjustedAt       time.Time  `json:"adjusted_at"`
}
