package schema

import (
	"time"
)

type SalesReportItem struct {
	Dimension    string `json:"dimension"`
	OrderCount   int    `json:"order_count"`
	RevenueMinor int64  `json:"revenue_minor"`
	Currency     string `json:"currency"`
}

type BuyerReportItem struct {
	BuyerID      int64  `json:"buyer_id"`
	BusinessName string `json:"business_name"`
	OrderCount   int    `json:"order_count"`
	TotalSpend   int64  `json:"total_spend_minor"`
	Currency     string `json:"currency"`
}

type ProductReportItem struct {
	ProductCode  string `json:"product_code"`
	ProductName  string `json:"product_name"`
	QuantitySold int    `json:"quantity_sold"`
	RevenueMinor int64  `json:"revenue_minor"`
	Currency     string `json:"currency"`
}

type SupplierReportItem struct {
	SupplierID     int64  `json:"supplier_id"`
	CompanyName    string `json:"company_name"`
	OrderCount     int    `json:"order_count"`
	FulfilledCount int    `json:"fulfilled_count"`
	TotalMinor     int64  `json:"total_minor"`
	Currency       string `json:"currency"`
}

type PromotionReportItem struct {
	PromotionCode string `json:"promotion_code"`
	PromotionName string `json:"promotion_name"`
	Redemptions   int    `json:"redemptions"`
	DiscountMinor int64  `json:"discount_minor"`
	Currency      string `json:"currency"`
}

type OperationsReportResponse struct {
	TotalOrders        int     `json:"total_orders"`
	PlacedOrders       int     `json:"placed_orders"`
	ConfirmedOrders    int     `json:"confirmed_orders"`
	ShippedOrders      int     `json:"shipped_orders"`
	DeliveredOrders    int     `json:"delivered_orders"`
	CancelledOrders    int     `json:"cancelled_orders"`
	OnHoldOrders       int     `json:"on_hold_orders"`
	AverageFulfillment float64 `json:"average_fulfillment_hours"`
}

type FinanceReportResponse struct {
	TotalInvoicedMinor  int64  `json:"total_invoiced_minor"`
	TotalCollectedMinor int64  `json:"total_collected_minor"`
	OutstandingBalance  int64  `json:"outstanding_balance_minor"`
	Currency            string `json:"currency"`
}

type ExportJobResponse struct {
	Code        string    `json:"code"`
	ReportType  string    `json:"report_type"`
	Status      string    `json:"status"`
	Format      string    `json:"format"`
	DownloadUrl *string   `json:"download_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateExportRequest struct {
	ReportType string `json:"report_type"`
	Format     string `json:"format,omitempty"`
}
