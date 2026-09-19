package service

// Promotions slice 1: code-based vouchers (1 promotion = 1 redeemable
// code). Eligibility = published + time window + budget + once-per-buyer.
// Precedence rule (M4 risk): exactly one voucher per cart — no stacking.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/promotions/repository"
	"github.com/atlas-platform/backend/internal/modules/promotions/schema"
	"github.com/jackc/pgx/v5"
)

var (
	ErrPromotionNotFound = errors.New("promotion not found")
	ErrVoucherUnknown    = errors.New("unknown voucher code")
	ErrVoucherState      = errors.New("voucher is not redeemable")
	ErrVoucherExpired    = errors.New("voucher is outside its validity window")
	ErrVoucherExhausted  = errors.New("voucher budget exhausted")
	ErrVoucherRedeemed   = errors.New("voucher already redeemed by this buyer")
	ErrInvalidPromotion  = errors.New("invalid promotion")
	ErrInvalidTransition = errors.New("invalid promotion status transition")
)

const (
	PromotionDraft     = "draft"
	PromotionPublished = "published"
	PromotionArchived  = "archived"
)

type EventPublisher interface {
	Publish(ctx context.Context, routingKey string, payload interface{}) error
}

type PromotionService struct {
	db        *database.DB
	repo      *repository.PromotionRepository
	publisher EventPublisher
}

func NewPromotionService(db *database.DB, publisher EventPublisher) *PromotionService {
	return &PromotionService{db: db, repo: repository.NewPromotionRepository(db), publisher: publisher}
}

func parseTimePtr(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *PromotionService) CreatePromotion(ctx context.Context, req schema.CreatePromotionRequest) (*schema.PromotionResponse, error) {
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		return nil, ErrInvalidPromotion
	}
	if req.Kind != "percent" && req.Kind != "fixed" {
		return nil, ErrInvalidPromotion
	}
	if req.ValueMinor <= 0 || (req.Kind == "percent" && req.ValueMinor > 100) {
		return nil, ErrInvalidPromotion
	}
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}
	validFrom, err := parseTimePtr(req.ValidFrom)
	if err != nil {
		return nil, ErrInvalidPromotion
	}
	validTo, err := parseTimePtr(req.ValidTo)
	if err != nil {
		return nil, ErrInvalidPromotion
	}
	if validFrom != nil && validTo != nil && validTo.Before(*validFrom) {
		return nil, ErrInvalidPromotion
	}
	if req.MaxRedemptions != nil && *req.MaxRedemptions <= 0 {
		return nil, ErrInvalidPromotion
	}
	p, err := s.repo.CreatePromotion(ctx, req.Code, req.Name, req.Kind, req.ValueMinor, currency, validFrom, validTo, req.MaxRedemptions)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, ErrInvalidPromotion
		}
		return nil, fmt.Errorf("create promotion: %w", err)
	}
	return toPromotionResponse(p, nil), nil
}

func (s *PromotionService) GetPromotion(ctx context.Context, id int64) (*schema.PromotionResponse, error) {
	p, err := s.repo.GetPromotionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPromotionNotFound
		}
		return nil, fmt.Errorf("get promotion: %w", err)
	}
	return toPromotionResponse(p, nil), nil
}

func (s *PromotionService) ListPromotions(ctx context.Context, status string, limit, offset int) ([]schema.PromotionResponse, int, error) {
	promos, err := s.repo.ListPromotions(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list promotions: %w", err)
	}
	total, err := s.repo.CountPromotions(ctx, status)
	if err != nil {
		return nil, 0, fmt.Errorf("count promotions: %w", err)
	}
	out := make([]schema.PromotionResponse, 0, len(promos))
	for _, p := range promos {
		out = append(out, *toPromotionResponse(p, nil))
	}
	return out, total, nil
}

// UpdatePromotion applies partial edits plus forward-only status moves
// (draft→published→archived, or draft→archived directly).
func (s *PromotionService) UpdatePromotion(ctx context.Context, id int64, req schema.UpdatePromotionRequest) (*schema.PromotionResponse, error) {
	current, err := s.repo.GetPromotionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPromotionNotFound
		}
		return nil, fmt.Errorf("get promotion: %w", err)
	}
	params := repository.UpdatePromotionParams{}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return nil, ErrInvalidPromotion
		}
		params.Name = req.Name
	}
	if req.ValueMinor != nil {
		if *req.ValueMinor <= 0 || (current.Kind == "percent" && *req.ValueMinor > 100) {
			return nil, ErrInvalidPromotion
		}
		params.ValueMinor = req.ValueMinor
	}
	if req.ValidFrom != nil || req.ValidTo != nil {
		vf := current.ValidFrom
		vt := current.ValidTo
		if req.ValidFrom != nil {
			parsed, err := parseTimePtr(req.ValidFrom)
			if err != nil {
				return nil, ErrInvalidPromotion
			}
			vf = parsed
			params.ValidFrom = &vf
		}
		if req.ValidTo != nil {
			parsed, err := parseTimePtr(req.ValidTo)
			if err != nil {
				return nil, ErrInvalidPromotion
			}
			vt = parsed
			params.ValidTo = &vt
		}
		if vf != nil && vt != nil && vt.Before(*vf) {
			return nil, ErrInvalidPromotion
		}
	}
	if req.MaxRedemptions != nil {
		// Budgets can be set or raised, never cleared or zeroed; archive
		// the promotion instead. Untouched when absent (nil).
		if *req.MaxRedemptions <= 0 {
			return nil, ErrInvalidPromotion
		}
		params.MaxRedemptions = &req.MaxRedemptions
	}
	if req.Status != nil {
		if !validPromotionTransition(current.Status, *req.Status) {
			return nil, ErrInvalidTransition
		}
		params.Status = req.Status
	}
	updated, err := s.repo.UpdatePromotion(ctx, id, params)
	if err != nil {
		return nil, fmt.Errorf("update promotion: %w", err)
	}
	return toPromotionResponse(updated, nil), nil
}

