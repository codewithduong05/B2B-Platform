package service

// Checkout: cart → per-supplier orders + inventory reservations.
//
// Transaction boundaries (inventory owns its own transactions, so a single
// cross-module transaction is impossible without bypassing the boundary):
//
//  1. Claim the idempotency key (single INSERT, unique on buyer+key).
//  2. Validate lines, resolve server-side prices, availability pre-check.
//  3. Reserve each line via InventoryService.ReserveStock (one atomic
//     transaction per line inside Inventory, FEFO + row locks there).
//  4. One commerce transaction: lock cart, re-verify payload hash, create
//     orders + lines + history, clear cart lines, mark claim completed.
//  5. Publish commerce.order.placed per order, after commit, first
//     execution only.
//
// Any failure before step 4 completes releases the attempt's reservations
// and deletes the pending claim, so the same key may be retried.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	catalog_queries "github.com/atlas-platform/backend/internal/database/queries/catalog"
	"github.com/atlas-platform/backend/internal/modules/commerce/events"
	"github.com/atlas-platform/backend/internal/modules/commerce/repository"
	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
	inventory_schema "github.com/atlas-platform/backend/internal/modules/inventory/schema"
	inventory_service "github.com/atlas-platform/backend/internal/modules/inventory/service"
	pricing_schema "github.com/atlas-platform/backend/internal/modules/pricing/schema"
	"github.com/jackc/pgx/v5"
)

type CheckoutResult struct {
	Response *schema.CheckoutResponse
	Replayed bool
}

// stalePendingTimeout bounds how long a pending idempotency claim blocks
// its key. Checkouts finish in seconds; a pending row older than this is a
// crashed owner, not a live attempt, and the key may be reclaimed. Without
// this, a process crash between claim and completion would brick the key.
const stalePendingTimeout = 5 * time.Minute

// checkoutLine is a validated cart line with resolved references.
type checkoutLine struct {
	line         repository.CartLine
	productID    int64
	supplierID   int64
	unitID       int64
	quantity     int
	productCode  string
	productName  string
	unitCode     string
	supplierCode string
	supplierName string
	unitPrice    int64
	lineTotal    int64
	resRequestID string
}

func newOrderCode() string {
	return newPublicCode("ord_")
}

func newOrderLineCode() string {
	return newPublicCode("oline_")
}

