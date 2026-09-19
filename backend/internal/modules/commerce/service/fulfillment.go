package service

// TASK-006: shipments (partial fulfillment) and invoices (manual B2B
// issue/void). Shipment lines may partially cover an order; over-shipping a
// line is rejected. Order-level shipped status is coverage-gated in the
// lifecycle service, not here.

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

var ErrShipmentNotFound = errors.New("shipment not found")
var ErrInvoiceNotFound = errors.New("invoice not found")

const (
	ShipmentStatusPreparing  = "preparing"
	ShipmentStatusShipped    = "shipped"
	ShipmentStatusDelivered  = "delivered"
	ShipmentStatusCancelled  = "cancelled"
	InvoiceStatusDraft       = "draft"
	InvoiceStatusIssued      = "issued"
	InvoiceStatusVoid        = "void"
)

var validShipmentTransitions = map[string][]string{
	ShipmentStatusPreparing: {ShipmentStatusShipped, ShipmentStatusDelivered, ShipmentStatusCancelled},
	ShipmentStatusShipped:   {ShipmentStatusDelivered, ShipmentStatusCancelled},
	ShipmentStatusDelivered: {},
	ShipmentStatusCancelled: {},
}

type ShipmentLineInput struct {
	OrderLineID int64
	Quantity    int
}

func newShipmentCode() string {
	return newPublicCode("shp_")
}

func newInvoiceCode() string {
	return newPublicCode("inv_")
}

func (s *CommerceService) shipmentByID(ctx context.Context, id int64) (repository.Shipment, error) {
	sh, err := s.repo.GetShipmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository.Shipment{}, ErrShipmentNotFound
		}
		return repository.Shipment{}, fmt.Errorf("get shipment: %w", err)
	}
	return sh, nil
}

func (s *CommerceService) invoiceByID(ctx context.Context, id int64) (repository.Invoice, error) {
	inv, err := s.repo.GetInvoiceByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository.Invoice{}, ErrInvoiceNotFound
		}
		return repository.Invoice{}, fmt.Errorf("get invoice: %w", err)
	}
	return inv, nil
}

// AdminListShipments returns staff shipment summaries with the list total.
func (s *CommerceService) AdminListShipments(ctx context.Context, orderID int64, limit, offset int) ([]schema.ShipmentResponse, int, error) {
	shipments, err := s.repo.ListShipments(ctx, orderID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list shipments: %w", err)
	}
	total, err := s.repo.CountShipments(ctx, orderID)
	if err != nil {
		return nil, 0, fmt.Errorf("count shipments: %w", err)
	}
	out := make([]schema.ShipmentResponse, 0, len(shipments))
	for _, sh := range shipments {
		slines, err := s.repo.GetShipmentLines(ctx, sh.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("get shipment lines: %w", err)
		}
		olines, err := s.repo.GetOrderLines(ctx, sh.OrderID)
		if err != nil {
			return nil, 0, fmt.Errorf("get order lines: %w", err)
		}
		ordered := make(map[int64]repository.OrderLine)
		for _, l := range olines {
			ordered[l.ID] = l
		}
		out = append(out, s.toShipmentResponse(sh, slines, ordered))
	}
	return out, total, nil
}

// AdminListInvoices returns staff invoice summaries with the list total.
func (s *CommerceService) AdminListInvoices(ctx context.Context, orderID int64, limit, offset int) ([]schema.InvoiceResponse, int, error) {
	invoices, err := s.repo.ListInvoices(ctx, orderID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list invoices: %w", err)
	}
	total, err := s.repo.CountInvoices(ctx, orderID)
	if err != nil {
		return nil, 0, fmt.Errorf("count invoices: %w", err)
	}
	out := make([]schema.InvoiceResponse, 0, len(invoices))
	for _, inv := range invoices {
		out = append(out, *toInvoiceResponse(inv))
	}
	return out, total, nil
}

