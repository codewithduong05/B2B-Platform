package schema

import "time"

type SalesReportItem struct {
	Dimension    string `json:"dimension"`
	OrderCount   int    `json:"order_count"`
	RevenueMinor int64  `json:"revenue_minor"`
	Currency     string `json:"currency"`
}

type BuyerReportItem struct {
	BuyerCode       string     `json:"buyer_code"`
	BusinessName    string     `json:"business_name"`
	OrderCount      int        `json:"order_count"`
	TotalSpendMinor int64      `json:"total_spend_minor"`
	LastOrderAt     *time.Time `json:"last_order_at,omitempty"`
	Currency        string     `json:"currency"`
}

type ProductReportItem struct {
	ProductCode  string `json:"product_code"`
	ProductName  string `json:"product_name"`
	OrderCount   int    `json:"order_count"`
	QuantitySold int    `json:"quantity_sold"`
	RevenueMinor int64  `json:"revenue_minor"`
	Currency     string `json:"currency"`
}

type SupplierReportItem struct {
	SupplierCode   string `json:"supplier_code"`
	SupplierName   string `json:"supplier_name"`
	OrderCount     int    `json:"order_count"`
	FulfilledCount int    `json:"fulfilled_count"`
	CancelledCount int    `json:"cancelled_count"`
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

type OrderCounts struct {
	Total      int `json:"total"`
	Placed     int `json:"placed"`
	Confirmed  int `json:"confirmed"`
	Processing int `json:"processing"`
	Shipped    int `json:"shipped"`
	Delivered  int `json:"delivered"`
	Cancelled  int `json:"cancelled"`
	OnHold     int `json:"on_hold"`
}

type ShipmentCounts struct {
	Total     int `json:"total"`
	Preparing int `json:"preparing"`
	Shipped   int `json:"shipped"`
	Delivered int `json:"delivered"`
	Cancelled int `json:"cancelled"`
}

type OperationsReportResponse struct {
	Orders    OrderCounts    `json:"orders"`
	Shipments ShipmentCounts `json:"shipments"`
}

type FinanceReportItem struct {
	Currency                string `json:"currency"`
	TotalInvoicedMinor      int64  `json:"total_invoiced_minor"`
	TotalCollectedMinor     int64  `json:"total_collected_minor"`
	OutstandingBalanceMinor int64  `json:"outstanding_balance_minor"`
	CreditsAppliedMinor     int64  `json:"credits_applied_minor"`
	RefundsMinor            int64  `json:"refunds_minor"`
}

type FinanceReportResponse struct {
	Items []FinanceReportItem `json:"items"`
}

type ExportJobResponse struct {
	Code        string    `json:"code"`
	ReportType  string    `json:"report_type"`
	Status      string    `json:"status"`
	Format      string    `json:"format"`
	DownloadUrl *string   `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
	StatusUrl   string    `json:"status_url,omitempty"`
}

type CreateExportRequest struct {
	ReportType string `json:"report_type"`
	Format     string `json:"format,omitempty"`
	From       string `json:"from,omitempty"`
	To         string `json:"to,omitempty"`
}
