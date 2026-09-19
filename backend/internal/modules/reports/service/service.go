package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/reports/repository"
	"github.com/atlas-platform/backend/internal/modules/reports/schema"
)

var ErrInvalidInput = errors.New("invalid report request")

var validReportTypes = map[string]bool{
	"sales":      true,
	"buyers":     true,
	"products":   true,
	"suppliers":  true,
	"promotions": true,
	"operations": true,
	"finance":    true,
}

var validExportStatuses = map[string]bool{
	"pending":    true,
	"processing": true,
	"completed":  true,
	"failed":     true,
}

type ReportsService struct {
	repo *repository.ReportsRepository
}

func NewReportsService(db *database.DB) *ReportsService {
	return &ReportsService{repo: repository.NewReportsRepository(db)}
}

// parseTime accepts an RFC 3339 instant or a date-only value (start of that
// UTC day). endOfDay shifts a date-only bound to the next midnight so a
// date-only `to` includes the whole named day; the window stays half-open.
func parseTime(v string, endOfDay bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02", v); err == nil {
		t = t.UTC()
		if endOfDay {
			t = t.AddDate(0, 0, 1)
		}
		return t, nil
	}
	return time.Time{}, ErrInvalidInput
}

func parseWindow(fromStr, toStr string) (*time.Time, *time.Time, error) {
	var from, to *time.Time
	if fromStr != "" {
		t, err := parseTime(fromStr, false)
		if err != nil {
			return nil, nil, err
		}
		from = &t
	}
	if toStr != "" {
		t, err := parseTime(toStr, true)
		if err != nil {
			return nil, nil, err
		}
		to = &t
	}
	if from != nil && to != nil && !from.Before(*to) {
		return nil, nil, ErrInvalidInput
	}
	return from, to, nil
}

func (s *ReportsService) SalesReport(ctx context.Context, groupBy, fromStr, toStr string, limit, offset int) ([]schema.SalesReportItem, int, error) {
	if groupBy == "" {
		groupBy = "period"
	}
	switch groupBy {
	case "period", "supplier", "category":
	default:
		return nil, 0, ErrInvalidInput
	}
	from, to, err := parseWindow(fromStr, toStr)
	if err != nil {
		return nil, 0, err
	}

	var rows []repository.SalesRow
	var total int
	switch groupBy {
	case "period":
		rows, err = s.repo.SalesByPeriod(ctx, from, to, limit, offset)
		if err == nil {
			total, err = s.repo.CountSalesByPeriod(ctx, from, to)
		}
	case "supplier":
		rows, err = s.repo.SalesBySupplier(ctx, from, to, limit, offset)
		if err == nil {
			total, err = s.repo.CountSalesBySupplier(ctx, from, to)
		}
	case "category":
		rows, err = s.repo.SalesByCategory(ctx, from, to, limit, offset)
		if err == nil {
			total, err = s.repo.CountSalesByCategory(ctx, from, to)
		}
	}
	if err != nil {
		return nil, 0, err
	}

	out := make([]schema.SalesReportItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, schema.SalesReportItem{
			Dimension:    r.Dimension,
			OrderCount:   r.OrderCount,
			RevenueMinor: r.RevenueMinor,
			Currency:     r.Currency,
		})
	}
	return out, total, nil
}

func (s *ReportsService) BuyerReport(ctx context.Context, fromStr, toStr string, limit, offset int) ([]schema.BuyerReportItem, int, error) {
	from, to, err := parseWindow(fromStr, toStr)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.repo.BuyerReport(ctx, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountBuyers(ctx, from, to)
	if err != nil {
		return nil, 0, err
	}

	out := make([]schema.BuyerReportItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, schema.BuyerReportItem{
			BuyerCode:       r.BuyerCode,
			BusinessName:    r.BusinessName,
			OrderCount:      r.OrderCount,
			TotalSpendMinor: r.TotalSpendMinor,
			LastOrderAt:     r.LastOrderAt,
			Currency:        r.Currency,
		})
	}
	return out, total, nil
}

func (s *ReportsService) ProductReport(ctx context.Context, fromStr, toStr string, limit, offset int) ([]schema.ProductReportItem, int, error) {
	from, to, err := parseWindow(fromStr, toStr)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.repo.ProductReport(ctx, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountProducts(ctx, from, to)
	if err != nil {
		return nil, 0, err
	}

	out := make([]schema.ProductReportItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, schema.ProductReportItem{
			ProductCode:  r.ProductCode,
			ProductName:  r.ProductName,
			OrderCount:   r.OrderCount,
			QuantitySold: r.QuantitySold,
			RevenueMinor: r.RevenueMinor,
			Currency:     r.Currency,
		})
	}
	return out, total, nil
}

func (s *ReportsService) SupplierReport(ctx context.Context, fromStr, toStr string, limit, offset int) ([]schema.SupplierReportItem, int, error) {
	from, to, err := parseWindow(fromStr, toStr)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.repo.SupplierReport(ctx, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountSuppliers(ctx, from, to)
	if err != nil {
		return nil, 0, err
	}

	out := make([]schema.SupplierReportItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, schema.SupplierReportItem{
			SupplierCode:   r.SupplierCode,
			SupplierName:   r.SupplierName,
			OrderCount:     r.OrderCount,
			FulfilledCount: r.FulfilledCount,
			CancelledCount: r.CancelledCount,
			TotalMinor:     r.TotalMinor,
			Currency:       r.Currency,
		})
	}
	return out, total, nil
}

