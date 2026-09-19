package repository

import (
	"context"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/database/queries/pricing"
)

type PriceListRepository struct {
	db *database.DB
	q  *pricing.Queries
}

func NewPriceListRepository(db *database.DB) *PriceListRepository {
	return &PriceListRepository{
		db: db,
		q:  pricing.New(db.Pool),
	}
}

func (r *PriceListRepository) CreatePriceList(ctx context.Context, params pricing.CreatePriceListParams) (pricing.CreatePriceListRow, error) {
	return r.q.CreatePriceList(ctx, params)
}

func (r *PriceListRepository) GetPriceListByID(ctx context.Context, id int64) (pricing.PricingPriceList, error) {
	return r.q.GetPriceListByID(ctx, id)
}

func (r *PriceListRepository) GetPriceListByCode(ctx context.Context, code string) (pricing.PricingPriceList, error) {
	return r.q.GetPriceListByCode(ctx, code)
}

func (r *PriceListRepository) UpdatePriceList(ctx context.Context, id int64, params pricing.UpdatePriceListParams) (pricing.UpdatePriceListRow, error) {
	return r.q.UpdatePriceList(ctx, params)
}

func (r *PriceListRepository) SoftDeletePriceList(ctx context.Context, id int64) error {
	return r.q.SoftDeletePriceList(ctx, id)
}

type PriceListFilters struct {
	Status   *pricing.PricingPriceListStatus
	IsActive *bool
}

func (r *PriceListRepository) ListPriceLists(ctx context.Context, limit, offset int32, filters PriceListFilters) ([]pricing.PricingPriceList, error) {
	var status pricing.PricingPriceListStatus
	var isActive bool

	if filters.Status != nil {
		status = *filters.Status
	}
	if filters.IsActive != nil {
		isActive = *filters.IsActive
	}

	return r.q.ListPriceLists(ctx, pricing.ListPriceListsParams{
		Column1: status,
		Column2: isActive,
		Limit:   limit,
		Offset:  offset,
	})
}

func (r *PriceListRepository) CountPriceLists(ctx context.Context, filters PriceListFilters) (int64, error) {
	var status pricing.PricingPriceListStatus
	var isActive bool

	if filters.Status != nil {
		status = *filters.Status
	}
	if filters.IsActive != nil {
		isActive = *filters.IsActive
	}

	return r.q.CountPriceLists(ctx, pricing.CountPriceListsParams{
		Column1: status,
		Column2: isActive,
	})
}

func (r *PriceListRepository) GetActivePriceLists(ctx context.Context) ([]pricing.PricingPriceList, error) {
	return r.q.GetActivePriceLists(ctx)
}

func (r *PriceListRepository) GetPriceListWithItems(ctx context.Context, id int64) (pricing.PricingPriceList, error) {
	return r.q.GetPriceListWithItems(ctx, id)
}

type PriceListItemRepository struct {
	db *database.DB
	q  *pricing.Queries
}

func NewPriceListItemRepository(db *database.DB) *PriceListItemRepository {
	return &PriceListItemRepository{
		db: db,
		q:  pricing.New(db.Pool),
	}
}

func (r *PriceListItemRepository) CreatePriceListItem(ctx context.Context, params pricing.CreatePriceListItemParams) (pricing.CreatePriceListItemRow, error) {
	return r.q.CreatePriceListItem(ctx, params)
}

func (r *PriceListItemRepository) GetPriceListItemByID(ctx context.Context, id int64) (pricing.PricingPriceListItem, error) {
	return r.q.GetPriceListItemByID(ctx, id)
}

func (r *PriceListItemRepository) UpdatePriceListItem(ctx context.Context, id int64, params pricing.UpdatePriceListItemParams) (pricing.UpdatePriceListItemRow, error) {
	return r.q.UpdatePriceListItem(ctx, params)
}

func (r *PriceListItemRepository) SoftDeletePriceListItem(ctx context.Context, id int64) error {
	return r.q.SoftDeletePriceListItem(ctx, id)
}

type PriceListItemFilters struct {
	PriceListID *int64
	ProductID   *int64
	UnitID      *int64
	IsActive    *bool
}