func validPromotionTransition(from, to string) bool {
	switch from {
	case PromotionDraft:
		return to == PromotionPublished || to == PromotionArchived
	case PromotionPublished:
		return to == PromotionArchived
	default:
		return false
	}
}

// PublishPromotion moves draft → published.
func (s *PromotionService) PublishPromotion(ctx context.Context, id int64) (*schema.PromotionResponse, error) {
	current, err := s.repo.GetPromotionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPromotionNotFound
		}
		return nil, fmt.Errorf("get promotion: %w", err)
	}
	if current.Status != PromotionDraft {
		return nil, ErrInvalidTransition
	}
	to := PromotionPublished
	updated, err := s.repo.UpdatePromotion(ctx, id, repository.UpdatePromotionParams{Status: &to})
	if err != nil {
		return nil, fmt.Errorf("publish promotion: %w", err)
	}
	return toPromotionResponse(updated, nil), nil
}

// ActivePromotions lists published, in-window, budget-available promotions
// with a per-buyer redeemable flag.
func (s *PromotionService) ActivePromotions(ctx context.Context, buyerID int64) ([]schema.PromotionResponse, error) {
	promos, err := s.repo.ListPromotions(ctx, PromotionPublished, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("list promotions: %w", err)
	}
	now := time.Now()
	out := make([]schema.PromotionResponse, 0, len(promos))
	for _, p := range promos {
		if p.ValidFrom != nil && now.Before(*p.ValidFrom) {
			continue
		}
		if p.ValidTo != nil && now.After(*p.ValidTo) {
			continue
		}
		if p.MaxRedemptions != nil && p.RedeemedCount >= *p.MaxRedemptions {
			continue
		}
		redeemed, err := s.repo.HasRedeemed(ctx, buyerID, p.ID)
		if err != nil {
			return nil, fmt.Errorf("redemption check: %w", err)
		}
		redeemable := !redeemed
		out = append(out, *toPromotionResponse(p, &redeemable))
	}
	return out, nil
}

// ValidateVoucher resolves a code to a redeemable promotion or a precise
// rejection. Shared by cart apply, quote, and checkout (single source of
// the eligibility rule).
func (s *PromotionService) ValidateVoucher(ctx context.Context, buyerID int64, code string) (*repository.Promotion, error) {
	if strings.TrimSpace(code) == "" {
		return nil, ErrVoucherUnknown
	}
	p, err := s.repo.GetPromotionByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVoucherUnknown
		}
		return nil, fmt.Errorf("get promotion: %w", err)
	}
	if p.Status != PromotionPublished {
		return nil, ErrVoucherState
	}
	now := time.Now()
	if (p.ValidFrom != nil && now.Before(*p.ValidFrom)) || (p.ValidTo != nil && now.After(*p.ValidTo)) {
		return nil, ErrVoucherExpired
	}
	if p.MaxRedemptions != nil && p.RedeemedCount >= *p.MaxRedemptions {
		return nil, ErrVoucherExhausted
	}
	redeemed, err := s.repo.HasRedeemed(ctx, buyerID, p.ID)
	if err != nil {
		return nil, fmt.Errorf("redemption check: %w", err)
	}
	if redeemed {
		return nil, ErrVoucherRedeemed
	}
	out := p
	return &out, nil
}

// ComputeDiscount returns the discount for a subtotal under a promotion,
// capped at the subtotal so totals never go negative. Integer math only.
func ComputeDiscount(p *repository.Promotion, subtotalMinor int64) int64 {
	if subtotalMinor <= 0 {
		return 0
	}
	var d int64
	switch p.Kind {
	case "percent":
		d = subtotalMinor * p.ValueMinor / 100
	case "fixed":
		d = p.ValueMinor
	default:
		return 0
	}
	if d > subtotalMinor {
		d = subtotalMinor
	}
	if d < 0 {
		return 0
	}
	return d
}

// Report aggregates redeemed counts and total discount cost per promotion.
func (s *PromotionService) Report(ctx context.Context) ([]schema.ReportRow, error) {
	rows, err := s.repo.Report(ctx)
	if err != nil {
		return nil, fmt.Errorf("promotion report: %w", err)
	}
	out := make([]schema.ReportRow, 0, len(rows))
	for _, r := range rows {
		row := schema.ReportRow{
			Code: r.Code, Name: r.Name, Status: r.Status,
			RedeemedCount: r.Redeemed, TotalCostMinor: r.Cost,
		}
		if r.MaxRedemptions != nil {
			remaining := *r.MaxRedemptions - r.Redeemed
			if remaining < 0 {
				remaining = 0
			}
			row.BudgetRemaining = &remaining
		}
		out = append(out, row)
	}
	return out, nil
}

func toPromotionResponse(p repository.Promotion, redeemable *bool) *schema.PromotionResponse {
	return &schema.PromotionResponse{
		Code: p.Code, Name: p.Name, Kind: p.Kind, ValueMinor: p.ValueMinor,
		Currency: p.Currency, Status: p.Status, ValidFrom: p.ValidFrom,
		ValidTo: p.ValidTo, MaxRedemptions: p.MaxRedemptions,
		RedeemedCount: p.RedeemedCount, Redeemable: redeemable,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}
