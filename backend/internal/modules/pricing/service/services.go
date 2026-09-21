package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/database/queries/pricing"
	"github.com/atlas-platform/backend/internal/modules/pricing/repository"
	"github.com/atlas-platform/backend/internal/modules/pricing/schema"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrPriceListNotFound     = errors.New("price list not found")
	ErrPriceListItemNotFound = errors.New("price list item not found")
	ErrInvalidPriceList      = errors.New("invalid price list")
	ErrInvalidPriceItem      = errors.New("invalid price item")
	ErrInvalidQuantityTier   = errors.New("invalid quantity tier")
	ErrInvalidAssignment     = errors.New("invalid assignment")
	ErrInvalidCurrency       = errors.New("invalid currency")
	ErrPriceNotFound         = errors.New("price not found")
	ErrNoApplicablePrice     = errors.New("no applicable price found")
	ErrDuplicateCode         = errors.New("code already exists")
	ErrInvalidEffectiveDates = errors.New("invalid effective dates")
	ErrInvalidQuantity       = errors.New("invalid quantity")
)

type PriceListService struct {
	priceListRepo    *repository.PriceListRepository
	priceItemRepo    *repository.PriceListItemRepository
	quantityTierRepo *repository.QuantityTierRepository
	assignmentRepo   *repository.PriceListAssignmentRepository
}

func NewPriceListService() *PriceListService {
	return &PriceListService{}
}

func (s *PriceListService) SetDependencies(
	priceListRepo *repository.PriceListRepository,
	priceItemRepo *repository.PriceListItemRepository,
	quantityTierRepo *repository.QuantityTierRepository,
	assignmentRepo *repository.PriceListAssignmentRepository,
) {
	s.priceListRepo = priceListRepo
	s.priceItemRepo = priceItemRepo
	s.quantityTierRepo = quantityTierRepo
	s.assignmentRepo = assignmentRepo
}