// CreateShipment records a (possibly partial) fulfillment of an order.
func (s *CommerceService) CreateShipment(ctx context.Context, actor int64, orderID int64, items []ShipmentLineInput, carrier, trackingCode *string) (*schema.ShipmentResponse, error) {
	order, err := s.orderByID(ctx, orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order.OnHold {
		return nil, ErrOrderOnHold
	}
	if order.Status == OrderStatusDelivered || order.Status == OrderStatusCancelled {
		return nil, ErrInvalidTransition
	}
	if len(items) == 0 {
		return nil, ErrInvalidQuantity
	}

	lines, err := s.repo.GetOrderLines(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("get order lines: %w", err)
	}
	ordered := make(map[int64]repository.OrderLine)
	for _, l := range lines {
		ordered[l.ID] = l
	}
	shipped, err := s.repo.ShippedQuantities(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("get shipped quantities: %w", err)
	}
	for _, it := range items {
		ol, ok := ordered[it.OrderLineID]
		if !ok {
			return nil, ErrCartLineNotFound
		}
		if it.Quantity <= 0 || shipped[it.OrderLineID]+it.Quantity > ol.Quantity {
			return nil, ErrInvalidQuantity
		}
	}

	var shipment repository.Shipment
	var createdLines []repository.ShipmentLine
	err = s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		// Locked re-validation: concurrent shipments serialize here, so a
		// second partial that fit before the first committed is rechecked
		// against committed coverage and rejected instead of over-shipping.
		locked, err := txRepo.GetOrderByIDForUpdate(ctx, order.ID)
		if err != nil {
			return fmt.Errorf("lock order: %w", err)
		}
		if locked.OnHold {
			return ErrOrderOnHold
		}
		if locked.Status == OrderStatusDelivered || locked.Status == OrderStatusCancelled {
			return ErrInvalidTransition
		}
		lockedLines, err := txRepo.GetOrderLines(ctx, order.ID)
		if err != nil {
			return fmt.Errorf("get order lines: %w", err)
		}
		lockedOrdered := make(map[int64]repository.OrderLine)
		for _, l := range lockedLines {
			lockedOrdered[l.ID] = l
		}
		lockedShipped, err := txRepo.ShippedQuantities(ctx, order.ID)
		if err != nil {
			return fmt.Errorf("get shipped quantities: %w", err)
		}
		for _, it := range items {
			ol, ok := lockedOrdered[it.OrderLineID]
			if !ok {
				return ErrCartLineNotFound
			}
			if it.Quantity <= 0 || lockedShipped[it.OrderLineID]+it.Quantity > ol.Quantity {
				return ErrOrderConflict
			}
		}
		sh, err := txRepo.CreateShipment(ctx, newShipmentCode(), order.ID, carrier, trackingCode, ShipmentStatusPreparing)
		if err != nil {
			return fmt.Errorf("create shipment: %w", err)
		}
		shipment = sh
		for _, it := range items {
			sl, err := txRepo.CreateShipmentLine(ctx, sh.ID, it.OrderLineID, it.Quantity)
			if err != nil {
				return fmt.Errorf("create shipment line: %w", err)
			}
			createdLines = append(createdLines, sl)
		}
		if _, err := txRepo.CreateOrderHistory(ctx, order.ID, &locked.Status, locked.Status, &actor, "shipment created: "+sh.Code); err != nil {
			return fmt.Errorf("record order history: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	resp := s.toShipmentResponse(shipment, createdLines, ordered)
	s.publishShipmentUpdated(ctx, &shipment)
	return &resp, nil
}

// UpdateShipment patches carrier/tracking and/or advances shipment status.
func (s *CommerceService) UpdateShipment(ctx context.Context, actor int64, shipmentID int64, carrier, trackingCode *string, status string) (*schema.ShipmentResponse, error) {
	shipment, err := s.shipmentByID(ctx, shipmentID)
	if err != nil {
		return nil, ErrShipmentNotFound
	}
	if status != "" && status != shipment.Status {
		ok := false
		for _, next := range validShipmentTransitions[shipment.Status] {
			if next == status {
				ok = true
				break
			}
		}
		if !ok {
			return nil, ErrInvalidTransition
		}
	} else {
		status = shipment.Status
	}

	var updated repository.Shipment
	err = s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		// Locked re-validation: a concurrent status change commits first,
		// so this attempt re-checks against the locked row instead of
		// blindly overwriting it.
		locked, err := txRepo.GetShipmentByIDForUpdate(ctx, shipment.ID)
		if err != nil {
			return fmt.Errorf("lock shipment: %w", err)
		}
		if status != locked.Status {
			ok := false
			for _, next := range validShipmentTransitions[locked.Status] {
				if next == status {
					ok = true
					break
				}
			}
			if !ok {
				return ErrOrderConflict
			}
		}
		u, err := txRepo.UpdateShipment(ctx, shipment.ID, carrier, trackingCode, status)
		if err != nil {
			return fmt.Errorf("update shipment: %w", err)
		}
		updated = u
		return nil
	})
	if err != nil {
		return nil, err
	}
	lines, err := s.repo.GetShipmentLines(ctx, updated.ID)
	if err != nil {
		return nil, fmt.Errorf("get shipment lines: %w", err)
	}
	orderLines, err := s.repo.GetOrderLines(ctx, updated.OrderID)
	if err != nil {
		return nil, fmt.Errorf("get order lines: %w", err)
	}
	ordered := make(map[int64]repository.OrderLine)
	for _, l := range orderLines {
		ordered[l.ID] = l
	}
	resp := s.toShipmentResponse(updated, lines, ordered)
	s.publishShipmentUpdated(ctx, &updated)
	return &resp, nil
}

func (s *CommerceService) toShipmentResponse(sh repository.Shipment, lines []repository.ShipmentLine, ordered map[int64]repository.OrderLine) schema.ShipmentResponse {
	resp := schema.ShipmentResponse{
		Code:         sh.Code,
		Status:       sh.Status,
		CreatedAt:    sh.CreatedAt,
		UpdatedAt:    sh.UpdatedAt,
	}
	if sh.Carrier != nil {
		resp.Carrier = *sh.Carrier
	}
	if sh.TrackingCode != nil {
		resp.TrackingCode = *sh.TrackingCode
	}
	for _, l := range lines {
		item := schema.ShipmentLineResponse{
			OrderLineID: l.OrderLineID,
			Quantity:    l.Quantity,
		}
		if ol, ok := ordered[l.OrderLineID]; ok {
			item.ProductCode = ol.ProductCode
			item.ProductName = ol.ProductName
		}
		resp.Lines = append(resp.Lines, item)
	}
	return resp
}

func (s *CommerceService) publishShipmentUpdated(ctx context.Context, shipment *repository.Shipment) {
	if s.publisher == nil {
		return
	}
	_ = s.publisher.Publish(ctx, events.EventShipmentUpdated, events.NewEnvelope(
		events.EventShipmentUpdated,
		map[string]interface{}{
			"shipment_code": shipment.Code,
			"order_id":      shipment.OrderID,
			"status":        shipment.Status,
		},
		"",
	))
}

// CreateInvoice drafts an invoice from the order totals.
func (s *CommerceService) CreateInvoice(ctx context.Context, actor int64, orderID int64) (*schema.InvoiceResponse, error) {
	order, err := s.orderByID(ctx, orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order.Status == OrderStatusCancelled {
		return nil, ErrInvalidTransition
	}
	inv, err := s.repo.CreateInvoice(ctx, newInvoiceCode(), order.ID, order.SubtotalMinor, order.TotalMinor, order.Currency)
	if err != nil {
		return nil, fmt.Errorf("create invoice: %w", err)
	}
	if _, err := s.repo.CreateOrderHistory(ctx, order.ID, &order.Status, order.Status, &actor, "invoice drafted: "+inv.Code); err != nil {
		return nil, fmt.Errorf("record order history: %w", err)
	}
	return toInvoiceResponse(inv), nil
}

// IssueInvoice moves a draft invoice to issued.
func (s *CommerceService) IssueInvoice(ctx context.Context, actor int64, invoiceID int64) (*schema.InvoiceResponse, error) {
	inv, err := s.invoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, ErrInvoiceNotFound
	}
	if inv.Status != InvoiceStatusDraft {
		return nil, ErrInvalidTransition
	}
	updated, err := s.repo.UpdateInvoiceStatus(ctx, inv.ID, InvoiceStatusIssued)
	if err != nil {
		return nil, fmt.Errorf("issue invoice: %w", err)
	}
	order, err := s.orderByID(ctx, inv.OrderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if _, err := s.repo.CreateOrderHistory(ctx, order.ID, &order.Status, order.Status, &actor, "invoice issued: "+inv.Code); err != nil {
		return nil, fmt.Errorf("record order history: %w", err)
	}
	return toInvoiceResponse(updated), nil
}

// VoidInvoice voids a draft or issued invoice. The row is kept (R10).
func (s *CommerceService) VoidInvoice(ctx context.Context, actor int64, invoiceID int64) (*schema.InvoiceResponse, error) {
	inv, err := s.invoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, ErrInvoiceNotFound
	}
	if inv.Status == InvoiceStatusVoid {
		return toInvoiceResponse(inv), nil
	}
	updated, err := s.repo.UpdateInvoiceStatus(ctx, inv.ID, InvoiceStatusVoid)
	if err != nil {
		return nil, fmt.Errorf("void invoice: %w", err)
	}
	order, err := s.orderByID(ctx, inv.OrderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if _, err := s.repo.CreateOrderHistory(ctx, order.ID, &order.Status, order.Status, &actor, "invoice voided: "+inv.Code); err != nil {
		return nil, fmt.Errorf("record order history: %w", err)
	}
	return toInvoiceResponse(updated), nil
}

func toInvoiceResponse(inv repository.Invoice) *schema.InvoiceResponse {
	out := &schema.InvoiceResponse{
		Code:          inv.Code,
		Status:        inv.Status,
		SubtotalMinor: inv.SubtotalMinor,
		TotalMinor:    inv.TotalMinor,
		BalanceMinor:  inv.BalanceMinor,
		Currency:      inv.Currency,
		CreatedAt:     inv.CreatedAt,
		UpdatedAt:     inv.UpdatedAt,
	}
	if inv.IssuedAt != nil {
		out.IssuedAt = inv.IssuedAt
	}
	return out
}
