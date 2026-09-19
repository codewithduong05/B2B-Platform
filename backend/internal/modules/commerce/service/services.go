package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	catalog_queries "github.com/atlas-platform/backend/internal/database/queries/catalog"
	catalog_repo "github.com/atlas-platform/backend/internal/modules/catalog/repository"
	"github.com/atlas-platform/backend/internal/modules/commerce/repository"
	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
	inventory_service "github.com/atlas-platform/backend/internal/modules/inventory/service"
	pricing_schema "github.com/atlas-platform/backend/internal/modules/pricing/schema"
	pricing_service "github.com/atlas-platform/backend/internal/modules/pricing/service"
	"github.com/jackc/pgx/v5"
)

var (
	ErrCartNotFound        = errors.New("cart not found")
	ErrCartLineNotFound    = errors.New("cart line not found")
	ErrProductNotFound     = errors.New("product not found")
	ErrInvalidQuantity     = errors.New("invalid quantity")
	ErrEmptyCart           = errors.New("cart is empty")
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrIdempotencyConflict = errors.New("idempotency key reused for a different intent")
	ErrCheckoutInFlight    = errors.New("checkout already in progress for this key")
	ErrCheckoutConflict    = errors.New("cart was consumed by another checkout")
	ErrInvalidIdemKey      = errors.New("invalid idempotency key")
	ErrOrderNotFound       = errors.New("order not found")
)

// EventPublisher mirrors the inventory module's publisher shape: routing key
// plus payload. The production wiring passes nil (same as inventory), in
// which case checkout publishes nothing.
type EventPublisher interface {
	Publish(ctx context.Context, routingKey string, payload interface{}) error
}

type CommerceService struct {
	db           *database.DB
	repo         *repository.CommerceRepository
	productRepo  *catalog_repo.ProductRepository
	supplierRepo *catalog_repo.SupplierRepository
	unitRepo     *catalog_repo.UnitRepository
	pricingSvc   *pricing_service.PriceListService
	inventorySvc *inventory_service.InventoryService
	publisher    EventPublisher
}

func NewCommerceService(
	db *database.DB,
	pricingSvc *pricing_service.PriceListService,
	inventorySvc *inventory_service.InventoryService,
	publisher EventPublisher,
) *CommerceService {
	return &CommerceService{
		db:           db,
		repo:         repository.NewCommerceRepository(db),
		productRepo:  catalog_repo.NewProductRepository(db),
		supplierRepo: catalog_repo.NewSupplierRepository(db),
		unitRepo:     catalog_repo.NewUnitRepository(db),
		pricingSvc:   pricingSvc,
		inventorySvc: inventorySvc,
		publisher:    publisher,
	}
}

func (s *CommerceService) getOrCreateCart(ctx context.Context, buyerID int64) (repository.Cart, error) {
	cart, err := s.repo.GetCartByBuyerID(ctx, buyerID)
	if err == nil {
		return cart, nil
	}

	return s.repo.CreateCart(ctx, newCartCode(), buyerID, "USD")
}

// newPublicCode builds an opaque public code that always fits the
// VARCHAR(26) code columns. time.Now().UnixNano() alone is 19 digits, so
// embedding extra ids (e.g. buyer_id) overflows once ids reach 2 digits.
// base36 keeps the timestamp compact and a random suffix prevents
// same-nanosecond collisions under concurrent creation.
func newPublicCode(prefix string) string {
	return fmt.Sprintf("%s%s%s",
		prefix,
		strconv.FormatInt(time.Now().UnixNano(), 36),
		strconv.FormatUint(uint64(rand.Uint32()), 36),
	)
}

func newCartCode() string {
	return newPublicCode("cart_")
}

func newCartLineCode() string {
	return newPublicCode("cline_")
}