func (s *PriceListService) CreatePriceList(ctx context.Context, req schema.CreatePriceListRequest) (*schema.PriceListSummary, error) {
	if req.EffectiveTo != "" {
		from, err := time.Parse(time.RFC3339, req.EffectiveFrom)
		if err != nil {
			return nil, fmt.Errorf("invalid effective_from: %w", err)
		}
		to, err := time.Parse(time.RFC3339, req.EffectiveTo)
		if err != nil {
			return nil, fmt.Errorf("invalid effective_to: %w", err)
		}
		if to.Before(from) {
			return nil, ErrInvalidEffectiveDates
		}
	}

	existing, err := s.priceListRepo.GetPriceListByCode(ctx, req.Code)
	if err == nil && existing.ID != 0 {
		return nil, ErrDuplicateCode
	}

	var effectiveFrom time.Time
	if req.EffectiveFrom != "" {
		effectiveFrom, _ = time.Parse(time.RFC3339, req.EffectiveFrom)
	} else {
		effectiveFrom = time.Now()
	}

	var effectiveTo pgtype.Timestamptz
	if req.EffectiveTo != "" {
		if t, err := time.Parse(time.RFC3339, req.EffectiveTo); err == nil {
			effectiveTo = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	params := pricing.CreatePriceListParams{
		Code:          req.Code,
		Name:          req.Name,
		Description:   pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Status:        pricing.PricingPriceListStatus(req.Status),
		PriceType:     pricing.PricingPriceType(req.PriceType),
		Currency:      req.Currency,
		EffectiveFrom: effectiveFrom,
		EffectiveTo:   effectiveTo,
		IsActive:      req.IsActive,
	}

	row, err := s.priceListRepo.CreatePriceList(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create price list: %w", err)
	}

	pl, err := s.priceListRepo.GetPriceListByID(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("get created price list: %w", err)
	}

	return s.toPriceListSummary(pl), nil
}

func (s *PriceListService) GetPriceList(ctx context.Context, id string) (*schema.PriceListDetail, error) {
	var pl pricing.PricingPriceList
	var err error

	if idInt, parseErr := strconv.ParseInt(id, 10, 64); parseErr == nil {
		pl, err = s.priceListRepo.GetPriceListByID(ctx, idInt)
	} else {
		pl, err = s.priceListRepo.GetPriceListByCode(ctx, id)
	}

	if err != nil {
		return nil, ErrPriceListNotFound
	}

	items, err := s.priceItemRepo.GetPriceListItemsByList(ctx, pl.ID)
	if err != nil {
		return nil, fmt.Errorf("get price list items: %w", err)
	}

	var itemSummaries []schema.PriceListItemSummary
	for _, item := range items {
		itemSummaries = append(itemSummaries, s.toPriceListItemSummary(item))
	}

	return &schema.PriceListDetail{
		PriceListSummary: *s.toPriceListSummary(pl),
		Items:            itemSummaries,
	}, nil
}

func (s *PriceListService) ListPriceLists(ctx context.Context, req schema.PriceListListRequest) (*schema.PriceListListResponse, error) {
	limit := int32(req.PageSize)
	offset := int32((req.Page - 1) * req.PageSize)

	var status *pricing.PricingPriceListStatus
	if req.Status != nil {
		s := pricing.PricingPriceListStatus(*req.Status)
		status = &s
	}

	filters := repository.PriceListFilters{
		Status:   status,
		IsActive: req.IsActive,
	}

	lists, err := s.priceListRepo.ListPriceLists(ctx, limit, offset, filters)
	if err != nil {
		return nil, fmt.Errorf("list price lists: %w", err)
	}

	total, err := s.priceListRepo.CountPriceLists(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("count price lists: %w", err)
	}

	var resp []schema.PriceListSummary
	for _, pl := range lists {
		resp = append(resp, *s.toPriceListSummary(pl))
	}

	return &schema.PriceListListResponse{
		Items:    resp,
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
		HasNext:  int64(req.Page*req.PageSize) < total,
	}, nil
}

func (s *PriceListService) UpdatePriceList(ctx context.Context, id string, req schema.UpdatePriceListRequest) (*schema.PriceListSummary, error) {
	var pl pricing.PricingPriceList
	var err error

	if idInt, parseErr := strconv.ParseInt(id, 10, 64); parseErr == nil {
		pl, err = s.priceListRepo.GetPriceListByID(ctx, idInt)
	} else {
		pl, err = s.priceListRepo.GetPriceListByCode(ctx, id)
	}

	if err != nil {
		return nil, ErrPriceListNotFound
	}

	params := pricing.UpdatePriceListParams{ID: pl.ID}

	if req.Name != nil {
		params.Name = *req.Name
	}
	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.Status != nil {
		params.Status = pricing.PricingPriceListStatus(*req.Status)
	}
	if req.PriceType != nil {
		params.PriceType = pricing.PricingPriceType(*req.PriceType)
	}
	if req.Currency != nil {
		params.Currency = *req.Currency
	}
	if req.EffectiveFrom != nil {
		if t, err := time.Parse(time.RFC3339, *req.EffectiveFrom); err == nil {
			params.EffectiveFrom = t
		}
	}
	if req.EffectiveTo != nil {
		if t, err := time.Parse(time.RFC3339, *req.EffectiveTo); err == nil {
			params.EffectiveTo = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}
	if req.IsActive != nil {
		params.IsActive = *req.IsActive
	}

	updated, err := s.priceListRepo.UpdatePriceList(ctx, pl.ID, params)
	if err != nil {
		return nil, fmt.Errorf("update price list: %w", err)
	}

	updatedPL, err := s.priceListRepo.GetPriceListByID(ctx, updated.ID)
	if err != nil {
		return nil, fmt.Errorf("get updated price list: %w", err)
	}

	return s.toPriceListSummary(updatedPL), nil
}

func (s *PriceListService) DeletePriceList(ctx context.Context, id string) error {
	var pl pricing.PricingPriceList
	var err error

	if idInt, parseErr := strconv.ParseInt(id, 10, 64); parseErr == nil {
		pl, err = s.priceListRepo.GetPriceListByID(ctx, idInt)
	} else {
		pl, err = s.priceListRepo.GetPriceListByCode(ctx, id)
	}

	if err != nil {
		return ErrPriceListNotFound
	}

	return s.priceListRepo.SoftDeletePriceList(ctx, pl.ID)
}

func (s *PriceListService) ReplacePriceListEntries(ctx context.Context, priceListID int64, req schema.ReplacePriceListEntriesRequest) ([]schema.PriceListItemSummary, error) {
	_, err := s.priceListRepo.GetPriceListByID(ctx, priceListID)
	if err != nil {
		return nil, ErrPriceListNotFound
	}

	return nil, fmt.Errorf("replace entries not yet implemented")
}

func (s *PriceListService) CreatePriceListItem(ctx context.Context, priceListID int64, req schema.CreatePriceListItemRequest) (*schema.PriceListItemSummary, error) {
	pl, err := s.priceListRepo.GetPriceListByID(ctx, priceListID)
	if err != nil {
		return nil, ErrPriceListNotFound
	}

	if !pl.IsActive {
		return nil, ErrInvalidPriceList
	}

	var effectiveFrom time.Time
	if req.EffectiveFrom != "" {
		effectiveFrom, _ = time.Parse(time.RFC3339, req.EffectiveFrom)
	} else {
		effectiveFrom = time.Now()
	}

	var effectiveTo pgtype.Timestamptz
	if req.EffectiveTo != "" {
		if t, err := time.Parse(time.RFC3339, req.EffectiveTo); err == nil {
			effectiveTo = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	params := pricing.CreatePriceListItemParams{
		PriceListID:   priceListID,
		ProductID:     req.ProductID,
		UnitID:        req.UnitID,
		Currency:      req.Currency,
		PriceMinor:    req.PriceMinor,
		MinQuantity:   int32(req.MinQuantity),
		MaxQuantity:   pgtype.Int4{Int32: int32(*req.MaxQuantity), Valid: req.MaxQuantity != nil},
		EffectiveFrom: effectiveFrom,
		EffectiveTo:   effectiveTo,
		IsActive:      req.IsActive,
	}

	row, err := s.priceItemRepo.CreatePriceListItem(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create price list item: %w", err)
	}

	item, err := s.priceItemRepo.GetPriceListItemByID(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("get created price list item: %w", err)
	}

	summary := s.toPriceListItemSummary(item)
	return &summary, nil
}

func (s *PriceListService) GetPriceListItems(ctx context.Context, priceListID int64) ([]schema.PriceListItemSummary, error) {
	items, err := s.priceItemRepo.GetPriceListItemsByList(ctx, priceListID)
	if err != nil {
		return nil, fmt.Errorf("get price list items: %w", err)
	}

	var resp []schema.PriceListItemSummary
	for _, item := range items {
		resp = append(resp, s.toPriceListItemSummary(item))
	}

	return resp, nil
}

func (s *PriceListService) GetPriceForProduct(ctx context.Context, priceListID, productID, unitID, quantity int64) (*schema.PriceListItemSummary, error) {
	items, err := s.priceItemRepo.GetPriceForProduct(ctx, priceListID, productID, unitID)
	if err != nil {
		return nil, fmt.Errorf("get price for product: %w", err)
	}

	if len(items) == 0 {
		return nil, ErrNoApplicablePrice
	}

	var bestMatch *pricing.GetPriceForProductRow
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if quantity >= int64(item.MinQuantity) {
			if item.MaxQuantity.Valid && quantity > int64(item.MaxQuantity.Int32) {
				continue
			}
			bestMatch = &item
			break
		}
	}

	if bestMatch == nil {
		return nil, ErrNoApplicablePrice
	}

	if bestMatch.TierID.Valid {
		tiers, err := s.quantityTierRepo.GetQuantityTiersByItem(ctx, bestMatch.ID)
		if err == nil && len(tiers) > 0 {
			for _, tier := range tiers {
				if quantity >= int64(tier.MinQuantity) {
					if tier.MaxQuantity.Valid && quantity > int64(tier.MaxQuantity.Int32) {
						continue
					}
					return &schema.PriceListItemSummary{
						ID:            tier.ID,
						PriceMinor:    tier.PriceMinor,
						MinQuantity:   int(tier.MinQuantity),
						MaxQuantity:   intPtr(int(tier.MaxQuantity.Int32)),
						EffectiveFrom: tier.CreatedAt,
					}, nil
				}
			}
		}
	}

	return s.toPriceListItemSummaryFromRow(bestMatch), nil
}

func (s *PriceListService) GetPriceForQuote(ctx context.Context, priceListID, productID, unitID, quantity int64) (*schema.PriceQuoteLineResponse, error) {
	items, err := s.priceItemRepo.GetPriceForQuote(ctx, priceListID, productID, unitID)
	if err != nil {
		return nil, fmt.Errorf("get price for quote: %w", err)
	}

	if len(items) == 0 {
		return nil, ErrNoApplicablePrice
	}

	var bestMatch *pricing.GetPriceForQuoteRow
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if quantity >= int64(item.MinQuantity) {
			if item.MaxQuantity.Valid && quantity > int64(item.MaxQuantity.Int32) {
				continue
			}
			bestMatch = &item
			break
		}
	}

	if bestMatch == nil {
		return nil, ErrNoApplicablePrice
	}

	unitPrice := bestMatch.PriceMinor
	var tierID *int64

	if bestMatch.TierID.Valid {
		tiers, err := s.quantityTierRepo.GetQuantityTiersByItem(ctx, bestMatch.ID)
		if err == nil && len(tiers) > 0 {
			for _, tier := range tiers {
				if quantity >= int64(tier.MinQuantity) {
					if tier.MaxQuantity.Valid && quantity > int64(tier.MaxQuantity.Int32) {
						continue
					}
					unitPrice = tier.PriceMinor
					tierID = &tier.ID
					break
				}
			}
		}
	}

	return &schema.PriceQuoteLineResponse{
		ProductID:       productID,
		UnitID:          unitID,
		Quantity:        int(quantity),
		UnitPriceMinor:  unitPrice,
		TotalPriceMinor: unitPrice * quantity,
		Currency:        bestMatch.Currency,
		PriceListID:     priceListID,
		PriceListItemID: bestMatch.ID,
		PriceType:       "standard",
		TierID:          tierID,
	}, nil
}

func (s *PriceListService) Quote(ctx context.Context, req schema.PriceQuoteRequest) (*schema.PriceQuoteResponse, error) {
	var responses []schema.PriceQuoteLineResponse

	for _, line := range req.Lines {
		priceListID := int64(0)
		if line.PriceListID != nil {
			priceListID = *line.PriceListID
		} else if line.BuyerProfileID != nil {
			assignment, err := s.assignmentRepo.GetActiveAssignmentForBuyer(ctx, *line.BuyerProfileID)
			if err == nil {
				priceListID = assignment.PriceListID
			}
		}

		if priceListID == 0 {
			lists, err := s.priceListRepo.GetActivePriceLists(ctx)
			if err != nil || len(lists) == 0 {
				return nil, ErrNoApplicablePrice
			}
			priceListID = lists[0].ID
		}

		quote, err := s.GetPriceForQuote(ctx, priceListID, line.ProductID, line.UnitID, int64(line.Quantity))
		if err != nil {
			return nil, fmt.Errorf("quote line for product %d: %w", line.ProductID, err)
		}
		responses = append(responses, *quote)
	}

	return &schema.PriceQuoteResponse{Lines: responses}, nil
}

func (s *PriceListService) CreatePriceListAssignment(ctx context.Context, req schema.CreatePriceListAssignmentRequest) (*schema.PriceListAssignmentSummary, error) {
	pl, err := s.priceListRepo.GetPriceListByID(ctx, req.PriceListID)
	if err != nil {
		return nil, ErrPriceListNotFound
	}

	if !pl.IsActive || pl.Status != pricing.PricingPriceListStatusActive {
		return nil, ErrInvalidPriceList
	}

	params := pricing.CreatePriceListAssignmentParams{
		Code:           generateCode(),
		PriceListID:    req.PriceListID,
		BuyerProfileID: req.BuyerProfileID,
		AssignedBy:     pgtype.Int8{Int64: 0, Valid: false},
		EffectiveFrom:  time.Now(),
		EffectiveTo:    pgtype.Timestamptz{},
		IsActive:       req.IsActive,
	}

	if req.EffectiveFrom != "" {
		if t, err := time.Parse(time.RFC3339, req.EffectiveFrom); err == nil {
			params.EffectiveFrom = t
		}
	}
	if req.EffectiveTo != "" {
		if t, err := time.Parse(time.RFC3339, req.EffectiveTo); err == nil {
			params.EffectiveTo = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	row, err := s.assignmentRepo.CreatePriceListAssignment(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create assignment: %w", err)
	}

	assignment, err := s.assignmentRepo.GetPriceListAssignmentByID(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("get created assignment: %w", err)
	}

	return s.toPriceListAssignmentSummary(assignment), nil
}

func (s *PriceListService) ListPriceListAssignments(ctx context.Context, req schema.PriceListAssignmentListRequest) (*schema.PriceListAssignmentListResponse, error) {
	limit := int32(req.PageSize)
	offset := int32((req.Page - 1) * req.PageSize)

	var priceListID *int64
	var buyerProfileID *int64
	var isActive *bool

	if req.PriceListID != nil {
		priceListID = req.PriceListID
	}
	if req.BuyerID != nil {
		buyerProfileID = req.BuyerID
	}
	if req.IsActive != nil {
		isActive = req.IsActive
	}

	filters := repository.PriceListAssignmentFilters{
		PriceListID:    priceListID,
		BuyerProfileID: buyerProfileID,
		IsActive:       isActive,
	}

	assignments, err := s.assignmentRepo.ListPriceListAssignments(ctx, limit, offset, filters)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}

	total, err := s.assignmentRepo.CountPriceListAssignments(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("count assignments: %w", err)
	}

	var resp []schema.PriceListAssignmentSummary
	for _, a := range assignments {
		resp = append(resp, *s.toPriceListAssignmentSummary(a))
	}

	return &schema.PriceListAssignmentListResponse{
		Items:    resp,
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
		HasNext:  int64(req.Page*req.PageSize) < total,
	}, nil
}

func (s *PriceListService) GetActiveAssignmentForBuyer(ctx context.Context, buyerProfileID int64) (*schema.PriceListAssignmentSummary, error) {
	assignment, err := s.assignmentRepo.GetActiveAssignmentForBuyer(ctx, buyerProfileID)
	if err != nil {
		return nil, ErrInvalidAssignment
	}
	return s.toPriceListAssignmentSummary(assignment), nil
}

func (s *PriceListService) GetProductPricingTiers(ctx context.Context, productID, unitID int64) (*schema.ProductPricingTiersResponse, error) {
	// Find active price lists
	plists, err := s.priceListRepo.GetActivePriceLists(ctx)
	if err != nil {
		return nil, fmt.Errorf("get active price lists: %w", err)
	}

	if len(plists) == 0 {
		return &schema.ProductPricingTiersResponse{
			ProductID: productID,
			UnitID:    unitID,
			BasePrice: 0,
			Currency:  "USD",
			Tiers:     []schema.ProductPricingTier{},
		}, nil
	}

	pl := plists[0]
	priceListID := pl.ID

	items, err := s.priceItemRepo.GetPriceForProduct(ctx, priceListID, productID, unitID)
	if err != nil {
		return nil, fmt.Errorf("get price for product: %w", err)
	}

	if len(items) == 0 {
		return &schema.ProductPricingTiersResponse{
			ProductID: productID,
			UnitID:    unitID,
			BasePrice: 0,
			Currency:  pl.Currency,
			Tiers:     []schema.ProductPricingTier{},
		}, nil
	}

	tiers := make([]schema.ProductPricingTier, 0, len(items))
	for _, item := range items {
		qtiers, err := s.quantityTierRepo.GetQuantityTiersByItem(ctx, item.ID)
		if err != nil || len(qtiers) == 0 {
			tiers = append(tiers, schema.ProductPricingTier{
				MinQuantity: int(item.MinQuantity),
				MaxQuantity: intPtrFromPgtype(item.MaxQuantity),
				PriceMinor:  item.PriceMinor,
				Currency:    pl.Currency,
			})
		} else {
			for _, qt := range qtiers {
				tiers = append(tiers, schema.ProductPricingTier{
					MinQuantity: int(qt.MinQuantity),
					MaxQuantity: intPtrFromPgtype(qt.MaxQuantity),
					PriceMinor:  qt.PriceMinor,
					Currency:    pl.Currency,
				})
			}
		}
	}

	return &schema.ProductPricingTiersResponse{
		ProductID: productID,
		UnitID:    unitID,
		BasePrice: items[0].PriceMinor,
		Currency:  pl.Currency,
		Tiers:     tiers,
	}, nil
}

func intPtrFromPgtype(v pgtype.Int4) *int {
	if v.Valid {
		val := int(v.Int32)
		return &val
	}
	return nil
}

func getPriceForQuoteParams(priceListID, productID, unitID int64) pricing.GetPriceForQuoteParams {
	return pricing.GetPriceForQuoteParams{
		PriceListID: priceListID,
		ProductID:   productID,
		UnitID:      unitID,
	}
}

func getPriceForProductParams(priceListID, productID, unitID int64) pricing.GetPriceForProductParams {
	return pricing.GetPriceForProductParams{
		PriceListID: priceListID,
		ProductID:   productID,
		UnitID:      unitID,
	}
}

func (s *PriceListService) toPriceListSummary(pl pricing.PricingPriceList) *schema.PriceListSummary {
	var effectiveTo *time.Time
	if pl.EffectiveTo.Valid {
		effectiveTo = &pl.EffectiveTo.Time
	}

	return &schema.PriceListSummary{
		Code:          pl.Code,
		Name:          pl.Name,
		Description:   pl.Description.String,
		Status:        string(pl.Status),
		PriceType:     string(pl.PriceType),
		Currency:      pl.Currency,
		EffectiveFrom: pl.EffectiveFrom,
		EffectiveTo:   effectiveTo,
		IsActive:      pl.IsActive,
		CreatedAt:     pl.CreatedAt,
		UpdatedAt:     pl.UpdatedAt,
	}
}

func (s *PriceListService) toPriceListItemSummary(item pricing.PricingPriceListItem) schema.PriceListItemSummary {
	var maxQty *int
	if item.MaxQuantity.Valid {
		v := int(item.MaxQuantity.Int32)
		maxQty = &v
	}

	var effectiveTo *time.Time
	if item.EffectiveTo.Valid {
		effectiveTo = &item.EffectiveTo.Time
	}

	return schema.PriceListItemSummary{
		ID:            item.ID,
		ProductID:     item.ProductID,
		UnitID:        item.UnitID,
		Currency:      item.Currency,
		PriceMinor:    item.PriceMinor,
		MinQuantity:   int(item.MinQuantity),
		MaxQuantity:   maxQty,
		EffectiveFrom: item.EffectiveFrom,
		EffectiveTo:   effectiveTo,
		IsActive:      item.IsActive,
	}
}

func (s *PriceListService) toPriceListItemSummaryFromRow(row *pricing.GetPriceForProductRow) *schema.PriceListItemSummary {
	var maxQty *int
	if row.MaxQuantity.Valid {
		v := int(row.MaxQuantity.Int32)
		maxQty = &v
	}

	var effectiveTo *time.Time
	if row.EffectiveTo.Valid {
		effectiveTo = &row.EffectiveTo.Time
	}

	return &schema.PriceListItemSummary{
		ID:            row.ID,
		ProductID:     row.ProductID,
		UnitID:        row.UnitID,
		Currency:      row.Currency,
		PriceMinor:    row.PriceMinor,
		MinQuantity:   int(row.MinQuantity),
		MaxQuantity:   maxQty,
		EffectiveFrom: row.EffectiveFrom,
		EffectiveTo:   effectiveTo,
		IsActive:      row.IsActive,
	}
}

func (s *PriceListService) toPriceListAssignmentSummary(a pricing.PricingPriceListAssignment) *schema.PriceListAssignmentSummary {
	var effectiveTo *time.Time
	if a.EffectiveTo.Valid {
		effectiveTo = &a.EffectiveTo.Time
	}

	var assignedBy int64
	if a.AssignedBy.Valid {
		assignedBy = a.AssignedBy.Int64
	}

	return &schema.PriceListAssignmentSummary{
		Code:           a.Code,
		PriceListID:    a.PriceListID,
		BuyerProfileID: a.BuyerProfileID,
		AssignedBy:     assignedBy,
		EffectiveFrom:  a.EffectiveFrom,
		EffectiveTo:    effectiveTo,
		IsActive:       a.IsActive,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}

func generateCode() string {
	return fmt.Sprintf("pl_%d", time.Now().UnixNano())
}

func intPtr(i int) *int {
	return &i
}

type Services struct {
	PriceList *PriceListService
}

func NewServices(db *database.DB) *Services {
	priceListRepo := repository.NewPriceListRepository(db)
	priceItemRepo := repository.NewPriceListItemRepository(db)
	quantityTierRepo := repository.NewQuantityTierRepository(db)
	assignmentRepo := repository.NewPriceListAssignmentRepository(db)

	priceList := NewPriceListService()
	priceList.SetDependencies(priceListRepo, priceItemRepo, quantityTierRepo, assignmentRepo)

	return &Services{
		PriceList: priceList,
	}
}