func (r *PriceListItemRepository) ListPriceListItems(ctx context.Context, limit, offset int32, filters PriceListItemFilters) ([]pricing.PricingPriceListItem, error) {
	var priceListID int64
	var productID int64
	var unitID int64
	var isActive bool

	if filters.PriceListID != nil {
		priceListID = *filters.PriceListID
	}
	if filters.ProductID != nil {
		productID = *filters.ProductID
	}
	if filters.UnitID != nil {
		unitID = *filters.UnitID
	}
	if filters.IsActive != nil {
		isActive = *filters.IsActive
	}

	return r.q.ListPriceListItems(ctx, pricing.ListPriceListItemsParams{
		Column1: priceListID,
		Column2: productID,
		Column3: unitID,
		Column4: isActive,
		Limit:   limit,
		Offset:  offset,
	})
}

func (r *PriceListItemRepository) CountPriceListItems(ctx context.Context, filters PriceListItemFilters) (int64, error) {
	var priceListID int64
	var productID int64
	var unitID int64
	var isActive bool

	if filters.PriceListID != nil {
		priceListID = *filters.PriceListID
	}
	if filters.ProductID != nil {
		productID = *filters.ProductID
	}
	if filters.UnitID != nil {
		unitID = *filters.UnitID
	}
	if filters.IsActive != nil {
		isActive = *filters.IsActive
	}

	return r.q.CountPriceListItems(ctx, pricing.CountPriceListItemsParams{
		Column1: priceListID,
		Column2: productID,
		Column3: unitID,
		Column4: isActive,
	})
}

func (r *PriceListItemRepository) GetPriceListItemsByList(ctx context.Context, priceListID int64) ([]pricing.PricingPriceListItem, error) {
	return r.q.GetPriceListItemsByList(ctx, priceListID)
}

func (r *PriceListItemRepository) GetPriceForProduct(ctx context.Context, priceListID, productID, unitID int64) ([]pricing.GetPriceForProductRow, error) {
	return r.q.GetPriceForProduct(ctx, pricing.GetPriceForProductParams{
		PriceListID: priceListID,
		ProductID:   productID,
		UnitID:      unitID,
	})
}

func (r *PriceListItemRepository) GetPriceForQuote(ctx context.Context, priceListID, productID, unitID int64) ([]pricing.GetPriceForQuoteRow, error) {
	return r.q.GetPriceForQuote(ctx, pricing.GetPriceForQuoteParams{
		PriceListID: priceListID,
		ProductID:   productID,
		UnitID:      unitID,
	})
}

type QuantityTierRepository struct {
	db *database.DB
	q  *pricing.Queries
}

func NewQuantityTierRepository(db *database.DB) *QuantityTierRepository {
	return &QuantityTierRepository{
		db: db,
		q:  pricing.New(db.Pool),
	}
}

func (r *QuantityTierRepository) CreateQuantityTier(ctx context.Context, params pricing.CreateQuantityTierParams) (pricing.CreateQuantityTierRow, error) {
	return r.q.CreateQuantityTier(ctx, params)
}

func (r *QuantityTierRepository) GetQuantityTierByID(ctx context.Context, id int64) (pricing.PricingQuantityTier, error) {
	return r.q.GetQuantityTierByID(ctx, id)
}

func (r *QuantityTierRepository) UpdateQuantityTier(ctx context.Context, id int64, params pricing.UpdateQuantityTierParams) (pricing.UpdateQuantityTierRow, error) {
	return r.q.UpdateQuantityTier(ctx, params)
}

func (r *QuantityTierRepository) SoftDeleteQuantityTier(ctx context.Context, id int64) error {
	return r.q.SoftDeleteQuantityTier(ctx, id)
}

type QuantityTierFilters struct {
	PriceListItemID *int64
	IsActive        *bool
}

func (r *QuantityTierRepository) ListQuantityTiers(ctx context.Context, limit, offset int32, filters QuantityTierFilters) ([]pricing.PricingQuantityTier, error) {
	var priceListItemID int64
	var isActive bool

	if filters.PriceListItemID != nil {
		priceListItemID = *filters.PriceListItemID
	}
	if filters.IsActive != nil {
		isActive = *filters.IsActive
	}

	return r.q.ListQuantityTiers(ctx, pricing.ListQuantityTiersParams{
		Column1: priceListItemID,
		Column2: isActive,
	})
}

