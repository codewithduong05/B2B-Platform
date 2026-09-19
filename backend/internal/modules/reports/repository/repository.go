package repository

import (
	"context"
	"time"

	"github.com/atlas-platform/backend/internal/database"
)

type ReportsRepository struct {
	db *database.DB
}

func NewReportsRepository(db *database.DB) *ReportsRepository {
	return &ReportsRepository{db: db}
}

// window is a half-open [from, to) filter on a timestamp column. NULL bounds
// mean unbounded; callers pass validated UTC instants.
const windowClause = `($1::timestamptz IS NULL OR o.placed_at >= $1) AND ($2::timestamptz IS NULL OR o.placed_at < $2)`

type SalesRow struct {
	Dimension    string
	OrderCount   int
	RevenueMinor int64
	Currency     string
}

// SalesByPeriod groups orders by UTC calendar month. Orders are counted once;
// no line-level join is involved, so revenue is never multiplied by line count.
func (r *ReportsRepository) SalesByPeriod(ctx context.Context, from, to *time.Time, limit, offset int) ([]SalesRow, error) {
	return r.salesRows(ctx, `
		SELECT TO_CHAR(o.placed_at AT TIME ZONE 'UTC', 'YYYY-MM') AS dimension,
		       COUNT(*)::int,
		       COALESCE(SUM(o.total_minor), 0)::bigint,
		       o.currency
		FROM commerce."order" o
		WHERE o.deleted_at IS NULL AND `+windowClause+`
		GROUP BY 1, o.currency
		ORDER BY 1 ASC, o.currency ASC
		LIMIT $3 OFFSET $4`, from, to, limit, offset)
}

func (r *ReportsRepository) CountSalesByPeriod(ctx context.Context, from, to *time.Time) (int, error) {
	return r.countGroups(ctx, `
		SELECT TO_CHAR(o.placed_at AT TIME ZONE 'UTC', 'YYYY-MM'), o.currency
		FROM commerce."order" o
		WHERE o.deleted_at IS NULL AND `+windowClause+`
		GROUP BY 1, 2`, from, to)
}

func (r *ReportsRepository) SalesBySupplier(ctx context.Context, from, to *time.Time, limit, offset int) ([]SalesRow, error) {
	return r.salesRows(ctx, `
		SELECT COALESCE(cs.name, 'Unknown') AS dimension,
		       COUNT(*)::int,
		       COALESCE(SUM(o.total_minor), 0)::bigint,
		       o.currency
		FROM commerce."order" o
		LEFT JOIN catalog.supplier cs ON cs.id = o.supplier_id
		WHERE o.deleted_at IS NULL AND `+windowClause+`
		GROUP BY 1, o.currency
		ORDER BY COALESCE(SUM(o.total_minor), 0) DESC, 1 ASC, o.currency ASC
		LIMIT $3 OFFSET $4`, from, to, limit, offset)
}

func (r *ReportsRepository) CountSalesBySupplier(ctx context.Context, from, to *time.Time) (int, error) {
	return r.countGroups(ctx, `
		SELECT COALESCE(cs.name, 'Unknown'), o.currency
		FROM commerce."order" o
		LEFT JOIN catalog.supplier cs ON cs.id = o.supplier_id
		WHERE o.deleted_at IS NULL AND `+windowClause+`
		GROUP BY 1, 2`, from, to)
}

// SalesByCategory attributes line revenue to the product's category. A single
// order may span categories, so order_count counts distinct orders while
// revenue sums line totals only — summing order totals here would multiply
// each order by its line count.
func (r *ReportsRepository) SalesByCategory(ctx context.Context, from, to *time.Time, limit, offset int) ([]SalesRow, error) {
	return r.salesRows(ctx, `
		SELECT COALESCE(c.name, 'Uncategorised') AS dimension,
		       COUNT(DISTINCT o.id)::int,
		       COALESCE(SUM(ol.total_price_minor), 0)::bigint,
		       ol.currency
		FROM commerce.order_line ol
		JOIN commerce."order" o ON o.id = ol.order_id AND o.deleted_at IS NULL
		LEFT JOIN catalog.product p ON p.id = ol.product_id
		LEFT JOIN catalog.category c ON c.id = p.category_id
		WHERE ol.deleted_at IS NULL AND `+windowClause+`
		GROUP BY 1, ol.currency
		ORDER BY COALESCE(SUM(ol.total_price_minor), 0) DESC, 1 ASC, ol.currency ASC
		LIMIT $3 OFFSET $4`, from, to, limit, offset)
}