func (s *CommerceService) GetCart(ctx context.Context, buyerID int64) (*schema.CartResponse, error) {
	cart, err := s.getOrCreateCart(ctx, buyerID)
	if err != nil {
		return nil, fmt.Errorf("get or create cart: %w", err)
	}

	lines, err := s.repo.GetCartLines(ctx, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("get cart lines: %w", err)
	}

	supplierGroupsMap := make(map[string]*schema.SupplierCartGroup)

	for _, l := range lines {
		prod, err := s.productRepo.GetProductByID(ctx, l.ProductID)
		if err != nil {
			continue
		}

		supplier, err := s.supplierRepo.GetSupplierByID(ctx, l.SupplierID)
		supplierCode := "default"
		supplierName := "Default Supplier"
		if err == nil {
			supplierCode = supplier.Code
			supplierName = supplier.Name
		}

		unit, err := s.unitRepo.GetUnitByID(ctx, l.UnitID)
		unitCode := "pcs"
		if err == nil {
			unitCode = unit.Code
		}

		lineResp := schema.CartLineResponse{
			Code:         l.Code,
			ProductCode:  prod.Code,
			ProductName:  prod.Name,
			Quantity:     l.Quantity,
			UnitCode:     unitCode,
			SupplierCode: supplierCode,
			CreatedAt:    l.CreatedAt,
			UpdatedAt:    l.UpdatedAt,
		}

		group, exists := supplierGroupsMap[supplierCode]
		if !exists {
			group = &schema.SupplierCartGroup{
				SupplierCode: supplierCode,
				SupplierName: supplierName,
				Items:        []schema.CartLineResponse{},
			}
			supplierGroupsMap[supplierCode] = group
		}
		group.Items = append(group.Items, lineResp)
	}

	var suppliers []schema.SupplierCartGroup
	for _, g := range supplierGroupsMap {
		suppliers = append(suppliers, *g)
	}

	return &schema.CartResponse{
		Code:      cart.Code,
		BuyerID:   cart.BuyerID,
		Currency:  cart.Currency,
		Suppliers: suppliers,
		CreatedAt: cart.CreatedAt,
		UpdatedAt: cart.UpdatedAt,
	}, nil
}

func (s *CommerceService) AddCartItem(ctx context.Context, buyerID int64, req schema.AddCartItemRequest) (*schema.CartResponse, error) {
	// commerce.cart_line.quantity is an INT column; larger values fail at
	// the driver with a 500, so reject them as invalid input instead.
	if req.Quantity <= 0 || int64(req.Quantity) > math.MaxInt32 {
		return nil, ErrInvalidQuantity
	}

	cart, err := s.getOrCreateCart(ctx, buyerID)
	if err != nil {
		return nil, fmt.Errorf("get or create cart: %w", err)
	}

	prod, err := s.productRepo.GetProductByCode(ctx, req.ProductCode)
	if err != nil {
		return nil, ErrProductNotFound
	}

	// Same visibility gate the catalog module enforces for buyer-facing
	// reads (status published + active): an unpublished product must be
	// unreachable, not merely unlisted. A line also cannot reference a
	// product without a supplier (FK + per-supplier grouping require it).
	if prod.Status != catalog_queries.CatalogProductStatusPublished || !prod.IsActive {
		return nil, ErrProductNotFound
	}
	if !prod.SupplierID.Valid || prod.SupplierID.Int64 == 0 {
		return nil, ErrProductNotFound
	}

	_, err = s.repo.UpsertCartLine(ctx, newCartLineCode(), cart.ID, prod.ID, prod.SupplierID.Int64, prod.BaseUnitID, req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("upsert cart line: %w", err)
	}

	return s.GetCart(ctx, buyerID)
}

func (s *CommerceService) UpdateCartItem(ctx context.Context, buyerID int64, lineCode string, req schema.UpdateCartItemRequest) (*schema.CartResponse, error) {
	if req.Quantity <= 0 || int64(req.Quantity) > math.MaxInt32 {
		return nil, ErrInvalidQuantity
	}

	cart, err := s.repo.GetCartByBuyerID(ctx, buyerID)
	if err != nil {
		return nil, ErrCartNotFound
	}

	line, err := s.repo.GetCartLineByCode(ctx, lineCode)
	if err != nil || line.CartID != cart.ID {
		return nil, ErrCartLineNotFound
	}

	// A concurrent delete can land between the ownership check above and
	// this write; the UPDATE then matches no rows. That is a 404 (the line
	// is gone), not a 500.
	_, err = s.repo.UpdateCartLineQuantity(ctx, line.ID, req.Quantity)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCartLineNotFound
		}
		return nil, fmt.Errorf("update cart line quantity: %w", err)
	}

	return s.GetCart(ctx, buyerID)
}

func (s *CommerceService) DeleteCartItem(ctx context.Context, buyerID int64, lineCode string) (*schema.CartResponse, error) {
	cart, err := s.repo.GetCartByBuyerID(ctx, buyerID)
	if err != nil {
		return nil, ErrCartNotFound
	}

	line, err := s.repo.GetCartLineByCode(ctx, lineCode)
	if err != nil || line.CartID != cart.ID {
		return nil, ErrCartLineNotFound
	}

	err = s.repo.DeleteCartLine(ctx, line.ID)
	if err != nil {
		return nil, fmt.Errorf("delete cart line: %w", err)
	}

	return s.GetCart(ctx, buyerID)
}

