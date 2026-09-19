package repository

// TASK-008: returns, credit notes, invoice reissue. conn() honors ambient
// transactions bound via NewCommerceRepositoryWithTx.

import (
	"context"
	"strconv"
	"time"
)

type ReturnRequest struct {
	ID          int64
	Code        string
	OrderID     int64
	BuyerID     int64
	Reason      string
	Status      string
	RequestedBy *int64
	DecidedBy   *int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ReturnLine struct {
	ID          int64
	ReturnID    int64
	OrderLineID int64
	Quantity    int
	CreatedAt   time.Time
}

type CreditNote struct {
	ID          int64
	Code        string
	BatchCode   string
	OrderID     int64
	InvoiceID   *int64
	ReturnID    *int64
	AmountMinor int64
	Currency    string
	Reason      string
	Status      string
	Actor       *int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r *CommerceRepository) CreateReturnRequest(ctx context.Context, code string, orderID, buyerID int64, reason string, requestedBy *int64) (ReturnRequest, error) {
	var rr ReturnRequest
	err := r.conn().QueryRow(ctx, `
		INSERT INTO commerce.return_request (code, order_id, buyer_id, reason, status, requested_by)
		VALUES ($1, $2, $3, $4, 'requested', $5)
		RETURNING id, code, order_id, buyer_id, reason, status, requested_by, decided_by, created_at, updated_at
	`, code, orderID, buyerID, reason, requestedBy).Scan(
		&rr.ID, &rr.Code, &rr.OrderID, &rr.BuyerID, &rr.Reason, &rr.Status,
		&rr.RequestedBy, &rr.DecidedBy, &rr.CreatedAt, &rr.UpdatedAt)
	return rr, err
}

func (r *CommerceRepository) CreateReturnLine(ctx context.Context, returnID, orderLineID int64, quantity int) (ReturnLine, error) {
	var l ReturnLine
	err := r.conn().QueryRow(ctx, `
		INSERT INTO commerce.return_line (return_id, order_line_id, quantity)
		VALUES ($1, $2, $3)
		RETURNING id, return_id, order_line_id, quantity, created_at
	`, returnID, orderLineID, quantity).Scan(
		&l.ID, &l.ReturnID, &l.OrderLineID, &l.Quantity, &l.CreatedAt)
	return l, err
}

func (r *CommerceRepository) GetReturnByID(ctx context.Context, id int64) (ReturnRequest, error) {
	var rr ReturnRequest
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, order_id, buyer_id, reason, status, requested_by, decided_by, created_at, updated_at
		FROM commerce.return_request
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&rr.ID, &rr.Code, &rr.OrderID, &rr.BuyerID, &rr.Reason, &rr.Status,
		&rr.RequestedBy, &rr.DecidedBy, &rr.CreatedAt, &rr.UpdatedAt)
	return rr, err
}

func (r *CommerceRepository) GetReturnByIDForUpdate(ctx context.Context, id int64) (ReturnRequest, error) {
	var rr ReturnRequest
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, order_id, buyer_id, reason, status, requested_by, decided_by, created_at, updated_at
		FROM commerce.return_request
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, id).Scan(&rr.ID, &rr.Code, &rr.OrderID, &rr.BuyerID, &rr.Reason, &rr.Status,
		&rr.RequestedBy, &rr.DecidedBy, &rr.CreatedAt, &rr.UpdatedAt)
	return rr, err
}

