package service

// TASK-006: B2B order lifecycle. The state machine is enforced here; every
// transition records an order_history row with the actor. Only transitions
// with a catalogued domain event publish one (cancelled, dispatched,
// shipment.updated); all other transitions are silent by design.

import (
	"context"
	"errors"
	"fmt"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/commerce/events"
	"github.com/atlas-platform/backend/internal/modules/commerce/repository"
	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidTransition     = errors.New("invalid order status transition")
	ErrOrderOnHold           = errors.New("order is on hold")
	ErrIncompleteFulfillment = errors.New("order lines are not fully shipped")
	ErrOverpayment           = errors.New("payment exceeds outstanding balance")
	ErrOrderConflict         = errors.New("order state changed underneath")
)

const (
	OrderStatusPlaced     = "placed"
	OrderStatusConfirmed  = "confirmed"
	OrderStatusProcessing = "processing"
	OrderStatusShipped    = "shipped"
	OrderStatusDelivered  = "delivered"
	OrderStatusCancelled  = "cancelled"
)

var validTransitions = map[string][]string{
	OrderStatusPlaced:     {OrderStatusConfirmed, OrderStatusCancelled},
	OrderStatusConfirmed:  {OrderStatusProcessing, OrderStatusCancelled},
	OrderStatusProcessing: {OrderStatusShipped, OrderStatusCancelled},
	OrderStatusShipped:    {OrderStatusDelivered},
	OrderStatusDelivered:  {},
	OrderStatusCancelled:  {},
}

func isValidTransition(from, to string) bool {
	for _, next := range validTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// ResolveOrderRef returns internal order identity for a buyer-owned order.
// It powers cross-module references (e.g. payment intents) without leaking
// internal IDs into public responses: ownership is enforced here.
func (s *CommerceService) ResolveOrderRef(ctx context.Context, buyerID int64, orderCode string) (orderID, totalMinor int64, currency string, err error) {
	o, err := s.repo.GetOrderByCode(ctx, orderCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, "", ErrOrderNotFound
		}
		return 0, 0, "", fmt.Errorf("get order: %w", err)
	}
	if o.BuyerID != buyerID {
		return 0, 0, "", ErrOrderNotFound
	}
	return o.ID, o.TotalMinor, o.Currency, nil
}

// orderByID loads an order mapping a missing row to ErrOrderNotFound while
// preserving genuine database errors.
func (s *CommerceService) orderByID(ctx context.Context, id int64) (repository.Order, error) {
	o, err := s.repo.GetOrderByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository.Order{}, ErrOrderNotFound
		}
		return repository.Order{}, fmt.Errorf("get order: %w", err)
	}
	return o, nil
}

// AdminListOrders returns staff order summaries with the list total.
func (s *CommerceService) AdminListOrders(ctx context.Context, status string, limit, offset int) ([]schema.OrderResponse, int, error) {
	orders, err := s.repo.ListOrdersAdmin(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	total, err := s.repo.CountOrdersAdmin(ctx, status)
	if err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}
	out := make([]schema.OrderResponse, 0, len(orders))
	for _, o := range orders {
		lines, err := s.repo.GetOrderLines(ctx, o.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("get order lines: %w", err)
		}
		out = append(out, s.toOrderResponse(ctx, o, lines))
	}
	return out, total, nil
}