func (r *ReportsRepository) CountSalesByCategory(ctx context.Context, from, to *time.Time) (int, error) {
	return r.countGroups(ctx, `
		SELECT COALESCE(c.name, 'Uncategorised'), ol.currency
		FROM commerce.order_line ol
		JOIN commerce."order" o ON o.id = ol.order_id AND o.deleted_at IS NULL
		LEFT JOIN catalog.product p ON p.id = ol.product_id
		LEFT JOIN catalog.category c ON c.id = p.category_id
		WHERE ol.deleted_at IS NULL AND `+windowClause+`
		GROUP BY 1, 2`, from, to)
}

func (r *ReportsRepository) salesRows(ctx context.Context, query string, from, to *time.Time, limit, offset int) ([]SalesRow, error) {
	rows, err := r.db.Pool.Query(ctx, query, from, to, limit, offset)
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
	return out, rows.Err()
}

// countGroups counts the number of grouped rows a report page can hold so the
// router can compute totals without loading the full result set.
func (r *ReportsRepository) countGroups(ctx context.Context, inner string, from, to *time.Time) (int, error) {
	var total int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM (`+inner+`) groups`, from, to).Scan(&total)
	return total, err
}

type BuyerRow struct {
	BuyerCode       string
	BusinessName    string
	OrderCount      int
	TotalSpendMinor int64
	LastOrderAt     *time.Time
	Currency        string
}

// BuyerReport lists every buyer profile (acquisition view: zero-order buyers
// are included) with lifetime spend inside the window.
func (r *ReportsRepository) BuyerReport(ctx context.Context, from, to *time.Time, limit, offset int) ([]BuyerRow, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT bp.code,
		       bp.business_name,
		       COUNT(o.id)::int,
		       COALESCE(SUM(o.total_minor), 0)::bigint,
		       MAX(o.placed_at),
		       COALESCE(o.currency, bp.currency)
		FROM identity.buyer_profile bp
		LEFT JOIN commerce."order" o ON o.buyer_id = bp.id AND o.deleted_at IS NULL AND `+windowClause+`
		WHERE bp.deleted_at IS NULL
		GROUP BY bp.id, bp.code, bp.business_name, bp.currency, o.currency
		ORDER BY COALESCE(SUM(o.total_minor), 0) DESC, bp.code ASC
		LIMIT $3 OFFSET $4`, from, to, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BuyerRow
	for rows.Next() {
		var br BuyerRow
		if err := rows.Scan(&br.BuyerCode, &br.BusinessName, &br.OrderCount, &br.TotalSpendMinor, &br.LastOrderAt, &br.Currency); err != nil {
			return nil, err
		}
		out = append(out, br)
	}
	return out, rows.Err()
}

func (r *ReportsRepository) CountBuyers(ctx context.Context, from, to *time.Time) (int, error) {
	return r.countGroups(ctx, `
		SELECT bp.id, o.currency
		FROM identity.buyer_profile bp
		LEFT JOIN commerce."order" o ON o.buyer_id = bp.id AND o.deleted_at IS NULL AND `+windowClause+`
		WHERE bp.deleted_at IS NULL
		GROUP BY bp.id, bp.currency, o.currency`, from, to)
}

type ProductRow struct {
	ProductCode  string
	ProductName  string
	OrderCount   int
	QuantitySold int
	RevenueMinor int64
	Currency     string
}

