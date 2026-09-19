package service

import (
	"context"
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

var (
	ErrNotFound     = errors.New("report resource not found")
	ErrInvalidInput = errors.New("invalid report request")
)

type ReportsService struct {
	db   *database.DB
	repo *repository.ReportsRepository
}

func NewReportsService(db *database.DB) *ReportsService {
	return &ReportsService{db: db, repo: repository.NewReportsRepository(db)}
}

func newCode(prefix string) string {
	return fmt.Sprintf("%s%s%s",
		prefix,
		strconv.FormatInt(time.Now().UnixNano(), 36),
		strconv.FormatUint(uint64(rand.Uint32()), 36),
	)
}

func (s *ReportsService) SalesReport(ctx context.Context, groupBy string) ([]schema.SalesReportItem, error) {
	if groupBy == "" {
		groupBy = "period"
	}
	rows, err := s.repo.SalesReport(ctx, groupBy)
	if err != nil {
		return nil, err
	}
	var out []schema.SalesReportItem
	for _, r := range rows {
		out = append(out, schema.SalesReportItem{
			Dimension:    r.Dimension,
			OrderCount:   r.OrderCount,
			RevenueMinor: r.RevenueMinor,
			Currency:     r.Currency,
		})
	}
	if out == nil {
		out = []schema.SalesReportItem{}
	}
	return out, nil
}

func (s *ReportsService) BuyerReport(ctx context.Context) ([]schema.BuyerReportItem, error) {
	rows, err := s.repo.BuyerReport(ctx)
	if err != nil {
		return nil, err
	}
	var out []schema.BuyerReportItem
	for _, r := range rows {
		out = append(out, schema.BuyerReportItem{
			BuyerID:      r.BuyerID,
			BusinessName: r.BusinessName,
			OrderCount:   r.OrderCount,
			TotalSpend:   r.TotalSpend,
			Currency:     r.Currency,
		})
	}
	if out == nil {
		out = []schema.BuyerReportItem{}
	}
	return out, nil
}

func (s *ReportsService) ProductReport(ctx context.Context) ([]schema.ProductReportItem, error) {
	rows, err := s.repo.ProductReport(ctx)
	if err != nil {
		return nil, err
	}
	var out []schema.ProductReportItem
	for _, r := range rows {
		out = append(out, schema.ProductReportItem{
			ProductCode:  r.ProductCode,
			ProductName:  r.ProductName,
			QuantitySold: r.QuantitySold,
			RevenueMinor: r.RevenueMinor,
			Currency:     r.Currency,
		})
	}
	if out == nil {
		out = []schema.ProductReportItem{}
	}
	return out, nil
}

func (s *ReportsService) SupplierReport(ctx context.Context) ([]schema.SupplierReportItem, error) {
	rows, err := s.repo.SupplierReport(ctx)
	if err != nil {
		return nil, err
	}
	var out []schema.SupplierReportItem
	for _, r := range rows {
		out = append(out, schema.SupplierReportItem{
			SupplierID:     r.SupplierID,
			CompanyName:    r.CompanyName,
			OrderCount:     r.OrderCount,
			FulfilledCount: r.FulfilledCount,
			TotalMinor:     r.TotalMinor,
			Currency:       r.Currency,
		})
	}
	if out == nil {
		out = []schema.SupplierReportItem{}
	}
	return out, nil
}

func (s *ReportsService) PromotionReport(ctx context.Context) ([]schema.PromotionReportItem, error) {
	rows, err := s.repo.PromotionReport(ctx)
	if err != nil {
		return nil, err
	}
	var out []schema.PromotionReportItem
	for _, r := range rows {
		out = append(out, schema.PromotionReportItem{
			PromotionCode: r.PromotionCode,
			PromotionName: r.PromotionName,
			Redemptions:   r.Redemptions,
			DiscountMinor: r.DiscountMinor,
			Currency:      r.Currency,
		})
	}
	if out == nil {
		out = []schema.PromotionReportItem{}
	}
	return out, nil
}

func (s *ReportsService) OperationsReport(ctx context.Context) (*schema.OperationsReportResponse, error) {
	os, err := s.repo.OperationsReport(ctx)
	if err != nil {
		return nil, err
	}
	return &schema.OperationsReportResponse{
		TotalOrders:        os.TotalOrders,
		PlacedOrders:       os.PlacedOrders,
		ConfirmedOrders:    os.ConfirmedOrders,
		ShippedOrders:      os.ShippedOrders,
		DeliveredOrders:    os.DeliveredOrders,
		CancelledOrders:    os.CancelledOrders,
		OnHoldOrders:       os.OnHoldOrders,
		AverageFulfillment: os.AverageFulfillment,
	}, nil
}

func (s *ReportsService) FinanceReport(ctx context.Context) (*schema.FinanceReportResponse, error) {
	fs, err := s.repo.FinanceReport(ctx)
	if err != nil {
		return nil, err
	}
	return &schema.FinanceReportResponse{
		TotalInvoicedMinor:  fs.TotalInvoicedMinor,
		TotalCollectedMinor: fs.TotalCollectedMinor,
		OutstandingBalance:  fs.OutstandingBalance,
		Currency:            fs.Currency,
	}, nil
}

func (s *ReportsService) CreateExportJob(ctx context.Context, req schema.CreateExportRequest) (*schema.ExportJobResponse, error) {
	if strings.TrimSpace(req.ReportType) == "" {
		return nil, ErrInvalidInput
	}
	format := req.Format
	if format == "" {
		format = "csv"
	}
	ej, err := s.repo.CreateExportJob(ctx, newCode("exp_"), req.ReportType, format)
	if err != nil {
		return nil, err
	}
	return &schema.ExportJobResponse{
		Code:        ej.Code,
		ReportType:  ej.ReportType,
		Status:      ej.Status,
		Format:      ej.Format,
		DownloadUrl: ej.DownloadUrl,
		CreatedAt:   ej.CreatedAt,
	}, nil
}

func (s *ReportsService) ListExportJobs(ctx context.Context) ([]schema.ExportJobResponse, error) {
	jobs, err := s.repo.ListExportJobs(ctx)
	if err != nil {
		return nil, err
	}
	var out []schema.ExportJobResponse
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
	if out == nil {
		out = []schema.ExportJobResponse{}
	}
	return out, nil
}
