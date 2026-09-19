package worker

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/atlas-platform/backend/internal/modules/reports/repository"
	"github.com/atlas-platform/backend/internal/storage"
)

type ExportWorker struct {
	repo    *repository.ReportsRepository
	storage storage.Storage
	logger  *slog.Logger
}

func NewExportWorker(repo *repository.ReportsRepository, store storage.Storage, logger *slog.Logger) *ExportWorker {
	return &ExportWorker{repo: repo, storage: store, logger: logger}
}

func (w *ExportWorker) ProcessNext(ctx context.Context) (bool, error) {
	job, err := w.repo.ClaimPendingExportJob(ctx)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("claim job: %w", err)
	}

	w.logger.Info("processing export job", "code", job.Code, "type", job.ReportType)

	if err := w.generateExport(ctx, job); err != nil {
		w.logger.Error("export failed", "code", job.Code, "error", err)
		_ = w.repo.FailExportJob(ctx, job.ID, err.Error())
		return true, err
	}

	w.logger.Info("export completed", "code", job.Code)
	return true, nil
}

func (w *ExportWorker) generateExport(ctx context.Context, job *repository.ExportJobFull) error {
	var data []byte
	var err error

	switch job.ReportType {
	case "sales":
		data, err = w.generateSalesReport(ctx, job)
	case "buyers":
		data, err = w.generateBuyerReport(ctx, job)
	case "products":
		data, err = w.generateProductReport(ctx, job)
	case "suppliers":
		data, err = w.generateSupplierReport(ctx, job)
	case "promotions":
		data, err = w.generatePromotionReport(ctx, job)
	case "operations":
		data, err = w.generateOperationsReport(ctx, job)
	case "finance":
		data, err = w.generateFinanceReport(ctx, job)
	default:
		return fmt.Errorf("unknown report type: %s", job.ReportType)
	}

	if err != nil {
		return err
	}

	key := fmt.Sprintf("exports/%s/%s.%s", job.ReportType, job.Code, job.Format)
	path, size, checksum, err := w.storage.Put(ctx, key, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("store file: %w", err)
	}

	url, err := w.storage.URL(ctx, path, time.Now().Add(24*time.Hour))
	if err != nil {
		return fmt.Errorf("generate URL: %w", err)
	}

	return w.repo.CompleteExportJob(ctx, job.ID, path, size, checksum, url)
}

func (w *ExportWorker) generateSalesReport(ctx context.Context, job *repository.ExportJobFull) ([]byte, error) {
	rows, err := w.repo.SalesByPeriod(ctx, nil, nil, 10000, 0)
	if err != nil {
		return nil, err
	}
	return salesToCSV(rows)
}

func (w *ExportWorker) generateBuyerReport(ctx context.Context, job *repository.ExportJobFull) ([]byte, error) {
	rows, err := w.repo.BuyerReport(ctx, nil, nil, 10000, 0)
	if err != nil {
		return nil, err
	}
	return buyerToCSV(rows)
}

func (w *ExportWorker) generateProductReport(ctx context.Context, job *repository.ExportJobFull) ([]byte, error) {
	rows, err := w.repo.ProductReport(ctx, nil, nil, 10000, 0)
	if err != nil {
		return nil, err
	}
	return productToCSV(rows)
}

func (w *ExportWorker) generateSupplierReport(ctx context.Context, job *repository.ExportJobFull) ([]byte, error) {
	rows, err := w.repo.SupplierReport(ctx, nil, nil, 10000, 0)
	if err != nil {
		return nil, err
	}
	return supplierToCSV(rows)
}

func (w *ExportWorker) generatePromotionReport(ctx context.Context, job *repository.ExportJobFull) ([]byte, error) {
	rows, err := w.repo.PromotionReport(ctx, nil, nil, 10000, 0)
	if err != nil {
		return nil, err
	}
	return promotionToCSV(rows)
}

func (w *ExportWorker) generateOperationsReport(ctx context.Context, job *repository.ExportJobFull) ([]byte, error) {
	resp, err := w.repo.OperationsReport(ctx, nil, nil)
	if err != nil {
		return nil, err
	}
	return operationsToCSV(resp)
}

func (w *ExportWorker) generateFinanceReport(ctx context.Context, job *repository.ExportJobFull) ([]byte, error) {
	resp, err := w.repo.FinanceReport(ctx, nil, nil)
	if err != nil {
		return nil, err
	}
	return financeToCSV(resp)
}

func salesToCSV(rows []repository.SalesRow) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"period", "order_count", "revenue_minor", "currency"})
	for _, r := range rows {
		_ = w.Write([]string{r.Dimension, strconv.Itoa(r.OrderCount), strconv.FormatInt(r.RevenueMinor, 10), r.Currency})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func buyerToCSV(rows []repository.BuyerRow) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"buyer_code", "business_name", "order_count", "total_spent_minor", "currency"})
	for _, r := range rows {
		_ = w.Write([]string{r.BuyerCode, r.BusinessName, strconv.Itoa(r.OrderCount), strconv.FormatInt(r.TotalSpendMinor, 10), r.Currency})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func productToCSV(rows []repository.ProductRow) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"product_code", "product_name", "order_count", "quantity_sold", "revenue_minor", "currency"})
	for _, r := range rows {
		_ = w.Write([]string{r.ProductCode, r.ProductName, strconv.Itoa(r.OrderCount), strconv.Itoa(r.QuantitySold), strconv.FormatInt(r.RevenueMinor, 10), r.Currency})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func supplierToCSV(rows []repository.SupplierRow) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"supplier_code", "supplier_name", "order_count", "fulfilled_count", "cancelled_count", "total_minor", "currency"})
	for _, r := range rows {
		_ = w.Write([]string{r.SupplierCode, r.SupplierName, strconv.Itoa(r.OrderCount), strconv.Itoa(r.FulfilledCount), strconv.Itoa(r.CancelledCount), strconv.FormatInt(r.TotalMinor, 10), r.Currency})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func promotionToCSV(rows []repository.PromotionRow) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"promotion_code", "promotion_name", "redemptions", "discount_minor", "currency"})
	for _, r := range rows {
		_ = w.Write([]string{r.PromotionCode, r.PromotionName, strconv.Itoa(r.Redemptions), strconv.FormatInt(r.DiscountMinor, 10), r.Currency})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func operationsToCSV(resp repository.OperationsSummary) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"metric", "value"})
	_ = w.Write([]string{"total_orders", strconv.Itoa(resp.OrderCounts[0])})
	_ = w.Write([]string{"placed_orders", strconv.Itoa(resp.OrderCounts[1])})
	_ = w.Write([]string{"confirmed_orders", strconv.Itoa(resp.OrderCounts[2])})
	_ = w.Write([]string{"dispatched_orders", strconv.Itoa(resp.OrderCounts[7])})
	w.Flush()
	return buf.Bytes(), w.Error()
}

func financeToCSV(rows []repository.FinanceAmount) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"source", "amount_minor", "currency"})
	for _, r := range rows {
		_ = w.Write([]string{r.Source, strconv.FormatInt(r.Amount, 10), r.Currency})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func parseParameters(params *string) map[string]interface{} {
	if params == nil || *params == "" {
		return nil
	}
	var out map[string]interface{}
	_ = json.Unmarshal([]byte(*params), &out)
	return out
}

var _ = strings.TrimSpace
