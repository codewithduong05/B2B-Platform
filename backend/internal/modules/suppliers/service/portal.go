package service

// TASK-012: supplier portal operations. Linkage chain (all pre-existing
// keys, no new tables):
//
//	principal user_id → suppliers.supplier_profile.user_id
//	→ catalog.supplier rows (supplier_id = profile id)
//	→ commerce."order" rows (supplier_id = catalog id)
//
// Reads go through owning repositories; the fulfilment ack reuses
// commerce's locked TransitionOrder engine; stock writes go through
// inventory's transactional PushSupplierStock. No cross-module table
// access: every hop uses its owner's surface.

import (
	"context"
	"errors"
	"fmt"
	"time"

	catalog_repo "github.com/atlas-platform/backend/internal/modules/catalog/repository"
	commerce_repo "github.com/atlas-platform/backend/internal/modules/commerce/repository"
	commerce_schema "github.com/atlas-platform/backend/internal/modules/commerce/schema"
	commerce_service "github.com/atlas-platform/backend/internal/modules/commerce/service"
	inventory_service "github.com/atlas-platform/backend/internal/modules/inventory/service"
	"github.com/atlas-platform/backend/internal/modules/suppliers/schema"
	"github.com/jackc/pgx/v5"
)

var (
	ErrPortalForbidden = errors.New("no supplier profile linked")
	ErrInvalidStock    = errors.New("invalid stock push")
)

// SetPortalDependencies wires the cross-module surfaces the portal needs.
// Called by main and tests; portal methods fail closed when unset.
func (s *SupplierService) SetPortalDependencies(
	catalogSupplierRepo *catalog_repo.SupplierRepository,
	catalogProductRepo *catalog_repo.ProductRepository,
	commerceRepo *commerce_repo.CommerceRepository,
	commerceSvc *commerce_service.CommerceService,
	inventorySvc *inventory_service.InventoryService,
) {
	s.catalogSupplierRepo = catalogSupplierRepo
	s.catalogProductRepo = catalogProductRepo
	s.commerceRepo = commerceRepo
	s.commerceSvc = commerceSvc
	s.inventorySvc = inventorySvc
}

func (s *SupplierService) portalReady() error {
	if s.catalogSupplierRepo == nil || s.catalogProductRepo == nil ||
		s.commerceRepo == nil || s.commerceSvc == nil || s.inventorySvc == nil {
		return fmt.Errorf("portal dependencies not configured")
	}
	return nil
}

// resolveCatalogIDs maps the caller's operational profile to its catalog
// supplier rows. Unlinked principals get ErrSupplierNotFound (404, never
// 403, per the information-leak rule).
func (s *SupplierService) resolveCatalogIDs(ctx context.Context, userID int64) ([]int64, error) {
	profile, err := s.repo.GetProfileByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSupplierNotFound
		}
		return nil, fmt.Errorf("get supplier profile: %w", err)
	}
	rows, err := s.catalogSupplierRepo.ListSuppliers(ctx, 200, 0)
	if err != nil {
		return nil, fmt.Errorf("list catalog suppliers: %w", err)
	}
	var ids []int64
	for _, row := range rows {
		if row.SupplierID == profile.ID {
			ids = append(ids, row.ID)
		}
	}
	return ids, nil
}

// MyOrders lists orders directed to the caller's catalog suppliers.
func (s *SupplierService) MyOrders(ctx context.Context, userID int64) ([]schema.SupplierOrderResponse, error) {
	if err := s.portalReady(); err != nil {
		return nil, err
	}
	ids, err := s.resolveCatalogIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	var out []schema.SupplierOrderResponse
	for _, id := range ids {
		orders, err := s.commerceRepo.ListOrdersBySupplier(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("list supplier orders: %w", err)
		}
		for _, o := range orders {
			lines, err := s.commerceRepo.GetOrderLines(ctx, o.ID)
			if err != nil {
				return nil, fmt.Errorf("get order lines: %w", err)
			}
			resp := schema.SupplierOrderResponse{
				Code: o.Code, Status: o.Status, Currency: o.Currency,
				SubtotalMinor: o.SubtotalMinor, DiscountsMinor: o.DiscountsMinor,
				TotalMinor: o.TotalMinor, PlacedAt: o.PlacedAt,
			}
			for _, l := range lines {
				resp.Lines = append(resp.Lines, schema.SupplierOrderLineResponse{
					ProductCode: l.ProductCode, ProductName: l.ProductName,
					Quantity: l.Quantity, UnitCode: l.UnitCode,
					UnitPriceMinor: l.UnitPriceMinor, TotalPriceMinor: l.TotalPriceMinor,
				})
			}
			out = append(out, resp)
		}
	}
	if out == nil {
		out = []schema.SupplierOrderResponse{}
	}
	return out, nil
}