// canonicalPayloadHash identifies the checkout intent: buyer + exact cart
// content. Same key + same hash replays; same key + different hash is 409.
func canonicalPayloadHash(buyerID int64, lines []repository.CartLine) string {
	type item struct {
		p, s, u int64
		q       int
	}
	items := make([]item, 0, len(lines))
	for _, l := range lines {
		items = append(items, item{l.ProductID, l.SupplierID, l.UnitID, l.Quantity})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].p != items[j].p {
			return items[i].p < items[j].p
		}
		if items[i].s != items[j].s {
			return items[i].s < items[j].s
		}
		return items[i].u < items[j].u
	})
	h := sha256.New()
	fmt.Fprintf(h, "buyer:%d;", buyerID)
	for _, it := range items {
		fmt.Fprintf(h, "%d:%d:%d:%d;", it.p, it.s, it.u, it.q)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// reservationRequestID derives a stable per-line inventory request id from
// the checkout key AND buyer, so retried attempts replay reservations
// instead of duplicating them. The buyer scope is load-bearing: bare keys
// are client-chosen and two buyers may pick the same string, which would
// otherwise alias their reservations. Bounded well under inventory's
// 100-char limit.
func reservationRequestID(idemKey string, buyerID, productID, supplierID int64) string {
	sum := sha256.Sum256([]byte(idemKey))
	return fmt.Sprintf("co-%s-%d-%d-%d", hex.EncodeToString(sum[:])[:16], buyerID, productID, supplierID)
}

func (s *CommerceService) Checkout(ctx context.Context, buyerID int64, idemKey string) (*CheckoutResult, error) {
	if idemKey == "" || len(idemKey) > 100 {
		return nil, ErrInvalidIdemKey
	}
	if s.inventorySvc == nil || s.pricingSvc == nil {
		return nil, fmt.Errorf("checkout dependencies not configured")
	}

	// The cart is implicit per buyer: a buyer without a cart row simply has
	// an empty cart (reads auto-create it), so checkout reports empty_cart
	// rather than cart_not_found.
	cart, err := s.getOrCreateCart(ctx, buyerID)
	if err != nil {
		return nil, fmt.Errorf("get or create cart: %w", err)
	}
	lines, err := s.repo.GetCartLines(ctx, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("get cart lines: %w", err)
	}
	payloadHash := canonicalPayloadHash(buyerID, lines)

	claim, created, err := s.repo.ClaimIdempotency(ctx, buyerID, idemKey, payloadHash)
	if err != nil {
		// Residual no-rows after bounded claim retries: concurrent claim
		// churn. Safe answer is in-flight (client retries), never 500.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCheckoutInFlight
		}
		return nil, fmt.Errorf("claim idempotency key: %w", err)
	}
	if !created {
		// The cart may already be cleared by the first execution; the
		// stored claim is authoritative for replays, not the cart state.
		// An empty cart with a completed claim is the normal retry-after-
		// success shape and replays. A non-empty cart must hash-match.
		if claim.Status == "completed" {
			if len(lines) != 0 && claim.PayloadHash != payloadHash {
				return nil, ErrIdempotencyConflict
			}
			var resp schema.CheckoutResponse
			if err := json.Unmarshal(claim.Result, &resp); err != nil {
				return nil, fmt.Errorf("decode stored checkout result: %w", err)
			}
			resp.Replayed = true
			return &CheckoutResult{Response: &resp, Replayed: true}, nil
		}
		// Pending: either a genuinely concurrent attempt (409) or a stale
		// row from a crashed owner (reclaim once, then proceed).
		if time.Since(claim.CreatedAt) < stalePendingTimeout {
			return nil, ErrCheckoutInFlight
		}
		_ = s.repo.DeleteIdempotency(ctx, claim.ID)
		reclaimed, recreated, err := s.repo.ClaimIdempotency(ctx, buyerID, idemKey, payloadHash)
		if err != nil {
			return nil, fmt.Errorf("claim idempotency key: %w", err)
		}
		if !recreated {
			return nil, ErrCheckoutInFlight
		}
		claim = reclaimed
	}
	if len(lines) == 0 {
		_ = s.repo.DeleteIdempotency(ctx, claim.ID)
		return nil, ErrEmptyCart
	}
	// From here on, every failure path must delete the pending claim so the
	// same key can be retried, and release any reservations made.
	fail := func(reservationIDs []string, err error) (*CheckoutResult, error) {
		if len(reservationIDs) > 0 && s.inventorySvc != nil {
			if rerr := s.inventorySvc.ReleaseReservationAttempt(ctx, reservationIDs); rerr != nil {
				_ = s.repo.DeleteIdempotency(ctx, claim.ID)
				return nil, fmt.Errorf("checkout failed (%v), rollback failed: %w", err, rerr)
			}
		}
		_ = s.repo.DeleteIdempotency(ctx, claim.ID)
		return nil, err
	}

	valid, err := s.validateCheckoutLines(ctx, buyerID, lines, idemKey)
	if err != nil {
		return fail(nil, err)
	}

	// Availability pre-check (best-effort; ReserveStock remains authoritative
	// under concurrency).
	if err := s.precheckAvailability(ctx, valid); err != nil {
		return fail(nil, err)
	}

	var reservationIDs []string
	for i := range valid {
		req := inventory_schema.ReserveStockRequest{
			ProductID:  valid[i].productID,
			SupplierID: valid[i].supplierID,
			Quantity:   int32(valid[i].quantity),
			RequestID:  valid[i].resRequestID,
		}
		_, err := s.inventorySvc.ReserveStock(ctx, req)
		if err != nil {
			switch {
			case errors.Is(err, inventory_service.ErrInsufficientStock),
				errors.Is(err, inventory_service.ErrStockLevelNotFound):
				return fail(reservationIDs, ErrInsufficientStock)
			case errors.Is(err, inventory_service.ErrDuplicateRequest):
				return fail(reservationIDs, ErrCheckoutInFlight)
			default:
				return fail(reservationIDs, fmt.Errorf("reserve stock: %w", err))
			}
		}
		reservationIDs = append(reservationIDs, valid[i].resRequestID)
	}

	resp, err := s.createOrderSet(ctx, buyerID, &claim, valid, payloadHash)
	if err != nil {
		return fail(reservationIDs, err)
	}

	s.publishOrderPlaced(ctx, buyerID, idemKey, resp)
	return &CheckoutResult{Response: resp, Replayed: false}, nil
}