// AdminGetOrder returns full staff order detail with history and shipments.
func (s *CommerceService) AdminGetOrder(ctx context.Context, id int64) (*schema.AdminOrderDetailResponse, error) {
	o, err := s.orderByID(ctx, id)
	if err != nil {
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
	shipments, err := s.repo.GetShipmentsByOrder(ctx, o.ID)
	if err != nil {
		return nil, fmt.Errorf("get shipments: %w", err)
	}
	ordered := make(map[int64]repository.OrderLine)
	for _, l := range lines {
		ordered[l.ID] = l
	}
	detail := &schema.AdminOrderDetailResponse{}
	detail.OrderResponse = s.toOrderResponse(ctx, o, lines)
	for _, h := range history {
		detail.History = append(detail.History, schema.OrderHistoryResponse{
			FromStatus: h.FromStatus, ToStatus: h.ToStatus, Actor: h.Actor,
			Reason: h.Reason, CreatedAt: h.CreatedAt,
		})
	}
	for _, sh := range shipments {
		slines, err := s.repo.GetShipmentLines(ctx, sh.ID)
		if err != nil {
			return nil, fmt.Errorf("get shipment lines: %w", err)
		}
		detail.Shipments = append(detail.Shipments, s.toShipmentResponse(sh, slines, ordered))
	}
	if detail.Shipments == nil {
		detail.Shipments = []schema.ShipmentResponse{}
	}
	return detail, nil
}

// TransitionOrder moves one order to a new status, enforcing the state
// machine and hold flag, recording history, and publishing the catalogued
// event for the transition (if any). actor is the staff principal ID.
//
// Concurrency: pre-checks outside the transaction preserve the 422 contract
// for sequential misuse; the order row is then locked inside the transaction
// and every precondition is re-validated. A mismatch means a concurrent
// writer won the race and surfaces as ErrOrderConflict (409), never a
// duplicated transition.
func (s *CommerceService) TransitionOrder(ctx context.Context, orderID int64, actor int64, toStatus, reason string) (*repository.Order, error) {
	order, err := s.orderByID(ctx, orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order.OnHold {
		return nil, ErrOrderOnHold
	}
	if !isValidTransition(order.Status, toStatus) {
		return nil, ErrInvalidTransition
	}

	// Coverage for shipped is checked inside the transaction (locked);
	// see below.
	var updated repository.Order
	err = s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		locked, err := txRepo.GetOrderByIDForUpdate(ctx, order.ID)
		if err != nil {
			return fmt.Errorf("lock order: %w", err)
		}
		if locked.OnHold {
			return ErrOrderOnHold
		}
		if locked.Status != order.Status || !isValidTransition(locked.Status, toStatus) {
			return ErrOrderConflict
		}
		if toStatus == OrderStatusShipped {
			covered, err := fulfillmentCoveredTx(ctx, txRepo, order.ID)
			if err != nil {
				return err
			}
			if !covered {
				return ErrIncompleteFulfillment
			}
		}
		u, err := txRepo.UpdateOrderStatus(ctx, order.ID, toStatus)
		if err != nil {
			return fmt.Errorf("update order status: %w", err)
		}
		updated = u
		from := locked.Status
		if _, err := txRepo.CreateOrderHistory(ctx, order.ID, &from, toStatus, &actor, reason); err != nil {
			return fmt.Errorf("record order history: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.publishLifecycleEvent(ctx, order.BuyerID, &updated)
	return &updated, nil
}

// CancelOrder cancels a pre-fulfillment order and releases its checkout
// reservations. Cancellation is rejected while active (non-cancelled)
// shipments exist: stock already handed to fulfillment must not be freed.
//
// Locking: the order row is locked, validated, and (after release) updated
// in a single transaction. The inventory release runs while the order lock
// is held — a deliberate, bounded same-database exception to "no I/O inside
// a transaction": without it, a concurrent ship could commit between release
// and status update, freeing stock for a shipped order. Lock ordering is
// global (order row → inventory lots/levels) and unopposed by any other
// path, so no deadlock cycle exists.
func (s *CommerceService) CancelOrder(ctx context.Context, orderID int64, actor int64, reason string) (*repository.Order, error) {
	order, err := s.orderByID(ctx, orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order.OnHold {
		return nil, ErrOrderOnHold
	}
	if !isValidTransition(order.Status, OrderStatusCancelled) {
		return nil, ErrInvalidTransition
	}
	if active, err := s.hasActiveShipments(ctx, order.ID); err != nil {
		return nil, err
	} else if active {
		return nil, ErrOrderConflict
	}

	var updated repository.Order
	err = s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		locked, err := txRepo.GetOrderByIDForUpdate(ctx, order.ID)
		if err != nil {
			return fmt.Errorf("lock order: %w", err)
		}
		if locked.OnHold {
			return ErrOrderOnHold
		}
		if locked.Status != order.Status || !isValidTransition(locked.Status, OrderStatusCancelled) {
			return ErrOrderConflict
		}
		if active, err := hasActiveShipmentsTx(ctx, txRepo, order.ID); err != nil {
			return err
		} else if active {
			return ErrOrderConflict
		}

		if locked.IdemKey != nil && *locked.IdemKey != "" {
			lines, err := txRepo.GetOrderLines(ctx, order.ID)
			if err != nil {
				return fmt.Errorf("get order lines: %w", err)
			}
			var requestIDs []string
			for _, l := range lines {
				requestIDs = append(requestIDs, reservationRequestID(*locked.IdemKey, locked.BuyerID, l.ProductID, l.SupplierID))
			}
			if err := s.inventorySvc.ReleaseReservationAttempt(ctx, requestIDs); err != nil {
				return fmt.Errorf("release reservations: %w", err)
			}
		}

		u, err := txRepo.UpdateOrderStatus(ctx, order.ID, OrderStatusCancelled)
		if err != nil {
			return fmt.Errorf("update order status: %w", err)
		}
		updated = u
		from := locked.Status
		if _, err := txRepo.CreateOrderHistory(ctx, order.ID, &from, OrderStatusCancelled, &actor, reason); err != nil {
			return fmt.Errorf("record order history: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.publishLifecycleEvent(ctx, order.BuyerID, &updated)
	return &updated, nil
}

// hasActiveShipments reports whether non-cancelled shipments exist.
func (s *CommerceService) hasActiveShipments(ctx context.Context, orderID int64) (bool, error) {
	return hasActiveShipmentsTx(ctx, s.repo, orderID)
}

type shipmentLister interface {
	GetShipmentsByOrder(ctx context.Context, orderID int64) ([]repository.Shipment, error)
}

func hasActiveShipmentsTx(ctx context.Context, repo shipmentLister, orderID int64) (bool, error) {
	shipments, err := repo.GetShipmentsByOrder(ctx, orderID)
	if err != nil {
		return false, fmt.Errorf("get shipments: %w", err)
	}
	for _, sh := range shipments {
		if sh.Status != ShipmentStatusCancelled {
			return true, nil
		}
	}
	return false, nil
}

// HoldOrder freezes an order: all status transitions are rejected until
// ReleaseHold. The hold itself is recorded in history (from == to).
func (s *CommerceService) HoldOrder(ctx context.Context, orderID int64, actor int64, reason string) (*repository.Order, error) {
	if reason == "" {
		return nil, fmt.Errorf("hold reason is required")
	}
	order, err := s.orderByID(ctx, orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order.Status == OrderStatusDelivered || order.Status == OrderStatusCancelled {
		return nil, ErrInvalidTransition
	}
	if order.OnHold {
		return &order, nil
	}

	var updated repository.Order
	err = s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		locked, err := txRepo.GetOrderByIDForUpdate(ctx, order.ID)
		if err != nil {
			return fmt.Errorf("lock order: %w", err)
		}
		if locked.Status == OrderStatusDelivered || locked.Status == OrderStatusCancelled {
			return ErrInvalidTransition
		}
		if locked.OnHold {
			updated = locked
			return nil
		}
		u, err := txRepo.SetOrderHold(ctx, order.ID, true, &reason)
		if err != nil {
			return fmt.Errorf("hold order: %w", err)
		}
		updated = u
		if _, err := txRepo.CreateOrderHistory(ctx, order.ID, &locked.Status, locked.Status, &actor, "hold: "+reason); err != nil {
			return fmt.Errorf("record order history: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// ReleaseHoldOrder lifts a hold.
func (s *CommerceService) ReleaseHoldOrder(ctx context.Context, orderID int64, actor int64, reason string) (*repository.Order, error) {
	order, err := s.orderByID(ctx, orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if !order.OnHold {
		return &order, nil
	}

	var updated repository.Order
	err = s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		locked, err := txRepo.GetOrderByIDForUpdate(ctx, order.ID)
		if err != nil {
			return fmt.Errorf("lock order: %w", err)
		}
		if !locked.OnHold {
			updated = locked
			return nil
		}
		u, err := txRepo.SetOrderHold(ctx, order.ID, false, nil)
		if err != nil {
			return fmt.Errorf("release hold: %w", err)
		}
		updated = u
		if _, err := txRepo.CreateOrderHistory(ctx, order.ID, &locked.Status, locked.Status, &actor, "hold released: "+reason); err != nil {
			return fmt.Errorf("record order history: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

// AddOrderNote records an internal note in history without changing status.
func (s *CommerceService) AddOrderNote(ctx context.Context, orderID int64, actor int64, note string) (*repository.Order, error) {
	if note == "" {
		return nil, fmt.Errorf("note is required")
	}
	order, err := s.orderByID(ctx, orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if _, err := s.repo.CreateOrderHistory(ctx, order.ID, &order.Status, order.Status, &actor, "note: "+note); err != nil {
		return nil, fmt.Errorf("record order note: %w", err)
	}
	return &order, nil
}

// fulfillmentCovered reports whether every order line is fully covered by
// non-cancelled shipments.
func (s *CommerceService) fulfillmentCovered(ctx context.Context, orderID int64) (bool, error) {
	return fulfillmentCoveredTx(ctx, s.repo, orderID)
}

// fulfillmentCoveredTx is the transaction-scoped variant used inside locked
// order transactions so coverage reads serialize with shipment creation.
func fulfillmentCoveredTx(ctx context.Context, txRepo *repository.CommerceRepository, orderID int64) (bool, error) {
	lines, err := txRepo.GetOrderLines(ctx, orderID)
	if err != nil {
		return false, fmt.Errorf("get order lines: %w", err)
	}
	if len(lines) == 0 {
		return false, nil
	}
	shipped, err := txRepo.ShippedQuantities(ctx, orderID)
	if err != nil {
		return false, fmt.Errorf("get shipped quantities: %w", err)
	}
	for _, l := range lines {
		if shipped[l.ID] < l.Quantity {
			return false, nil
		}
	}
	return true, nil
}

// RecordPayment allocates a successful payment amount across the order's
// issued invoices oldest-first. Called by the payments module after an
// intent succeeds. Amounts beyond the outstanding balance are rejected.
func (s *CommerceService) RecordPayment(ctx context.Context, orderID, amountMinor int64) (int64, error) {
	if amountMinor <= 0 {
		return amountMinor, ErrInvalidQuantity
	}
	var remaining int64
	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		// Locked invoice reads: concurrent allocations serialize instead
		// of double-applying against the same outstanding balance.
		invoices, err := txRepo.ListInvoicesForUpdate(ctx, orderID)
		if err != nil {
			return fmt.Errorf("list invoices: %w", err)
		}
		var outstanding int64
		for _, inv := range invoices {
			if inv.Status == "issued" {
				outstanding += inv.BalanceMinor
			}
		}
		if amountMinor > outstanding {
			return ErrOverpayment
		}
		rem, err := txRepo.ApplyInvoicePayment(ctx, orderID, amountMinor)
		if err != nil {
			return fmt.Errorf("apply payment: %w", err)
		}
		remaining = rem
		return nil
	})
	if err != nil {
		return amountMinor, err
	}
	return remaining, nil
}

func (s *CommerceService) publishLifecycleEvent(ctx context.Context, buyerID int64, order *repository.Order) {
	if s.publisher == nil {
		return
	}
	var eventType string
	switch order.Status {
	case OrderStatusCancelled:
		eventType = events.EventOrderCancelled
	case OrderStatusShipped:
		eventType = events.EventOrderDispatched
	default:
		return
	}
	_ = s.publisher.Publish(ctx, eventType, events.NewEnvelope(
		eventType,
		events.OrderPlacedPayload{
			OrderCode:  order.Code,
			BuyerID:    buyerID,
			TotalMinor: order.TotalMinor,
			Currency:   order.Currency,
		},
		"",
	))
}