// ProductReport groups sold lines by stable product id; the display code and
// name come from the line's price snapshot (renames across history collapse
// into one row, keeping the lexicographically latest snapshot).
func (r *ReportsRepository) ProductReport(ctx context.Context, from, to *time.Time, limit, offset int) ([]ProductRow, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT MAX(ol.product_code),
		       MAX(ol.product_name),
		       COUNT(DISTINCT ol.order_id)::int,
		       COALESCE(SUM(ol.quantity), 0)::int,
		       COALESCE(SUM(ol.total_price_minor), 0)::bigint,
		       ol.currency
		FROM commerce.order_line ol
		JOIN commerce."order" o ON o.id = ol.order_id AND o.deleted_at IS NULL
		WHERE ol.deleted_at IS NULL AND `+windowClause+`
		GROUP BY ol.product_id, ol.currency
		ORDER BY COALESCE(SUM(ol.total_price_minor), 0) DESC, MAX(ol.product_code) ASC
		LIMIT $3 OFFSET $4`, from, to, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProductRow
	for rows.Next() {
		var pr ProductRow
		if err := rows.Scan(&pr.ProductCode, &pr.ProductName, &pr.OrderCount, &pr.QuantitySold, &pr.RevenueMinor, &pr.Currency); err != nil {
			return nil, err
		}
		out = append(out, pr)
	}
	return out, rows.Err()
}

func (r *ReportsRepository) CountProducts(ctx context.Context, from, to *time.Time) (int, error) {
	return r.countGroups(ctx, `
		SELECT ol.product_id, ol.currency
		FROM commerce.order_line ol
		JOIN commerce."order" o ON o.id = ol.order_id AND o.deleted_at IS NULL
		WHERE ol.deleted_at IS NULL AND `+windowClause+`
		GROUP BY 1, 2`, from, to)
}

type SupplierRow struct {
	SupplierCode   string
	SupplierName   string
	OrderCount     int
	FulfilledCount int
	CancelledCount int
	TotalMinor     int64
	Currency       string
}

// SupplierReport covers suppliers with orders in the window. Orders join to
// catalog.supplier directly (the FK target of commerce."order"), so a supplier
// row can never be duplicated by the profile linkage.
func (r *ReportsRepository) SupplierReport(ctx context.Context, from, to *time.Time, limit, offset int) ([]SupplierRow, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT cs.code,
		       cs.name,
		       COUNT(o.id)::int,
		       COUNT(o.id) FILTER (WHERE o.status IN ('shipped', 'delivered'))::int,
		       COUNT(o.id) FILTER (WHERE o.status = 'cancelled')::int,
		       COALESCE(SUM(o.total_minor), 0)::bigint,
		       o.currency
		FROM catalog.supplier cs
		JOIN commerce."order" o ON o.supplier_id = cs.id AND o.deleted_at IS NULL AND `+windowClause+`
		WHERE cs.deleted_at IS NULL
		GROUP BY cs.id, cs.code, cs.name, o.currency
		ORDER BY COALESCE(SUM(o.total_minor), 0) DESC, cs.code ASC
		LIMIT $3 OFFSET $4`, from, to, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SupplierRow
	for rows.Next() {
		var sr SupplierRow
		if err := rows.Scan(&sr.SupplierCode, &sr.SupplierName, &sr.OrderCount, &sr.FulfilledCount, &sr.CancelledCount, &sr.TotalMinor, &sr.Currency); err != nil {
			return nil, err
		}
		out = append(out, sr)
	}
	return out, rows.Err()
}

func (r *ReportsRepository) CountSuppliers(ctx context.Context, from, to *time.Time) (int, error) {
	return r.countGroups(ctx, `
		SELECT cs.id, o.currency
		FROM catalog.supplier cs
		JOIN commerce."order" o ON o.supplier_id = cs.id AND o.deleted_at IS NULL AND `+windowClause+`
		WHERE cs.deleted_at IS NULL
		GROUP BY 1, 2`, from, to)
}

type PromotionRow struct {
	PromotionCode string
	PromotionName string
	Redemptions   int
	DiscountMinor int64
	Currency      string
}

