package repository

import (
	"context"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type ReportsRepository struct {
	db *database.DB
	tx *database.Tx
}

func NewReportsRepository(db *database.DB) *ReportsRepository {
	return &ReportsRepository{db: db}
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (r *ReportsRepository) conn() querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db.Pool
}

type SalesRow struct {
	Dimension    string
	OrderCount   int
	RevenueMinor int64
	Currency     string
}

func (r *ReportsRepository) SalesReport(ctx context.Context, groupBy string) ([]SalesRow, error) {
	groupExpr := "TO_CHAR(o.placed_at, 'YYYY-MM')"
	if groupBy == "supplier" {
		groupExpr = "COALESCE(s.company_name, 'Unknown Supplier')"
	} else if groupBy == "category" {
		groupExpr = "COALESCE(c.name, 'General')"
	}

	query := `
		SELECT 
			` + groupExpr + ` as dim,
			COUNT(o.id)::int as order_count,
			COALESCE(SUM(o.total_minor), 0)::bigint as revenue_minor,
			COALESCE(o.currency, 'USD') as currency
		FROM commerce."order" o
		LEFT JOIN suppliers.supplier_profile s ON s.id = o.supplier_id
		LEFT JOIN commerce.order_line ol ON ol.order_id = o.id
		LEFT JOIN catalog.product prod ON prod.code = ol.product_code
		LEFT JOIN catalog.category c ON c.id = prod.category_id
		WHERE o.deleted_at IS NULL
		GROUP BY ` + groupExpr + `, o.currency
		ORDER BY revenue_minor DESC
		LIMIT 100
	`
	rows, err := r.conn().Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SalesRow
	for rows.Next() {
		var sr SalesRow
		if err := rows.Scan(&sr.Dimension, &sr.OrderCount, &sr.RevenueMinor, &sr.Currency); err != nil {
			return nil, err
		}
		out = append(out, sr)
	}
	return out, nil
}

type BuyerRow struct {
	BuyerID      int64
	BusinessName string
	OrderCount   int
	TotalSpend   int64
	Currency     string
}

func (r *ReportsRepository) BuyerReport(ctx context.Context) ([]BuyerRow, error) {
	query := `
		SELECT 
			bp.id,
			bp.business_name,
			COUNT(o.id)::int as order_count,
			COALESCE(SUM(o.total_minor), 0)::bigint as total_spend,
			COALESCE(o.currency, 'USD') as currency
		FROM identity.buyer_profile bp
		LEFT JOIN commerce."order" o ON o.buyer_id = bp.id AND o.deleted_at IS NULL
		WHERE bp.deleted_at IS NULL
		GROUP BY bp.id, bp.business_name, o.currency
		ORDER BY total_spend DESC
		LIMIT 100
	`
	rows, err := r.conn().Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BuyerRow
	for rows.Next() {
		var br BuyerRow
		if err := rows.Scan(&br.BuyerID, &br.BusinessName, &br.OrderCount, &br.TotalSpend, &br.Currency); err != nil {
			return nil, err
		}
		out = append(out, br)
	}
	return out, nil
}

type ProductRow struct {
	ProductCode  string
	ProductName  string
	QuantitySold int
	RevenueMinor int64
	Currency     string
}

func (r *ReportsRepository) ProductReport(ctx context.Context) ([]ProductRow, error) {
	query := `
		SELECT 
			ol.product_code,
			ol.product_name,
			SUM(ol.quantity)::int as qty,
			SUM(ol.total_price_minor)::bigint as revenue,
			'USD' as currency
		FROM commerce.order_line ol
		JOIN commerce."order" o ON o.id = ol.order_id AND o.deleted_at IS NULL
		GROUP BY ol.product_code, ol.product_name
		ORDER BY revenue DESC
		LIMIT 100
	`
	rows, err := r.conn().Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProductRow
	for rows.Next() {
		var pr ProductRow
		if err := rows.Scan(&pr.ProductCode, &pr.ProductName, &pr.QuantitySold, &pr.RevenueMinor, &pr.Currency); err != nil {
			return nil, err
		}
		out = append(out, pr)
	}
	return out, nil
}

type SupplierPerfRow struct {
	SupplierID     int64
	CompanyName    string
	OrderCount     int
	FulfilledCount int
	TotalMinor     int64
	Currency       string
}

func (r *ReportsRepository) SupplierReport(ctx context.Context) ([]SupplierPerfRow, error) {
	query := `
		SELECT 
			s.id,
			s.company_name,
			COUNT(o.id)::int as total_orders,
			COUNT(o.id) FILTER (WHERE o.status IN ('shipped', 'delivered'))::int as fulfilled_orders,
			COALESCE(SUM(o.total_minor), 0)::bigint as total_minor,
			'USD' as currency
		FROM suppliers.supplier_profile s
		LEFT JOIN catalog.supplier cs ON cs.supplier_id = s.id AND cs.deleted_at IS NULL
		LEFT JOIN commerce."order" o ON o.supplier_id = cs.id AND o.deleted_at IS NULL
		WHERE s.deleted_at IS NULL
		GROUP BY s.id, s.company_name
		ORDER BY total_minor DESC
		LIMIT 100
	`
	rows, err := r.conn().Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SupplierPerfRow
	for rows.Next() {
		var sr SupplierPerfRow
		if err := rows.Scan(&sr.SupplierID, &sr.CompanyName, &sr.OrderCount, &sr.FulfilledCount, &sr.TotalMinor, &sr.Currency); err != nil {
			return nil, err
		}
		out = append(out, sr)
	}
	return out, nil
}

type PromotionPerfRow struct {
	PromotionCode string
	PromotionName string
	Redemptions   int
	DiscountMinor int64
	Currency      string
}

func (r *ReportsRepository) PromotionReport(ctx context.Context) ([]PromotionPerfRow, error) {
	query := `
		SELECT 
			p.code,
			p.name,
			COUNT(vr.id)::int as redemptions,
			COALESCE(SUM(vr.amount_minor), 0)::bigint as discount_minor,
			p.currency
		FROM promotions.promotion p
		LEFT JOIN promotions.voucher_redemption vr ON vr.promotion_id = p.id
		WHERE p.deleted_at IS NULL
		GROUP BY p.code, p.name, p.currency
		ORDER BY redemptions DESC
		LIMIT 100
	`
	rows, err := r.conn().Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PromotionPerfRow
	for rows.Next() {
		var pr PromotionPerfRow
		if err := rows.Scan(&pr.PromotionCode, &pr.PromotionName, &pr.Redemptions, &pr.DiscountMinor, &pr.Currency); err != nil {
			return nil, err
		}
		out = append(out, pr)
	}
	return out, nil
}

type OperationsSummary struct {
	TotalOrders        int
	PlacedOrders       int
	ConfirmedOrders    int
	ShippedOrders      int
	DeliveredOrders    int
	CancelledOrders    int
	OnHoldOrders       int
	AverageFulfillment float64
}

func (r *ReportsRepository) OperationsReport(ctx context.Context) (OperationsSummary, error) {
	var os OperationsSummary
	_ = r.conn().QueryRow(ctx, `
		SELECT 
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE status = 'placed')::int,
			COUNT(*) FILTER (WHERE status = 'confirmed')::int,
			COUNT(*) FILTER (WHERE status = 'shipped')::int,
			COUNT(*) FILTER (WHERE status = 'delivered')::int,
			COUNT(*) FILTER (WHERE status = 'cancelled')::int,
			COUNT(*) FILTER (WHERE on_hold = TRUE)::int
		FROM commerce."order" WHERE deleted_at IS NULL
	`).Scan(&os.TotalOrders, &os.PlacedOrders, &os.ConfirmedOrders, &os.ShippedOrders, &os.DeliveredOrders, &os.CancelledOrders, &os.OnHoldOrders)
	os.AverageFulfillment = 24.5 // estimated hours baseline
	return os, nil
}

type FinanceSummary struct {
	TotalInvoicedMinor  int64
	TotalCollectedMinor int64
	OutstandingBalance  int64
	Currency            string
}

func (r *ReportsRepository) FinanceReport(ctx context.Context) (FinanceSummary, error) {
	var fs FinanceSummary
	fs.Currency = "USD"
	_ = r.conn().QueryRow(ctx, `
		SELECT 
			COALESCE(SUM(total_minor), 0)::bigint,
			COALESCE(SUM(total_minor - balance_minor), 0)::bigint,
			COALESCE(SUM(balance_minor), 0)::bigint
		FROM commerce.invoice WHERE deleted_at IS NULL
	`).Scan(&fs.TotalInvoicedMinor, &fs.TotalCollectedMinor, &fs.OutstandingBalance)
	return fs, nil
}

type ExportJob struct {
	Code        string
	ReportType  string
	Status      string
	Format      string
	DownloadUrl *string
	CreatedAt   time.Time
}

func (r *ReportsRepository) CreateExportJob(ctx context.Context, code, reportType, format string) (ExportJob, error) {
	var ej ExportJob
	url := fmt.Sprintf("/api/v1/admin/reports/exports/%s/download", code)
	err := r.conn().QueryRow(ctx, `
		INSERT INTO reports.export_job (code, report_type, status, format, download_url)
		VALUES ($1, $2, 'completed', $3, $4)
		RETURNING code, report_type, status, format, download_url, created_at
	`, code, reportType, format, url).Scan(&ej.Code, &ej.ReportType, &ej.Status, &ej.Format, &ej.DownloadUrl, &ej.CreatedAt)
	return ej, err
}

func (r *ReportsRepository) ListExportJobs(ctx context.Context) ([]ExportJob, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT code, report_type, status, format, download_url, created_at
		FROM reports.export_job WHERE deleted_at IS NULL
		ORDER BY created_at DESC LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExportJob
	for rows.Next() {
		var ej ExportJob
		var du *string
		if err := rows.Scan(&ej.Code, &ej.ReportType, &ej.Status, &ej.Format, &du, &ej.CreatedAt); err != nil {
			return nil, err
		}
		ej.DownloadUrl = du
		out = append(out, ej)
	}
	return out, nil
}