func (s *ReportsService) PromotionReport(ctx context.Context, fromStr, toStr string, limit, offset int) ([]schema.PromotionReportItem, int, error) {
	from, to, err := parseWindow(fromStr, toStr)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.repo.PromotionReport(ctx, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountPromotions(ctx, from, to)
	if err != nil {
		return nil, 0, err
	}

	out := make([]schema.PromotionReportItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, schema.PromotionReportItem{
			PromotionCode: r.PromotionCode,
			PromotionName: r.PromotionName,
			Redemptions:   r.Redemptions,
			DiscountMinor: r.DiscountMinor,
			Currency:      r.Currency,
		})
	}
	return out, total, nil
}

func (s *ReportsService) OperationsReport(ctx context.Context, fromStr, toStr string) (*schema.OperationsReportResponse, error) {
	from, to, err := parseWindow(fromStr, toStr)
	if err != nil {
		return nil, err
	}
	os, err := s.repo.OperationsReport(ctx, from, to)
	if err != nil {
		return nil, err
	}
	return &schema.OperationsReportResponse{
		Orders: schema.OrderCounts{
			Total:      os.OrderCounts[0],
			Placed:     os.OrderCounts[1],
			Confirmed:  os.OrderCounts[2],
			Processing: os.OrderCounts[3],
			Shipped:    os.OrderCounts[4],
			Delivered:  os.OrderCounts[5],
			Cancelled:  os.OrderCounts[6],
			OnHold:     os.OrderCounts[7],
		},
		Shipments: schema.ShipmentCounts{
			Total:     os.ShipmentCounts[0],
			Preparing: os.ShipmentCounts[1],
			Shipped:   os.ShipmentCounts[2],
			Delivered: os.ShipmentCounts[3],
			Cancelled: os.ShipmentCounts[4],
		},
	}, nil
}

func (s *ReportsService) FinanceReport(ctx context.Context, fromStr, toStr string) (*schema.FinanceReportResponse, error) {
	from, to, err := parseWindow(fromStr, toStr)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.FinanceReport(ctx, from, to)
	if err != nil {
		return nil, err
	}

	items := []schema.FinanceReportItem{}
	idx := map[string]int{}
	for _, r := range rows {
		i, ok := idx[r.Currency]
		if !ok {
			i = len(items)
			idx[r.Currency] = i
			items = append(items, schema.FinanceReportItem{Currency: r.Currency})
		}
		switch r.Source {
		case "invoiced":
			items[i].TotalInvoicedMinor = r.Amount
		case "collected":
			items[i].TotalCollectedMinor = r.Amount
		case "outstanding":
			items[i].OutstandingBalanceMinor = r.Amount
		case "credits":
			items[i].CreditsAppliedMinor = r.Amount
		case "refunds":
			items[i].RefundsMinor = r.Amount
		}
	}
	return &schema.FinanceReportResponse{Items: items}, nil
}

func (s *ReportsService) ListExportJobs(ctx context.Context, status string, limit, offset int) ([]schema.ExportJobResponse, int, error) {
	if status != "" && !validExportStatuses[status] {
		return nil, 0, ErrInvalidInput
	}
	jobs, err := s.repo.ListExportJobs(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountExportJobs(ctx, status)
	if err != nil {
		return nil, 0, err
	}

	out := make([]schema.ExportJobResponse, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, schema.ExportJobResponse{
			Code:        j.Code,
			ReportType:  j.ReportType,
			Status:      j.Status,
			Format:      j.Format,
			DownloadUrl: j.DownloadUrl,
			CreatedAt:   j.CreatedAt,
		})
	}
	return out, total, nil
}

// ExportStatusUrl is the collection endpoint that exposes job status; the
// contract caps this slice at nine endpoints, so there is no per-job GET.
const ExportStatusUrl = "/api/v1/admin/reports/exports"

func newExportCode() string {
	return fmt.Sprintf("exp_%s%s",
		strconv.FormatInt(time.Now().UnixNano(), 36),
		strconv.FormatUint(uint64(rand.Uint32()), 36),
	)
}

func (s *ReportsService) CreateExportJob(ctx context.Context, req schema.CreateExportRequest, createdBy int64) (*schema.ExportJobResponse, error) {
	reportType := strings.TrimSpace(req.ReportType)
	if !validReportTypes[reportType] {
		return nil, ErrInvalidInput
	}
	format := req.Format
	if format == "" {
		format = "csv"
	}
	if format != "csv" {
		return nil, ErrInvalidInput
	}
	from, to, err := parseWindow(req.From, req.To)
	if err != nil {
		return nil, err
	}

	var parameters *string
	if from != nil || to != nil {
		p := map[string]string{}
		if from != nil {
			p["from"] = from.Format(time.RFC3339)
		}
		if to != nil {
			p["to"] = to.Format(time.RFC3339)
		}
		raw, err := json.Marshal(p)
		if err != nil {
			return nil, err
		}
		s := string(raw)
		parameters = &s
	}

	var actor *int64
	if createdBy > 0 {
		actor = &createdBy
	}
	job, err := s.repo.CreateExportJob(ctx, newExportCode(), reportType, format, parameters, actor)
	if err != nil {
		return nil, err
	}
	return &schema.ExportJobResponse{
		Code:        job.Code,
		ReportType:  job.ReportType,
		Status:      job.Status,
		Format:      job.Format,
		DownloadUrl: job.DownloadUrl,
		CreatedAt:   job.CreatedAt,
	}, nil
}

func (s *ReportsService) GetExportJobByCode(ctx context.Context, code string) (*schema.ExportJobResponse, error) {
	job, err := s.repo.GetExportJobByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return &schema.ExportJobResponse{
		Code:        job.Code,
		ReportType:  job.ReportType,
		Status:      job.Status,
		Format:      job.Format,
		DownloadUrl: job.DownloadUrl,
		CreatedAt:   job.CreatedAt,
	}, nil
}