// PromotionReport attributes redeemed discount value per promotion. The join
// is 1:N (one promotion, many redemptions) and only redemption amounts are
// summed, so totals cannot be inflated by other tables.
func (r *ReportsRepository) PromotionReport(ctx context.Context, from, to *time.Time, limit, offset int) ([]PromotionRow, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT p.code,
		       p.name,
		       COUNT(vr.id)::int,
		       COALESCE(SUM(vr.amount_minor), 0)::bigint,
		       p.currency
		FROM promotions.promotion p
		LEFT JOIN promotions.voucher_redemption vr ON vr.promotion_id = p.id
		     AND ($1::timestamptz IS NULL OR vr.created_at >= $1)
		     AND ($2::timestamptz IS NULL OR vr.created_at < $2)
		WHERE p.deleted_at IS NULL
		GROUP BY p.id, p.code, p.name, p.currency
		ORDER BY COALESCE(SUM(vr.amount_minor), 0) DESC, p.code ASC
		LIMIT $3 OFFSET $4`, from, to, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PromotionRow
	for rows.Next() {
		var pr PromotionRow
		if err := rows.Scan(&pr.PromotionCode, &pr.PromotionName, &pr.Redemptions, &pr.DiscountMinor, &pr.Currency); err != nil {
			return nil, err
		}
		out = append(out, pr)
	}
	return out, rows.Err()
}

func (r *ReportsRepository) CountPromotions(ctx context.Context, from, to *time.Time) (int, error) {
	return r.countGroups(ctx, `
		SELECT p.id
		FROM promotions.promotion p
		LEFT JOIN promotions.voucher_redemption vr ON vr.promotion_id = p.id
		     AND ($1::timestamptz IS NULL OR vr.created_at >= $1)
		     AND ($2::timestamptz IS NULL OR vr.created_at < $2)
		WHERE p.deleted_at IS NULL
		GROUP BY 1`, from, to)
}

type OperationsSummary struct {
	OrderCounts    [8]int
	ShipmentCounts [5]int
}

func (r *ReportsRepository) OperationsReport(ctx context.Context, from, to *time.Time) (OperationsSummary, error) {
	var os OperationsSummary
	if err := r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)::int,
		       COUNT(*) FILTER (WHERE o.status = 'placed')::int,
		       COUNT(*) FILTER (WHERE o.status = 'confirmed')::int,
		       COUNT(*) FILTER (WHERE o.status = 'processing')::int,
		       COUNT(*) FILTER (WHERE o.status = 'shipped')::int,
		       COUNT(*) FILTER (WHERE o.status = 'delivered')::int,
		       COUNT(*) FILTER (WHERE o.status = 'cancelled')::int,
		       COUNT(*) FILTER (WHERE o.on_hold)::int
		FROM commerce."order" o
		WHERE o.deleted_at IS NULL AND `+windowClause+`
	`, from, to).Scan(
		&os.OrderCounts[0], &os.OrderCounts[1], &os.OrderCounts[2], &os.OrderCounts[3],
		&os.OrderCounts[4], &os.OrderCounts[5], &os.OrderCounts[6], &os.OrderCounts[7],
	); err != nil {
		return os, err
	}
	if err := r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)::int,
		       COUNT(*) FILTER (WHERE s.status = 'preparing')::int,
		       COUNT(*) FILTER (WHERE s.status = 'shipped')::int,
		       COUNT(*) FILTER (WHERE s.status = 'delivered')::int,
		       COUNT(*) FILTER (WHERE s.status = 'cancelled')::int
		FROM commerce.shipment s
		WHERE s.deleted_at IS NULL
		  AND ($1::timestamptz IS NULL OR s.created_at >= $1)
		  AND ($2::timestamptz IS NULL OR s.created_at < $2)
	`, from, to).Scan(
		&os.ShipmentCounts[0], &os.ShipmentCounts[1], &os.ShipmentCounts[2],
		&os.ShipmentCounts[3], &os.ShipmentCounts[4],
	); err != nil {
		return os, err
	}
	return os, nil
}

type FinanceAmount struct {
	Source   string
	Currency string
	Amount   int64
}

// FinanceReport derives per-currency totals from the existing financial
// records: issued invoices (invoiced/collected/outstanding), applied credit
// notes, and approved or applied refunds. Draft and void invoices are
// excluded. The window filters each record by its own event timestamp.
func (r *ReportsRepository) FinanceReport(ctx context.Context, from, to *time.Time) ([]FinanceAmount, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT src, currency, SUM(amount)::bigint
		FROM (
			SELECT 'invoiced' AS src, inv.currency, inv.total_minor AS amount
			FROM commerce.invoice inv
			WHERE inv.deleted_at IS NULL AND inv.status = 'issued'
			  AND ($1::timestamptz IS NULL OR inv.issued_at >= $1)
			  AND ($2::timestamptz IS NULL OR inv.issued_at < $2)
			UNION ALL
			SELECT 'collected', inv.currency, inv.total_minor - inv.balance_minor
			FROM commerce.invoice inv
			WHERE inv.deleted_at IS NULL AND inv.status = 'issued'
			  AND ($1::timestamptz IS NULL OR inv.issued_at >= $1)
			  AND ($2::timestamptz IS NULL OR inv.issued_at < $2)
			UNION ALL
			SELECT 'outstanding', inv.currency, inv.balance_minor
			FROM commerce.invoice inv
			WHERE inv.deleted_at IS NULL AND inv.status = 'issued'
			  AND ($1::timestamptz IS NULL OR inv.issued_at >= $1)
			  AND ($2::timestamptz IS NULL OR inv.issued_at < $2)
			UNION ALL
			SELECT 'credits', cn.currency, cn.amount_minor
			FROM commerce.credit_note cn
			WHERE cn.deleted_at IS NULL AND cn.status = 'applied'
			  AND ($1::timestamptz IS NULL OR cn.created_at >= $1)
			  AND ($2::timestamptz IS NULL OR cn.created_at < $2)
			UNION ALL
			SELECT 'refunds', rf.currency, rf.amount_minor
			FROM payments.payment_refund rf
			WHERE rf.deleted_at IS NULL AND rf.status IN ('approved', 'applied')
			  AND ($1::timestamptz IS NULL OR rf.created_at >= $1)
			  AND ($2::timestamptz IS NULL OR rf.created_at < $2)
		) sources
		GROUP BY src, currency
		ORDER BY currency ASC, src ASC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FinanceAmount
	for rows.Next() {
		var fa FinanceAmount
		if err := rows.Scan(&fa.Source, &fa.Currency, &fa.Amount); err != nil {
			return nil, err
		}
		out = append(out, fa)
	}
	return out, rows.Err()
}

type ExportJob struct {
	Code        string
	ReportType  string
	Status      string
	Format      string
	DownloadUrl *string
	CreatedAt   time.Time
}

func (r *ReportsRepository) CreateExportJob(ctx context.Context, code, reportType, format string, parameters *string, createdBy *int64) (ExportJob, error) {
	var ej ExportJob
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO reports.export_job (code, report_type, status, format, parameters, created_by)
		VALUES ($1, $2, 'pending', $3, $4, $5)
		RETURNING code, report_type, status, format, download_url, created_at
	`, code, reportType, format, parameters, createdBy).Scan(
		&ej.Code, &ej.ReportType, &ej.Status, &ej.Format, &ej.DownloadUrl, &ej.CreatedAt)
	return ej, err
}