// validateCheckoutLines re-applies the cart availability gates (published,
// active, supplier present) and resolves server-side unit prices.
func (s *CommerceService) validateCheckoutLines(ctx context.Context, buyerID int64, lines []repository.CartLine, idemKey string) ([]checkoutLine, error) {
	var quoteLines []pricing_schema.PriceQuoteLineRequest
	for _, l := range lines {
		bID := buyerID
		quoteLines = append(quoteLines, pricing_schema.PriceQuoteLineRequest{
			ProductID:      l.ProductID,
			UnitID:         l.UnitID,
			Quantity:       l.Quantity,
			BuyerProfileID: &bID,
		})
	}
	priceMap := make(map[int64]int64)
	quoteResp, err := s.pricingSvc.Quote(ctx, pricing_schema.PriceQuoteRequest{Lines: quoteLines})
	if err == nil && quoteResp != nil {
		for _, ql := range quoteResp.Lines {
			priceMap[ql.ProductID] = ql.UnitPriceMinor
		}
	}

	valid := make([]checkoutLine, 0, len(lines))
	for _, l := range lines {
		prod, err := s.productRepo.GetProductByID(ctx, l.ProductID)
		if err != nil {
			return nil, ErrProductNotFound
		}
		if prod.Status != catalog_queries.CatalogProductStatusPublished || !prod.IsActive {
			return nil, ErrProductNotFound
		}
		if !prod.SupplierID.Valid || prod.SupplierID.Int64 == 0 {
			return nil, ErrProductNotFound
		}
		supplier, err := s.supplierRepo.GetSupplierByID(ctx, l.SupplierID)
		if err != nil {
			return nil, ErrProductNotFound
		}
		unit, err := s.unitRepo.GetUnitByID(ctx, l.UnitID)
		if err != nil {
			return nil, ErrProductNotFound
		}

		unitPrice := priceMap[l.ProductID]
		if unitPrice == 0 && prod.BasePriceMinor.Valid {
			unitPrice = prod.BasePriceMinor.Int64
		}
		if unitPrice <= 0 {
			return nil, fmt.Errorf("no applicable price for product %d", l.ProductID)
		}

		valid = append(valid, checkoutLine{
			line:         l,
			productID:    l.ProductID,
			supplierID:   l.SupplierID,
			unitID:       l.UnitID,
			quantity:     l.Quantity,
			productCode:  prod.Code,
			productName:  prod.Name,
			unitCode:     unit.Code,
			supplierCode: supplier.Code,
			supplierName: supplier.Name,
			unitPrice:    unitPrice,
			lineTotal:    unitPrice * int64(l.Quantity),
			resRequestID: reservationRequestID(idemKey, buyerID, l.ProductID, l.SupplierID),
		})
	}
	return valid, nil
}

func (s *CommerceService) precheckAvailability(ctx context.Context, valid []checkoutLine) error {
	productIDs := make([]int64, 0, len(valid))
	for _, v := range valid {
		productIDs = append(productIDs, v.productID)
	}
	availabilities, err := s.inventorySvc.GetAvailability(ctx, productIDs, nil)
	if err != nil {
		return fmt.Errorf("check availability: %w", err)
	}
	availMap := make(map[int64]int32)
	for _, av := range availabilities {
		availMap[av.ProductID] += av.AvailableQuantity
	}
	for _, v := range valid {
		if availMap[v.productID] < int32(v.quantity) {
			return ErrInsufficientStock
		}
	}
	return nil
}