// AcknowledgeOrder confirms fulfilment of the supplier's own placed order,
// advancing it placed → confirmed through commerce's locked engine (actor
// is the supplier user). Anything else is a 422; concurrent acks collapse
// to exactly one success via the engine's conflict mapping.
func (s *SupplierService) AcknowledgeOrder(ctx context.Context, userID int64, orderCode, note string) (*schema.SupplierOrderResponse, error) {
	if err := s.portalReady(); err != nil {
		return nil, err
	}
	ids, err := s.resolveCatalogIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	allowed := make(map[int64]bool, len(ids))
	for _, id := range ids {
		allowed[id] = true
	}
	order, err := s.commerceRepo.GetOrderByCode(ctx, orderCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSupplierNotFound
		}
		return nil, fmt.Errorf("get order: %w", err)
	}
	if !allowed[order.SupplierID] {
		return nil, ErrSupplierNotFound
	}
	updated, err := s.commerceSvc.TransitionOrder(ctx, order.ID, userID, commerce_service.OrderStatusConfirmed, supplierNote(note))
	if err != nil {
		return nil, err
	}
	return s.supplierOrderResponse(ctx, updated.ID)
}

func supplierNote(note string) string {
	if note == "" {
		return "supplier acknowledgement"
	}
	return "supplier acknowledgement: " + note
}

func (s *SupplierService) supplierOrderResponse(ctx context.Context, orderID int64) (*schema.SupplierOrderResponse, error) {
	order, err := s.commerceRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, ErrSupplierNotFound
	}
	lines, err := s.commerceRepo.GetOrderLines(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("get order lines: %w", err)
	}
	resp := &schema.SupplierOrderResponse{
		Code: order.Code, Status: order.Status, Currency: order.Currency,
		SubtotalMinor: order.SubtotalMinor, DiscountsMinor: order.DiscountsMinor,
		TotalMinor: order.TotalMinor, PlacedAt: order.PlacedAt,
	}
	for _, l := range lines {
		resp.Lines = append(resp.Lines, schema.SupplierOrderLineResponse{
			ProductCode: l.ProductCode, ProductName: l.ProductName,
			Quantity: l.Quantity, UnitCode: l.UnitCode,
			UnitPriceMinor: l.UnitPriceMinor, TotalPriceMinor: l.TotalPriceMinor,
		})
	}
	return resp, nil
}

// PushStock records supplier-pushed stock for one of the caller's products.
// The product must belong to one of the caller's catalog suppliers;
// quantity bounds match the INT column. Allocation stays FEFO-owned.
func (s *SupplierService) PushStock(ctx context.Context, userID int64, req schema.PushStockRequest) (*schema.StockPushResponse, error) {
	if err := s.portalReady(); err != nil {
		return nil, err
	}
	if req.Quantity <= 0 || int64(req.Quantity) > 2147483647 {
		return nil, ErrInvalidStock
	}
	ids, err := s.resolveCatalogIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	allowed := make(map[int64]bool, len(ids))
	for _, id := range ids {
		allowed[id] = true
	}
	prod, err := s.catalogProductRepo.GetProductByCode(ctx, req.ProductCode)
	if err != nil {
		return nil, ErrSupplierNotFound
	}
	if !prod.SupplierID.Valid || !allowed[prod.SupplierID.Int64] {
		return nil, ErrSupplierNotFound
	}
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, ErrInvalidStock
		}
		expiresAt = &t
	}
	lot, err := s.inventorySvc.PushSupplierStock(ctx, prod.ID, prod.SupplierID.Int64, req.LotNumber, int32(req.Quantity), expiresAt)
	if err != nil {
		return nil, err
	}
	_ = commerce_schema.OrderResponse{}
	return &schema.StockPushResponse{
		LotCode:           lot.Code,
		LotNumber:         lot.LotNumber,
		AvailableQuantity: int(lot.AvailableQuantity),
	}, nil
}