func (r *ReportsRepository) ListExportJobs(ctx context.Context, status string, limit, offset int) ([]ExportJob, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT code, report_type, status, format, download_url, created_at
		FROM reports.export_job
		WHERE deleted_at IS NULL AND ($1 = '' OR status = $1)
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExportJob
	for rows.Next() {
		var ej ExportJob
		if err := rows.Scan(&ej.Code, &ej.ReportType, &ej.Status, &ej.Format, &ej.DownloadUrl, &ej.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, ej)
	}
	return out, rows.Err()
}

func (r *ReportsRepository) CountExportJobs(ctx context.Context, status string) (int, error) {
	var total int
	err := r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM reports.export_job
		WHERE deleted_at IS NULL AND ($1 = '' OR status = $1)
	`, status).Scan(&total)
	return total, err
}

type ExportJobFull struct {
	ID           int64
	Code         string
	ReportType   string
	Status       string
	Format       string
	Parameters   *string
	DownloadUrl  *string
	FilePath     *string
	FileSize     *int64
	FileChecksum *string
	StartedAt    *time.Time
	CompletedAt  *time.Time
	ErrorMessage *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (r *ReportsRepository) ClaimPendingExportJob(ctx context.Context) (*ExportJobFull, error) {
	var j ExportJobFull
	err := r.db.Pool.QueryRow(ctx, `
		UPDATE reports.export_job
		SET status = 'processing', started_at = NOW(), updated_at = NOW()
		WHERE id = (
			SELECT id FROM reports.export_job
			WHERE status = 'pending' AND deleted_at IS NULL
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, code, report_type, status, format, parameters, download_url,
		          file_path, file_size, file_checksum, started_at, completed_at,
		          error_message, created_at, updated_at
	`).Scan(
		&j.ID, &j.Code, &j.ReportType, &j.Status, &j.Format, &j.Parameters,
		&j.DownloadUrl, &j.FilePath, &j.FileSize, &j.FileChecksum,
		&j.StartedAt, &j.CompletedAt, &j.ErrorMessage, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *ReportsRepository) CompleteExportJob(ctx context.Context, id int64, filePath string, fileSize int64, checksum, downloadURL string) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE reports.export_job
		SET status = 'completed', completed_at = NOW(), updated_at = NOW(),
		    file_path = $2, file_size = $3, file_checksum = $4, download_url = $5
		WHERE id = $1
	`, id, filePath, fileSize, checksum, downloadURL)
	return err
}

func (r *ReportsRepository) FailExportJob(ctx context.Context, id int64, errMsg string) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE reports.export_job
		SET status = 'failed', completed_at = NOW(), updated_at = NOW(), error_message = $2
		WHERE id = $1
	`, id, errMsg)
	return err
}

func (r *ReportsRepository) GetExportJobByCode(ctx context.Context, code string) (*ExportJobFull, error) {
	var j ExportJobFull
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, code, report_type, status, format, parameters, download_url,
		       file_path, file_size, file_checksum, started_at, completed_at,
		       error_message, created_at, updated_at
		FROM reports.export_job
		WHERE code = $1 AND deleted_at IS NULL
	`).Scan(
		&j.ID, &j.Code, &j.ReportType, &j.Status, &j.Format, &j.Parameters,
		&j.DownloadUrl, &j.FilePath, &j.FileSize, &j.FileChecksum,
		&j.StartedAt, &j.CompletedAt, &j.ErrorMessage, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &j, nil
}