func (r *CommerceRepository) GetReturnLines(ctx context.Context, returnID int64) ([]ReturnLine, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT id, return_id, order_line_id, quantity, created_at
		FROM commerce.return_line
		WHERE return_id = $1
		ORDER BY id ASC
	`, returnID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ReturnLine
	for rows.Next() {
		var l ReturnLine
		if err := rows.Scan(&l.ID, &l.ReturnID, &l.OrderLineID, &l.Quantity, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// ReturnedQuantities sums approved/completed return lines per order line.
func (r *CommerceRepository) ReturnedQuantities(ctx context.Context, orderID int64) (map[int64]int, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT rl.order_line_id, COALESCE(SUM(rl.quantity), 0)
		FROM commerce.return_line rl
		JOIN commerce.return_request rr ON rr.id = rl.return_id
		WHERE rr.order_id = $1 AND rr.status IN ('approved', 'completed') AND rr.deleted_at IS NULL
		GROUP BY rl.order_line_id
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64]int)
	for rows.Next() {
		var lineID int64
		var qty int64
		if err := rows.Scan(&lineID, &qty); err != nil {
			return nil, err
		}
		out[lineID] = int(qty)
	}
	return out, rows.Err()
}

func (r *CommerceRepository) UpdateReturnStatus(ctx context.Context, id int64, status string, decidedBy *int64) (ReturnRequest, error) {
	var rr ReturnRequest
	err := r.conn().QueryRow(ctx, `
		UPDATE commerce.return_request
		SET status = $2::varchar, decided_by = COALESCE($3, decided_by), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, order_id, buyer_id, reason, status, requested_by, decided_by, created_at, updated_at
	`, id, status, decidedBy).Scan(&rr.ID, &rr.Code, &rr.OrderID, &rr.BuyerID, &rr.Reason, &rr.Status,
		&rr.RequestedBy, &rr.DecidedBy, &rr.CreatedAt, &rr.UpdatedAt)
	return rr, err
}

func (r *CommerceRepository) ListReturns(ctx context.Context, orderID, buyerID int64, limit, offset int) ([]ReturnRequest, error) {
	query := `
		SELECT id, code, order_id, buyer_id, reason, status, requested_by, decided_by, created_at, updated_at
		FROM commerce.return_request
		WHERE deleted_at IS NULL
	`
	args := []any{}
	nextParam := func() string {
		return "$" + strconv.Itoa(len(args))
	}
	if orderID > 0 {
		args = append(args, orderID)
		query += ` AND order_id = ` + nextParam()
	}
	if buyerID > 0 {
		args = append(args, buyerID)
		query += ` AND buyer_id = ` + nextParam()
	}
	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		if limit > 200 {
			limit = 200
		}
		args = append(args, limit)
		query += ` LIMIT ` + nextParam()
	}
	if offset > 0 {
		args = append(args, offset)
		query += ` OFFSET ` + nextParam()
	}
	rows, err := r.conn().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ReturnRequest
	for rows.Next() {
		var rr ReturnRequest
		if err := rows.Scan(&rr.ID, &rr.Code, &rr.OrderID, &rr.BuyerID, &rr.Reason, &rr.Status,
			&rr.RequestedBy, &rr.DecidedBy, &rr.CreatedAt, &rr.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, rr)
	}
	return out, rows.Err()
}

func (r *CommerceRepository) CreateCreditNote(ctx context.Context, code, batchCode string, orderID int64, invoiceID, returnID *int64, amountMinor int64, currency, reason string, actor *int64) (CreditNote, error) {
	var cn CreditNote
	err := r.conn().QueryRow(ctx, `
		INSERT INTO commerce.credit_note (code, batch_code, order_id, invoice_id, return_id, amount_minor, currency, reason, status, actor)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'applied', $9)
		RETURNING id, code, batch_code, order_id, invoice_id, return_id, amount_minor, currency, reason, status, actor, created_at, updated_at
	`, code, batchCode, orderID, invoiceID, returnID, amountMinor, currency, reason, actor).Scan(
		&cn.ID, &cn.Code, &cn.BatchCode, &cn.OrderID, &cn.InvoiceID, &cn.ReturnID,
		&cn.AmountMinor, &cn.Currency, &cn.Reason, &cn.Status, &cn.Actor, &cn.CreatedAt, &cn.UpdatedAt)
	return cn, err
}

func (r *CommerceRepository) GetCreditNoteByID(ctx context.Context, id int64) (CreditNote, error) {
	var cn CreditNote
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, batch_code, order_id, invoice_id, return_id, amount_minor, currency, reason, status, actor, created_at, updated_at
		FROM commerce.credit_note
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&cn.ID, &cn.Code, &cn.BatchCode, &cn.OrderID, &cn.InvoiceID, &cn.ReturnID,
		&cn.AmountMinor, &cn.Currency, &cn.Reason, &cn.Status, &cn.Actor, &cn.CreatedAt, &cn.UpdatedAt)
	return cn, err
}

func (r *CommerceRepository) ListCreditNotes(ctx context.Context, orderID int64) ([]CreditNote, error) {
	query := `
		SELECT id, code, batch_code, order_id, invoice_id, return_id, amount_minor, currency, reason, status, actor, created_at, updated_at
		FROM commerce.credit_note
		WHERE deleted_at IS NULL
	`
	args := []any{}
	if orderID > 0 {
		args = append(args, orderID)
		query += ` AND order_id = $1`
	}
	query += ` ORDER BY created_at DESC`
	rows, err := r.conn().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CreditNote
	for rows.Next() {
		var cn CreditNote
		if err := rows.Scan(&cn.ID, &cn.Code, &cn.BatchCode, &cn.OrderID, &cn.InvoiceID, &cn.ReturnID,
			&cn.AmountMinor, &cn.Currency, &cn.Reason, &cn.Status, &cn.Actor, &cn.CreatedAt, &cn.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, cn)
	}
	return out, rows.Err()
}

func (r *CommerceRepository) VoidCreditNote(ctx context.Context, id int64) (CreditNote, error) {
	var cn CreditNote
	err := r.conn().QueryRow(ctx, `
		UPDATE commerce.credit_note
		SET status = 'void', updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, batch_code, order_id, invoice_id, return_id, amount_minor, currency, reason, status, actor, created_at, updated_at
	`, id).Scan(&cn.ID, &cn.Code, &cn.BatchCode, &cn.OrderID, &cn.InvoiceID, &cn.ReturnID,
		&cn.AmountMinor, &cn.Currency, &cn.Reason, &cn.Status, &cn.Actor, &cn.CreatedAt, &cn.UpdatedAt)
	return cn, err
}

// CreateInvoiceReplacement drafts a new invoice linked to a voided one.
func (r *CommerceRepository) CreateInvoiceReplacement(ctx context.Context, code string, orderID, subtotalMinor, totalMinor int64, currency, replacesCode string) (Invoice, error) {
	var inv Invoice
	err := r.conn().QueryRow(ctx, `
		INSERT INTO commerce.invoice (code, order_id, subtotal_minor, total_minor, balance_minor, currency, status, replaces_code)
		VALUES ($1, $2, $3, $4, $4, $5, 'draft', $6)
		RETURNING id, code, order_id, subtotal_minor, total_minor, balance_minor, currency, status, issued_at, replaces_code, created_at, updated_at
	`, code, orderID, subtotalMinor, totalMinor, currency, replacesCode).Scan(
		&inv.ID, &inv.Code, &inv.OrderID, &inv.SubtotalMinor, &inv.TotalMinor,
		&inv.BalanceMinor, &inv.Currency, &inv.Status, &inv.IssuedAt, &inv.ReplacesCode, &inv.CreatedAt, &inv.UpdatedAt)
	return inv, err
}