// createOrderSet runs the atomic commerce transaction: lock cart, verify the
// payload hash (the cart must not have changed under this key), create one
// order per supplier with price snapshots, record history, clear the cart,
// and mark the idempotency claim completed with the stored result.
func (s *CommerceService) createOrderSet(ctx context.Context, buyerID int64, claim *repository.IdempotencyClaim, valid []checkoutLine, payloadHash string) (*schema.CheckoutResponse, error) {
	var resp *schema.CheckoutResponse
	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)

		locked, err := txRepo.LockCartByBuyerID(ctx, buyerID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCartNotFound
			}
			return fmt.Errorf("lock cart: %w", err)
		}
		lockedLines, err := txRepo.GetCartLinesForUpdate(ctx, locked.ID)
		if err != nil {
			return fmt.Errorf("re-read cart lines: %w", err)
		}
		if len(lockedLines) == 0 {
			return ErrCheckoutConflict
		}
		if canonicalPayloadHash(buyerID, lockedLines) != payloadHash {
			return ErrIdempotencyConflict
		}
		var lockedLineIDs []int64
		for _, l := range lockedLines {
			lockedLineIDs = append(lockedLineIDs, l.ID)
		}

		groups := make(map[int64][]checkoutLine)
		var supplierOrder []int64
		for _, v := range valid {
			if _, ok := groups[v.supplierID]; !ok {
				supplierOrder = append(supplierOrder, v.supplierID)
			}
			groups[v.supplierID] = append(groups[v.supplierID], v)
		}

		resp = &schema.CheckoutResponse{Currency: locked.Currency}
		for _, supplierID := range supplierOrder {
			items := groups[supplierID]
			var subtotal int64
			for _, it := range items {
				subtotal += it.lineTotal
			}
			order, err := txRepo.CreateOrder(ctx, newOrderCode(), buyerID, supplierID,
				locked.ID, locked.Code, locked.Currency, subtotal, subtotal)
			if err != nil {
				return fmt.Errorf("create order: %w", err)
			}
			orderResp := schema.OrderResponse{
				Code:          order.Code,
				SupplierCode:  items[0].supplierCode,
				SupplierName:  items[0].supplierName,
				Status:        order.Status,
				Currency:      order.Currency,
				SubtotalMinor: order.SubtotalMinor,
				TotalMinor:    order.TotalMinor,
				PlacedAt:      order.PlacedAt,
			}
			for _, it := range items {
				ol, err := txRepo.CreateOrderLine(ctx, newOrderLineCode(), order.ID,
					it.productID, it.supplierID, it.unitID, it.quantity,
					it.unitPrice, it.lineTotal, locked.Currency,
					it.productCode, it.productName, it.unitCode)
				if err != nil {
					return fmt.Errorf("create order line: %w", err)
				}
				orderResp.Lines = append(orderResp.Lines, schema.OrderLineResponse{
					Code:            ol.Code,
					ProductCode:     ol.ProductCode,
					ProductName:     ol.ProductName,
					Quantity:        ol.Quantity,
					UnitCode:        ol.UnitCode,
					UnitPriceMinor:  ol.UnitPriceMinor,
					TotalPriceMinor: ol.TotalPriceMinor,
					Currency:        ol.Currency,
				})
			}
			var actor = buyerID
			if _, err := txRepo.CreateOrderHistory(ctx, order.ID, nil, order.Status, &actor, "checkout"); err != nil {
				return fmt.Errorf("record order history: %w", err)
			}
			resp.Orders = append(resp.Orders, orderResp)
		}

		if err := txRepo.ClearCartLinesByIDs(ctx, lockedLineIDs); err != nil {
			return fmt.Errorf("clear cart lines: %w", err)
		}

		raw, err := json.Marshal(resp)
		if err != nil {
			return fmt.Errorf("encode checkout result: %w", err)
		}
		if err := txRepo.CompleteIdempotency(ctx, claim.ID, raw); err != nil {
			return fmt.Errorf("complete idempotency claim: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *CommerceService) publishOrderPlaced(ctx context.Context, buyerID int64, idemKey string, resp *schema.CheckoutResponse) {
	if s.publisher == nil {
		return
	}
	for _, o := range resp.Orders {
		// Order IDs are internal; the payload carries public codes only.
		_ = s.publisher.Publish(ctx, events.EventOrderPlaced, events.NewEnvelope(
			events.EventOrderPlaced,
			events.OrderPlacedPayload{
				OrderCode:      o.Code,
				BuyerID:        buyerID,
				TotalMinor:     o.TotalMinor,
				Currency:       o.Currency,
				IdempotencyKey: idemKey,
			},
			idemKey,
		))
	}
}

// ListOrders returns the buyer's orders, newest first, without history.
func (s *CommerceService) ListOrders(ctx context.Context, buyerID int64) ([]schema.OrderResponse, error) {
	orders, err := s.repo.ListOrdersByBuyer(ctx, buyerID)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	out := make([]schema.OrderResponse, 0, len(orders))
	for _, o := range orders {
		lines, err := s.repo.GetOrderLines(ctx, o.ID)
		if err != nil {
			return nil, fmt.Errorf("get order lines: %w", err)
		}
		out = append(out, s.toOrderResponse(ctx, o, lines))
	}
	return out, nil
}

// GetOrder returns one order with history; foreign buyers get 404 (never 403,
// per the contract's information-leak rule).
func (s *CommerceService) GetOrder(ctx context.Context, buyerID int64, orderCode string) (*schema.OrderDetailResponse, error) {
	o, err := s.repo.GetOrderByCode(ctx, orderCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order: %w", err)
	}
	if o.BuyerID != buyerID {
		return nil, ErrOrderNotFound
	}
	lines, err := s.repo.GetOrderLines(ctx, o.ID)
	if err != nil {
		return nil, fmt.Errorf("get order lines: %w", err)
	}
	history, err := s.repo.GetOrderHistory(ctx, o.ID)
	if err != nil {
		return nil, fmt.Errorf("get order history: %w", err)
	}
	detail := &schema.OrderDetailResponse{OrderResponse: s.toOrderResponse(ctx, o, lines)}
	for _, h := range history {
		detail.History = append(detail.History, schema.OrderHistoryResponse{
			FromStatus: h.FromStatus,
			ToStatus:   h.ToStatus,
			Actor:      h.Actor,
			Reason:     h.Reason,
			CreatedAt:  h.CreatedAt,
		})
	}
	return detail, nil
}

func (s *CommerceService) toOrderResponse(ctx context.Context, o repository.Order, lines []repository.OrderLine) schema.OrderResponse {
	supplierCode := "default"
	supplierName := "Default Supplier"
	if supplier, err := s.supplierRepo.GetSupplierByID(ctx, o.SupplierID); err == nil {
		supplierCode = supplier.Code
		supplierName = supplier.Name
	}
	resp := schema.OrderResponse{
		Code:           o.Code,
		SupplierCode:   supplierCode,
		SupplierName:   supplierName,
		Status:         o.Status,
		Currency:       o.Currency,
		SubtotalMinor:  o.SubtotalMinor,
		DiscountsMinor: o.DiscountsMinor,
		TotalMinor:     o.TotalMinor,
		PlacedAt:       o.PlacedAt,
	}
	for _, l := range lines {
		resp.Lines = append(resp.Lines, schema.OrderLineResponse{
			Code:            l.Code,
			ProductCode:     l.ProductCode,
			ProductName:     l.ProductName,
			Quantity:        l.Quantity,
			UnitCode:        l.UnitCode,
			UnitPriceMinor:  l.UnitPriceMinor,
			TotalPriceMinor: l.TotalPriceMinor,
			Currency:        l.Currency,
		})
	}
	return resp
}
