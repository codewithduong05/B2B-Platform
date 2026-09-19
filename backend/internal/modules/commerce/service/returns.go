package service

// TASK-008: returns (financial only — completing a return creates credit
// notes but never touches inventory, by explicit scope decision) and credit
// notes (auto-applied oldest-first on approval, void reverses).

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/commerce/repository"
	"github.com/atlas-platform/backend/internal/modules/commerce/schema"
	"github.com/jackc/pgx/v5"
)

var (
	ErrReturnNotFound = errors.New("return request not found")
	ErrReturnState    = errors.New("return is not actionable")
	ErrReturnQuantity = errors.New("invalid return quantity")
	ErrCreditNotFound = errors.New("credit note not found")
	ErrCreditExceeds  = errors.New("credit exceeds outstanding balance")
)

const (
	ReturnStatusRequested = "requested"
	ReturnStatusApproved  = "approved"
	ReturnStatusRejected  = "rejected"
	ReturnStatusCompleted = "completed"
)

type ReturnLineInput struct {
	OrderLineID int64
	Quantity    int
}

func newReturnCode() string {
	return newPublicCode("ret_")
}

func newCreditCode(prefix string) string {
	return newPublicCode(prefix)
}

// RequestReturn opens a buyer return request. Only shipped/delivered orders
// qualify; quantities cannot exceed ordered minus already-returned.
func (s *CommerceService) RequestReturn(ctx context.Context, buyerID int64, orderCode string, items []ReturnLineInput, reason string) (*schema.ReturnResponse, error) {
	if reason == "" || len(items) == 0 {
		return nil, ErrInvalidQuantity
	}
	order, err := s.repo.GetOrderByCode(ctx, orderCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order: %w", err)
	}
	if order.BuyerID != buyerID {
		return nil, ErrOrderNotFound
	}
	if order.Status != OrderStatusShipped && order.Status != OrderStatusDelivered {
		return nil, ErrReturnState
	}

	lines, err := s.repo.GetOrderLines(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("get order lines: %w", err)
	}
	ordered := make(map[int64]repository.OrderLine)
	for _, l := range lines {
		ordered[l.ID] = l
	}
	returned, err := s.repo.ReturnedQuantities(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("returned quantities: %w", err)
	}
	for _, it := range items {
		ol, ok := ordered[it.OrderLineID]
		if !ok {
			return nil, ErrCartLineNotFound
		}
		if it.Quantity <= 0 || returned[it.OrderLineID]+it.Quantity > ol.Quantity {
			return nil, ErrReturnQuantity
		}
	}

	var req repository.ReturnRequest
	var createdLines []repository.ReturnLine
	err = s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		rr, err := txRepo.CreateReturnRequest(ctx, newReturnCode(), order.ID, buyerID, reason, &buyerID)
		if err != nil {
			return fmt.Errorf("create return: %w", err)
		}
		req = rr
		for _, it := range items {
			l, err := txRepo.CreateReturnLine(ctx, rr.ID, it.OrderLineID, it.Quantity)
			if err != nil {
				return fmt.Errorf("create return line: %w", err)
			}
			createdLines = append(createdLines, l)
		}
		if _, err := txRepo.CreateOrderHistory(ctx, order.ID, &order.Status, order.Status, &buyerID, "return requested: "+rr.Code); err != nil {
			return fmt.Errorf("record order history: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.toReturnResponse(ctx, req, createdLines)
}

// ApproveReturn moves requested → approved (locked; concurrent approves
// serialize and only the first acts).
func (s *CommerceService) ApproveReturn(ctx context.Context, actor int64, returnID int64) (*schema.ReturnResponse, error) {
	var req repository.ReturnRequest
	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		locked, err := txRepo.GetReturnByIDForUpdate(ctx, returnID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReturnNotFound
			}
			return fmt.Errorf("lock return: %w", err)
		}
		if locked.Status != ReturnStatusRequested {
			return ErrReturnState
		}
		u, err := txRepo.UpdateReturnStatus(ctx, locked.ID, ReturnStatusApproved, &actor)
		if err != nil {
			return fmt.Errorf("approve return: %w", err)
		}
		req = u
		order, err := txRepo.GetOrderByID(ctx, locked.OrderID)
		if err != nil {
			return fmt.Errorf("get order: %w", err)
		}
		if _, err := txRepo.CreateOrderHistory(ctx, order.ID, &order.Status, order.Status, &actor, "return approved: "+locked.Code); err != nil {
			return fmt.Errorf("record order history: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.loadReturnResponse(ctx, req.ID)
}

// RejectReturn moves requested → rejected with history.
func (s *CommerceService) RejectReturn(ctx context.Context, actor int64, returnID int64, reason string) (*schema.ReturnResponse, error) {
	var req repository.ReturnRequest
	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		locked, err := txRepo.GetReturnByIDForUpdate(ctx, returnID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReturnNotFound
			}
			return fmt.Errorf("lock return: %w", err)
		}
		if locked.Status != ReturnStatusRequested {
			return ErrReturnState
		}
		u, err := txRepo.UpdateReturnStatus(ctx, locked.ID, ReturnStatusRejected, &actor)
		if err != nil {
			return fmt.Errorf("reject return: %w", err)
		}
		req = u
		order, err := txRepo.GetOrderByID(ctx, locked.OrderID)
		if err != nil {
			return fmt.Errorf("get order: %w", err)
		}
		note := "return rejected: " + locked.Code
		if reason != "" {
			note += " (" + reason + ")"
		}
		if _, err := txRepo.CreateOrderHistory(ctx, order.ID, &order.Status, order.Status, &actor, note); err != nil {
			return fmt.Errorf("record order history: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.loadReturnResponse(ctx, req.ID)
}

// CompleteReturn moves approved → completed and creates the credit notes,
// auto-applied oldest-first. No inventory movement (scope decision).
func (s *CommerceService) CompleteReturn(ctx context.Context, actor int64, returnID int64) (*schema.ReturnResponse, error) {
	var req repository.ReturnRequest
	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		locked, err := txRepo.GetReturnByIDForUpdate(ctx, returnID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrReturnNotFound
			}
			return fmt.Errorf("lock return: %w", err)
		}
		if locked.Status != ReturnStatusApproved {
			return ErrReturnState
		}
		u, err := txRepo.UpdateReturnStatus(ctx, locked.ID, ReturnStatusCompleted, &actor)
		if err != nil {
			return fmt.Errorf("complete return: %w", err)
		}
		req = u
		order, err := txRepo.GetOrderByID(ctx, locked.OrderID)
		if err != nil {
			return fmt.Errorf("get order: %w", err)
		}
		if _, err := txRepo.CreateOrderHistory(ctx, order.ID, &order.Status, order.Status, &actor, "return completed: "+locked.Code); err != nil {
			return fmt.Errorf("record order history: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Credit the returned lines at their snapshotted order prices.
	lines, err := s.repo.GetReturnLines(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("get return lines: %w", err)
	}
	orderLines, err := s.repo.GetOrderLines(ctx, req.OrderID)
	if err != nil {
		return nil, fmt.Errorf("get order lines: %w", err)
	}
	byID := make(map[int64]repository.OrderLine)
	for _, l := range orderLines {
		byID[l.ID] = l
	}
	batch := newCreditCode("cnb_")
	for _, rl := range lines {
		ol, ok := byID[rl.OrderLineID]
		if !ok {
			return nil, ErrCartLineNotFound
		}
		amount := ol.UnitPriceMinor * int64(rl.Quantity)
		if _, err := s.applyCredit(ctx, req.OrderID, &req.ID, amount, orderCurrency(ctx, s, req.OrderID), "return "+req.Code, &actor, batch); err != nil {
			return nil, err
		}
	}
	return s.loadReturnResponse(ctx, req.ID)
}

func orderCurrency(ctx context.Context, s *CommerceService, orderID int64) string {
	o, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return "USD"
	}
	return o.Currency
}

// applyCredit creates per-invoice credit-note rows oldest-first inside one
// transaction, reducing issued balances. Amounts beyond outstanding are
// rejected (all-or-nothing).
func (s *CommerceService) applyCredit(ctx context.Context, orderID int64, returnID *int64, amountMinor int64, currency, reason string, actor *int64, batch string) ([]repository.CreditNote, error) {
	if amountMinor <= 0 {
		return nil, ErrInvalidQuantity
	}
	var notes []repository.CreditNote
	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		invoices, err := txRepo.ListInvoicesForUpdate(ctx, orderID)
		if err != nil {
			return fmt.Errorf("list invoices: %w", err)
		}
		// Oldest-first: ListInvoices returns newest-first; iterate reversed.
		remaining := amountMinor
		type allocation struct {
			invoiceID int64
			take      int64
		}
		var plan []allocation
		for i := len(invoices) - 1; i >= 0; i-- {
			if remaining <= 0 {
				break
			}
			inv := invoices[i]
			if inv.Status != "issued" || inv.BalanceMinor <= 0 {
				continue
			}
			take := remaining
			if take > inv.BalanceMinor {
				take = inv.BalanceMinor
			}
			plan = append(plan, allocation{inv.ID, take})
			remaining -= take
		}
		if remaining > 0 {
			return ErrCreditExceeds
		}
		for _, p := range plan {
			invoiceID := p.invoiceID
			cn, err := txRepo.CreateCreditNote(ctx, newCreditCode("cn_"), batch, orderID, &invoiceID, returnID, p.take, currency, reason, actor)
			if err != nil {
				return fmt.Errorf("create credit note: %w", err)
			}
			if _, err := txRepo.DecrementInvoiceBalance(ctx, p.invoiceID, p.take); err != nil {
				return fmt.Errorf("apply credit: %w", err)
			}
			notes = append(notes, cn)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return notes, nil
}

func (s *CommerceService) loadReturnResponse(ctx context.Context, returnID int64) (*schema.ReturnResponse, error) {
	req, err := s.repo.GetReturnByID(ctx, returnID)
	if err != nil {
		return nil, ErrReturnNotFound
	}
	lines, err := s.repo.GetReturnLines(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("get return lines: %w", err)
	}
	return s.toReturnResponse(ctx, req, lines)
}

func (s *CommerceService) toReturnResponse(ctx context.Context, req repository.ReturnRequest, lines []repository.ReturnLine) (*schema.ReturnResponse, error) {
	_ = ctx
	resp := &schema.ReturnResponse{
		Code: req.Code, OrderID: req.OrderID, Reason: req.Reason, Status: req.Status,
		RequestedBy: req.RequestedBy, DecidedBy: req.DecidedBy,
		CreatedAt: req.CreatedAt, UpdatedAt: req.UpdatedAt,
	}
	for _, l := range lines {
		resp.Lines = append(resp.Lines, schema.ReturnLineResponse{
			OrderLineID: l.OrderLineID, Quantity: l.Quantity,
		})
	}
	return resp, nil
}

// ListReturns lists return requests, optionally filtered by order or buyer.
func (s *CommerceService) ListReturns(ctx context.Context, orderID, buyerID int64, limit, offset int) ([]schema.ReturnResponse, error) {
	reqs, err := s.repo.ListReturns(ctx, orderID, buyerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list returns: %w", err)
	}
	out := make([]schema.ReturnResponse, 0, len(reqs))
	for _, req := range reqs {
		lines, err := s.repo.GetReturnLines(ctx, req.ID)
		if err != nil {
			return nil, fmt.Errorf("get return lines: %w", err)
		}
		resp, err := s.toReturnResponse(ctx, req, lines)
		if err != nil {
			return nil, err
		}
		out = append(out, *resp)
	}
	return out, nil
}

// CreateManualCredit creates an ad-hoc credit note (staff) applied
// oldest-first against the order's issued invoices.
func (s *CommerceService) CreateManualCredit(ctx context.Context, actor int64, orderID, amountMinor int64, reason string) ([]schema.CreditNoteResponse, error) {
	if reason == "" || amountMinor <= 0 {
		return nil, ErrInvalidQuantity
	}
	order, err := s.orderByID(ctx, orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order.Status == OrderStatusCancelled {
		return nil, ErrInvalidTransition
	}
	notes, err := s.applyCredit(ctx, order.ID, nil, amountMinor, order.Currency, reason, &actor, newCreditCode("cnb_"))
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.CreateOrderHistory(ctx, order.ID, &order.Status, order.Status, &actor, "credit issued: "+notes[0].BatchCode); err != nil {
		return nil, fmt.Errorf("record order history: %w", err)
	}
	return toCreditResponses(notes), nil
}

// VoidCreditNote voids one applied credit-note row, restoring its amount to
// the linked invoice (capped at the invoice total).
func (s *CommerceService) VoidCreditNote(ctx context.Context, actor int64, creditID int64) (*schema.CreditNoteResponse, error) {
	cn, err := s.repo.GetCreditNoteByID(ctx, creditID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCreditNotFound
		}
		return nil, fmt.Errorf("get credit note: %w", err)
	}
	if cn.Status != "applied" {
		return nil, ErrInvalidTransition
	}
	var updated repository.CreditNote
	err = s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		u, err := txRepo.VoidCreditNote(ctx, cn.ID)
		if err != nil {
			return fmt.Errorf("void credit note: %w", err)
		}
		updated = u
		if cn.InvoiceID != nil {
			if _, err := txRepo.RestoreInvoiceBalance(ctx, *cn.InvoiceID, cn.AmountMinor); err != nil {
				return fmt.Errorf("restore invoice balance: %w", err)
			}
		}
		order, err := txRepo.GetOrderByID(ctx, cn.OrderID)
		if err != nil {
			return fmt.Errorf("get order: %w", err)
		}
		if _, err := txRepo.CreateOrderHistory(ctx, order.ID, &order.Status, order.Status, &actor, "credit voided: "+cn.Code); err != nil {
			return fmt.Errorf("record order history: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := toCreditResponses([]repository.CreditNote{updated})
	return &out[0], nil
}

// ReissueInvoice voids an invoice and drafts a replacement linked via
// replaces_code. Only draft/issued invoices qualify.
func (s *CommerceService) ReissueInvoice(ctx context.Context, actor int64, invoiceID int64) (*schema.InvoiceResponse, error) {
	inv, err := s.invoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv.Status == InvoiceStatusVoid {
		return nil, ErrInvalidTransition
	}
	var replacement *schema.InvoiceResponse
	err = s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewCommerceRepositoryWithTx(tx)
		if _, err := txRepo.UpdateInvoiceStatus(ctx, inv.ID, InvoiceStatusVoid); err != nil {
			return fmt.Errorf("void invoice: %w", err)
		}
		repl, err := txRepo.CreateInvoiceReplacement(ctx, newInvoiceCode(), inv.OrderID, inv.SubtotalMinor, inv.TotalMinor, inv.Currency, inv.Code)
		if err != nil {
			return fmt.Errorf("create replacement: %w", err)
		}
		out := toInvoiceResponse(repl)
		replacement = out
		order, err := txRepo.GetOrderByID(ctx, inv.OrderID)
		if err != nil {
			return fmt.Errorf("get order: %w", err)
		}
		if _, err := txRepo.CreateOrderHistory(ctx, order.ID, &order.Status, order.Status, &actor, "invoice reissued: "+inv.Code+" -> "+repl.Code); err != nil {
			return fmt.Errorf("record order history: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return replacement, nil
}

// ListInvoicesMe returns the buyer's own invoices across their orders.
func (s *CommerceService) ListInvoicesMe(ctx context.Context, buyerID int64) ([]schema.InvoiceResponse, error) {
	orders, err := s.repo.ListOrdersByBuyer(ctx, buyerID)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	var out []schema.InvoiceResponse
	for _, o := range orders {
		invoices, err := s.repo.ListInvoices(ctx, o.ID, 0, 0)
		if err != nil {
			return nil, fmt.Errorf("list invoices: %w", err)
		}
		for _, inv := range invoices {
			out = append(out, *toInvoiceResponse(inv))
		}
	}
	if out == nil {
		out = []schema.InvoiceResponse{}
	}
	return out, nil
}

// GetInvoiceMe returns one invoice iff it belongs to the buyer's order.
func (s *CommerceService) GetInvoiceMe(ctx context.Context, buyerID int64, invoiceID int64) (*schema.InvoiceResponse, error) {
	inv, err := s.invoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	order, err := s.repo.GetOrderByID(ctx, inv.OrderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order.BuyerID != buyerID {
		return nil, ErrInvoiceNotFound
	}
	return toInvoiceResponse(inv), nil
}

// GetInvoiceMeByCode returns one invoice by public code iff it belongs to
// the buyer's order.
func (s *CommerceService) GetInvoiceMeByCode(ctx context.Context, buyerID int64, code string) (*schema.InvoiceResponse, error) {
	invoices, err := s.ListInvoicesMe(ctx, buyerID)
	if err != nil {
		return nil, err
	}
	for _, inv := range invoices {
		if inv.Code == code {
			out := inv
			return &out, nil
		}
	}
	return nil, ErrInvoiceNotFound
}

// ListCreditNotes lists credit notes for an order (staff path).
func (s *CommerceService) ListCreditNotes(ctx context.Context, orderID int64) ([]schema.CreditNoteResponse, error) {
	notes, err := s.repo.ListCreditNotes(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("list credit notes: %w", err)
	}
	return toCreditResponses(notes), nil
}

// OrderInvoices returns the buyer's invoices for one order code.
func (s *CommerceService) OrderInvoices(ctx context.Context, buyerID int64, orderCode string) ([]schema.InvoiceResponse, error) {
	order, err := s.repo.GetOrderByCode(ctx, orderCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order: %w", err)
	}
	if order.BuyerID != buyerID {
		return nil, ErrOrderNotFound
	}
	invoices, err := s.repo.ListInvoices(ctx, order.ID, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	out := make([]schema.InvoiceResponse, 0, len(invoices))
	for _, inv := range invoices {
		out = append(out, *toInvoiceResponse(inv))
	}
	return out, nil
}

// AgedDebtRow is one unpaid invoice balance with its age in days.
type AgedDebtRow struct {
	BuyerID     int64
	OrderID     int64
	OrderCode   string
	InvoiceCode string
	Balance     int64
	IssuedAt    time.Time
	DaysOld     int
	Bucket      string
}

func ageBucket(daysOld int) string {
	switch {
	case daysOld <= 30:
		return "current"
	case daysOld <= 60:
		return "31-60"
	case daysOld <= 90:
		return "61-90"
	default:
		return "90+"
	}
}

// AgedDebt lists unpaid issued-invoice balances with ageing buckets,
// optionally filtered by buyer. Reads the primary (no replica exists yet;
// M5 moves analytics to a replica).
func (s *CommerceService) AgedDebt(ctx context.Context, buyerID int64, now time.Time) ([]AgedDebtRow, error) {
	rows, err := s.repo.AgedDebtRows(ctx, buyerID)
	if err != nil {
		return nil, fmt.Errorf("aged debt rows: %w", err)
	}
	out := make([]AgedDebtRow, 0, len(rows))
	for _, r := range rows {
		days := int(now.Sub(r.IssuedAt).Hours() / 24)
		if days < 0 {
			days = 0
		}
		out = append(out, AgedDebtRow{
			BuyerID: r.BuyerID, OrderID: r.OrderID, OrderCode: r.OrderCode,
			InvoiceCode: r.InvoiceCode, Balance: r.Balance,
			IssuedAt: r.IssuedAt, DaysOld: days, Bucket: ageBucket(days),
		})
	}
	return out, nil
}

func toCreditResponses(notes []repository.CreditNote) []schema.CreditNoteResponse {
	out := make([]schema.CreditNoteResponse, 0, len(notes))
	for _, cn := range notes {
		out = append(out, schema.CreditNoteResponse{
			Code: cn.Code, BatchCode: cn.BatchCode, OrderID: cn.OrderID,
			InvoiceID: cn.InvoiceID, ReturnID: cn.ReturnID,
			AmountMinor: cn.AmountMinor, Currency: cn.Currency,
			Reason: cn.Reason, Status: cn.Status, Actor: cn.Actor,
			CreatedAt: cn.CreatedAt, UpdatedAt: cn.UpdatedAt,
		})
	}
	return out
}