func (r *QuantityTierRepository) GetQuantityTiersByItem(ctx context.Context, priceListItemID int64) ([]pricing.PricingQuantityTier, error) {
	return r.q.GetQuantityTiersByItem(ctx, priceListItemID)
}

type PriceListAssignmentRepository struct {
	db *database.DB
	q  *pricing.Queries
}

func NewPriceListAssignmentRepository(db *database.DB) *PriceListAssignmentRepository {
	return &PriceListAssignmentRepository{
		db: db,
		q:  pricing.New(db.Pool),
	}
}

func (r *PriceListAssignmentRepository) CreatePriceListAssignment(ctx context.Context, params pricing.CreatePriceListAssignmentParams) (pricing.CreatePriceListAssignmentRow, error) {
	return r.q.CreatePriceListAssignment(ctx, params)
}

func (r *PriceListAssignmentRepository) GetPriceListAssignmentByID(ctx context.Context, id int64) (pricing.PricingPriceListAssignment, error) {
	return r.q.GetPriceListAssignmentByID(ctx, id)
}

func (r *PriceListAssignmentRepository) GetPriceListAssignmentByCode(ctx context.Context, code string) (pricing.PricingPriceListAssignment, error) {
	return r.q.GetPriceListAssignmentByCode(ctx, code)
}

func (r *PriceListAssignmentRepository) UpdatePriceListAssignment(ctx context.Context, id int64, params pricing.UpdatePriceListAssignmentParams) (pricing.UpdatePriceListAssignmentRow, error) {
	return r.q.UpdatePriceListAssignment(ctx, params)
}

func (r *PriceListAssignmentRepository) SoftDeletePriceListAssignment(ctx context.Context, id int64) error {
	return r.q.SoftDeletePriceListAssignment(ctx, id)
}

type PriceListAssignmentFilters struct {
	PriceListID    *int64
	BuyerProfileID *int64
	IsActive       *bool
}

func (r *PriceListAssignmentRepository) ListPriceListAssignments(ctx context.Context, limit, offset int32, filters PriceListAssignmentFilters) ([]pricing.PricingPriceListAssignment, error) {
	var priceListID int64
	var buyerProfileID int64
	var isActive bool

	if filters.PriceListID != nil {
		priceListID = *filters.PriceListID
	}
	if filters.BuyerProfileID != nil {
		buyerProfileID = *filters.BuyerProfileID
	}
	if filters.IsActive != nil {
		isActive = *filters.IsActive
	}

	return r.q.ListPriceListAssignments(ctx, pricing.ListPriceListAssignmentsParams{
		Column1: priceListID,
		Column2: buyerProfileID,
		Column3: isActive,
		Limit:   limit,
		Offset:  offset,
	})
}

func (r *PriceListAssignmentRepository) CountPriceListAssignments(ctx context.Context, filters PriceListAssignmentFilters) (int64, error) {
	var priceListID int64
	var buyerProfileID int64
	var isActive bool

	if filters.PriceListID != nil {
		priceListID = *filters.PriceListID
	}
	if filters.BuyerProfileID != nil {
		buyerProfileID = *filters.BuyerProfileID
	}
	if filters.IsActive != nil {
		isActive = *filters.IsActive
	}

	return r.q.CountPriceListAssignments(ctx, pricing.CountPriceListAssignmentsParams{
		Column1: priceListID,
		Column2: buyerProfileID,
		Column3: isActive,
	})
}

func (r *PriceListAssignmentRepository) GetActiveAssignmentForBuyer(ctx context.Context, buyerProfileID int64) (pricing.PricingPriceListAssignment, error) {
	return r.q.GetActiveAssignmentForBuyer(ctx, buyerProfileID)
}

func (r *PriceListAssignmentRepository) WithTx(ctx context.Context, fn func(*PriceListAssignmentRepository) error) error {
	return r.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := &PriceListAssignmentRepository{
			db: r.db,
			q:  pricing.New(tx),
		}
		return fn(txRepo)
	})
}