func (s *CommerceService) QuoteCart(ctx context.Context, buyerID int64) (*schema.CartQuoteResponse, error) {
	cart, err := s.repo.GetCartByBuyerID(ctx, buyerID)
	if err != nil {
		return &schema.CartQuoteResponse{
			Suppliers:      []schema.SupplierQuoteGroup{},
			SubtotalMinor:  0,
			DiscountsMinor: 0,
			TotalMinor:     0,
			Currency:       "USD",
		}, nil
	}

	lines, err := s.repo.GetCartLines(ctx, cart.ID)
	if err != nil || len(lines) == 0 {
		return &schema.CartQuoteResponse{
			Suppliers:      []schema.SupplierQuoteGroup{},
			SubtotalMinor:  0,
			DiscountsMinor: 0,
			TotalMinor:     0,
			Currency:       cart.Currency,
		}, nil
	}

	var productIDs []int64
	var quoteLines []pricing_schema.PriceQuoteLineRequest
	for _, l := range lines {
		productIDs = append(productIDs, l.ProductID)
		bID := buyerID
		quoteLines = append(quoteLines, pricing_schema.PriceQuoteLineRequest{
			ProductID:      l.ProductID,
			UnitID:         l.UnitID,
			Quantity:       l.Quantity,
			BuyerProfileID: &bID,
		})
	}

	priceMap := make(map[int64]int64)
	if s.pricingSvc != nil {
		quoteResp, err := s.pricingSvc.Quote(ctx, pricing_schema.PriceQuoteRequest{Lines: quoteLines})
		if err == nil && quoteResp != nil {
			for _, ql := range quoteResp.Lines {
				priceMap[ql.ProductID] = ql.UnitPriceMinor
			}
		}
	}

	availabilityMap := make(map[int64]int32)
	if s.inventorySvc != nil {
		availabilities, err := s.inventorySvc.GetAvailability(ctx, productIDs, nil)
		if err == nil {
			for _, av := range availabilities {
				availabilityMap[av.ProductID] += av.AvailableQuantity
			}
		}
	}

	supplierGroupsMap := make(map[string]*schema.SupplierQuoteGroup)
	var totalSubtotal int64

	for _, l := range lines {
		prod, err := s.productRepo.GetProductByID(ctx, l.ProductID)
		if err != nil {
			continue
		}

		supplier, err := s.supplierRepo.GetSupplierByID(ctx, l.SupplierID)
		supplierCode := "default"
		supplierName := "Default Supplier"
		if err == nil {
			supplierCode = supplier.Code
			supplierName = supplier.Name
		}

		unit, err := s.unitRepo.GetUnitByID(ctx, l.UnitID)
		unitCode := "pcs"
		if err == nil {
			unitCode = unit.Code
		}

		unitPriceMinor := priceMap[l.ProductID]
		if unitPriceMinor == 0 && prod.BasePriceMinor.Valid {
			unitPriceMinor = prod.BasePriceMinor.Int64
		}

		totalPriceMinor := unitPriceMinor * int64(l.Quantity)
		availQty := availabilityMap[l.ProductID]
		available := availQty >= int32(l.Quantity)

		quoteLineResp := schema.CartLineQuote{
			Code:              l.Code,
			ProductCode:       prod.Code,
			ProductName:       prod.Name,
			Quantity:          l.Quantity,
			UnitCode:          unitCode,
			UnitPriceMinor:    unitPriceMinor,
			TotalPriceMinor:   totalPriceMinor,
			Currency:          cart.Currency,
			Available:         available,
			AvailableQuantity: availQty,
		}

		group, exists := supplierGroupsMap[supplierCode]
		if !exists {
			group = &schema.SupplierQuoteGroup{
				SupplierCode:  supplierCode,
				SupplierName:  supplierName,
				Items:         []schema.CartLineQuote{},
				SubtotalMinor: 0,
				Currency:      cart.Currency,
			}
			supplierGroupsMap[supplierCode] = group
		}
		group.Items = append(group.Items, quoteLineResp)
		group.SubtotalMinor += totalPriceMinor
		totalSubtotal += totalPriceMinor
	}

	var suppliers []schema.SupplierQuoteGroup
	for _, g := range supplierGroupsMap {
		suppliers = append(suppliers, *g)
	}

	return &schema.CartQuoteResponse{
		Suppliers:      suppliers,
		SubtotalMinor:  totalSubtotal,
		DiscountsMinor: 0,
		TotalMinor:     totalSubtotal,
		Currency:       cart.Currency,
	}, nil
}
