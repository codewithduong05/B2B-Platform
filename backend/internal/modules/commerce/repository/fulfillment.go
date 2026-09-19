package repository

// TASK-006: order lifecycle, shipments, invoices. All methods honor the
// repository conn() so they run inside the caller's transaction when bound
// via NewCommerceRepositoryWithTx.

import (
	"context"
	"strconv"
	"time"
)

type Shipment struct {
	ID           int64
	Code         string
	OrderID      int64
	Carrier      *string
	TrackingCode *string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ShipmentLine struct {
	ID          int64
	ShipmentID  int64
	OrderLineID int64
	Quantity    int
	CreatedAt   time.Time
}

type Invoice struct {
	ID            int64
	Code          string
	OrderID       int64
	SubtotalMinor int64
	TotalMinor    int64
	BalanceMinor  int64
	Currency      string
	Status        string
	IssuedAt      *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func scanOrderFull(o *Order, cartID **int64, cartCode **string) []any {
	return []any{&o.ID, &o.Code, &o.BuyerID, &o.SupplierID, cartID,
		cartCode, &o.Currency, &o.SubtotalMinor, &o.DiscountsMinor,
		&o.TotalMinor, &o.Status, &o.OnHold, &o.HoldReason, &o.IdemKey,
		&o.PlacedAt, &o.CreatedAt, &o.UpdatedAt}
}

const orderFullColumns = `id, code, buyer_id, supplier_id, cart_id, cart_code, currency, subtotal_minor, discounts_minor, total_minor, status, on_hold, hold_reason, idem_key, placed_at, created_at, updated_at`

func scanOrderInto(o *Order, cartID *int64, cartCode *string) {
	o.CartID = cartID
	if cartCode != nil {
		o.CartCode = *cartCode
	}
}

// GetOrderByIDForUpdate loads one order holding a row lock, serializing
// concurrent transitions against the same order.
func (r *CommerceRepository) GetOrderByIDForUpdate(ctx context.Context, id int64) (Order, error) {
	var o Order
	var cartID *int64
	var cartCode *string
	err := r.conn().QueryRow(ctx, `
		SELECT `+orderFullColumns+`
		FROM commerce."order"
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, id).Scan(scanOrderFull(&o, &cartID, &cartCode)...)
	scanOrderInto(&o, cartID, cartCode)
	return o, err
}

// GetOrderByID loads one order by integer PK (staff/admin path).
func (r *CommerceRepository) GetOrderByID(ctx context.Context, id int64) (Order, error) {
	var o Order
	var cartID *int64
	var cartCode *string
	err := r.conn().QueryRow(ctx, `
		SELECT `+orderFullColumns+`
		FROM commerce."order"
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(scanOrderFull(&o, &cartID, &cartCode)...)
	scanOrderInto(&o, cartID, cartCode)
	return o, err
}

// UpdateOrderStatus sets the order status, returning the updated row.
func (r *CommerceRepository) UpdateOrderStatus(ctx context.Context, id int64, status string) (Order, error) {
	var o Order
	var cartID *int64
	var cartCode *string
	err := r.conn().QueryRow(ctx, `
		UPDATE commerce."order"
		SET status = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+orderFullColumns+`
	`, id, status).Scan(scanOrderFull(&o, &cartID, &cartCode)...)
	scanOrderInto(&o, cartID, cartCode)
	return o, err
}

// SetOrderHold toggles the hold flag with a reason.
func (r *CommerceRepository) SetOrderHold(ctx context.Context, id int64, onHold bool, reason *string) (Order, error) {
	var o Order
	var cartID *int64
	var cartCode *string
	err := r.conn().QueryRow(ctx, `
		UPDATE commerce."order"
		SET on_hold = $2, hold_reason = $3, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+orderFullColumns+`
	`, id, onHold, reason).Scan(scanOrderFull(&o, &cartID, &cartCode)...)
	scanOrderInto(&o, cartID, cartCode)
	return o, err
}

// SetOrderIdemKey records the checkout idempotency key on the order so a
// later cancellation can derive and release the attempt's reservations.
func (r *CommerceRepository) SetOrderIdemKey(ctx context.Context, id int64, key string) error {
	_, err := r.conn().Exec(ctx, `
		UPDATE commerce."order"
		SET idem_key = $2, updated_at = NOW()
		WHERE id = $1
	`, id, key)
	return err
}

// CountOrdersAdmin counts orders for the admin list envelope.
func (r *CommerceRepository) CountOrdersAdmin(ctx context.Context, status string) (int, error) {
	query := `SELECT COUNT(*) FROM commerce."order" WHERE deleted_at IS NULL`
	args := []any{}
	if status != "" {
		args = append(args, status)
		query += ` AND status = $1::commerce.order_status`
	}
	var n int
	err := r.conn().QueryRow(ctx, query, args...).Scan(&n)
	return n, err
}

// ListOrdersAdmin lists orders with optional status filter (staff path).
func (r *CommerceRepository) ListOrdersAdmin(ctx context.Context, status string, limit, offset int) ([]Order, error) {
	query := `
		SELECT ` + orderFullColumns + `
		FROM commerce."order"
		WHERE deleted_at IS NULL
	`
	args := []any{}
	if status != "" {
		args = append(args, status)
		query += ` AND status = $1::commerce.order_status`
	}
	query += ` ORDER BY placed_at DESC`
	if limit > 0 {
		args = append(args, limit)
		query += ` LIMIT $` + strconv.Itoa(len(args))
	}
	if offset > 0 {
		args = append(args, offset)
		query += ` OFFSET $` + strconv.Itoa(len(args))
	}
	rows, err := r.conn().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Order
	for rows.Next() {
		var o Order
		var cartID *int64
		var cartCode *string
		if err := rows.Scan(scanOrderFull(&o, &cartID, &cartCode)...); err != nil {
			return nil, err
		}
		scanOrderInto(&o, cartID, cartCode)
		out = append(out, o)
	}
	return out, rows.Err()
}

func (r *CommerceRepository) CreateShipment(ctx context.Context, code string, orderID int64, carrier, trackingCode *string, status string) (Shipment, error) {
	var s Shipment
	err := r.conn().QueryRow(ctx, `
		INSERT INTO commerce.shipment (code, order_id, carrier, tracking_code, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, code, order_id, carrier, tracking_code, status, created_at, updated_at
	`, code, orderID, carrier, trackingCode, status).Scan(
		&s.ID, &s.Code, &s.OrderID, &s.Carrier, &s.TrackingCode, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func (r *CommerceRepository) CreateShipmentLine(ctx context.Context, shipmentID, orderLineID int64, quantity int) (ShipmentLine, error) {
	var l ShipmentLine
	err := r.conn().QueryRow(ctx, `
		INSERT INTO commerce.shipment_line (shipment_id, order_line_id, quantity)
		VALUES ($1, $2, $3)
		RETURNING id, shipment_id, order_line_id, quantity, created_at
	`, shipmentID, orderLineID, quantity).Scan(
		&l.ID, &l.ShipmentID, &l.OrderLineID, &l.Quantity, &l.CreatedAt)
	return l, err
}
func (r *CommerceRepository) GetShipmentByID(ctx context.Context, id int64) (Shipment, error) {
	var s Shipment
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, order_id, carrier, tracking_code, status, created_at, updated_at
		FROM commerce.shipment
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&s.ID, &s.Code, &s.OrderID, &s.Carrier, &s.TrackingCode, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

// GetShipmentByIDForUpdate loads one shipment holding a row lock,
// serializing concurrent status transitions on the same shipment.
func (r *CommerceRepository) GetShipmentByIDForUpdate(ctx context.Context, id int64) (Shipment, error) {
	var s Shipment
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, order_id, carrier, tracking_code, status, created_at, updated_at
		FROM commerce.shipment
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, id).Scan(&s.ID, &s.Code, &s.OrderID, &s.Carrier, &s.TrackingCode, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func (r *CommerceRepository) GetShipmentsByOrder(ctx context.Context, orderID int64) ([]Shipment, error) {	rows, err := r.conn().Query(ctx, `
		SELECT id, code, order_id, carrier, tracking_code, status, created_at, updated_at
		FROM commerce.shipment
		WHERE order_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Shipment
	for rows.Next() {
		var s Shipment
		if err := rows.Scan(&s.ID, &s.Code, &s.OrderID, &s.Carrier, &s.TrackingCode, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *CommerceRepository) GetShipmentLines(ctx context.Context, shipmentID int64) ([]ShipmentLine, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT id, shipment_id, order_line_id, quantity, created_at
		FROM commerce.shipment_line
		WHERE shipment_id = $1
		ORDER BY id ASC
	`, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ShipmentLine
	for rows.Next() {
		var l ShipmentLine
		if err := rows.Scan(&l.ID, &l.ShipmentID, &l.OrderLineID, &l.Quantity, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// ShippedQuantities returns shipped (non-cancelled shipments) qty per order line.
func (r *CommerceRepository) ShippedQuantities(ctx context.Context, orderID int64) (map[int64]int, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT sl.order_line_id, COALESCE(SUM(sl.quantity), 0)
		FROM commerce.shipment_line sl
		JOIN commerce.shipment s ON s.id = sl.shipment_id
		WHERE s.order_id = $1 AND s.status != 'cancelled' AND s.deleted_at IS NULL
		GROUP BY sl.order_line_id
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

func (r *CommerceRepository) UpdateShipment(ctx context.Context, id int64, carrier, trackingCode *string, status string) (Shipment, error) {
	var s Shipment
	err := r.conn().QueryRow(ctx, `
		UPDATE commerce.shipment
		SET carrier = COALESCE($2, carrier),
		    tracking_code = COALESCE($3, tracking_code),
		    status = $4,
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, order_id, carrier, tracking_code, status, created_at, updated_at
	`, id, carrier, trackingCode, status).Scan(
		&s.ID, &s.Code, &s.OrderID, &s.Carrier, &s.TrackingCode, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func (r *CommerceRepository) CreateInvoice(ctx context.Context, code string, orderID, subtotalMinor, totalMinor int64, currency string) (Invoice, error) {
	var inv Invoice
	err := r.conn().QueryRow(ctx, `
		INSERT INTO commerce.invoice (code, order_id, subtotal_minor, total_minor, balance_minor, currency, status)
		VALUES ($1, $2, $3, $4, $4, $5, 'draft')
		RETURNING id, code, order_id, subtotal_minor, total_minor, balance_minor, currency, status, issued_at, created_at, updated_at
	`, code, orderID, subtotalMinor, totalMinor, currency).Scan(
		&inv.ID, &inv.Code, &inv.OrderID, &inv.SubtotalMinor, &inv.TotalMinor,
		&inv.BalanceMinor, &inv.Currency, &inv.Status, &inv.IssuedAt, &inv.CreatedAt, &inv.UpdatedAt)
	return inv, err
}

func (r *CommerceRepository) GetInvoiceByID(ctx context.Context, id int64) (Invoice, error) {
	var inv Invoice
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, order_id, subtotal_minor, total_minor, balance_minor, currency, status, issued_at, created_at, updated_at
		FROM commerce.invoice
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&inv.ID, &inv.Code, &inv.OrderID, &inv.SubtotalMinor, &inv.TotalMinor,
		&inv.BalanceMinor, &inv.Currency, &inv.Status, &inv.IssuedAt, &inv.CreatedAt, &inv.UpdatedAt)
	return inv, err
}

func (r *CommerceRepository) ListInvoices(ctx context.Context, orderID int64, limit, offset int) ([]Invoice, error) {
	return r.listInvoices(ctx, orderID, limit, offset, false)
}

// ListInvoicesForUpdate lists an order's invoices holding row locks, so a
// concurrent payment allocation serializes instead of double-applying.
func (r *CommerceRepository) ListInvoicesForUpdate(ctx context.Context, orderID int64) ([]Invoice, error) {
	invoices, err := r.listInvoices(ctx, orderID, 0, 0, true)
	if err != nil {
		return nil, err
	}
	return invoices, nil
}

func (r *CommerceRepository) listInvoices(ctx context.Context, orderID int64, limit, offset int, forUpdate bool) ([]Invoice, error) {
	query := `
		SELECT id, code, order_id, subtotal_minor, total_minor, balance_minor, currency, status, issued_at, created_at, updated_at
		FROM commerce.invoice
		WHERE deleted_at IS NULL
	`
	args := []any{}
	if orderID > 0 {
		args = append(args, orderID)
		query += ` AND order_id = $1`
	}
	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		args = append(args, limit)
		query += ` LIMIT $` + strconv.Itoa(len(args))
	}
	if offset > 0 {
		args = append(args, offset)
		query += ` OFFSET $` + strconv.Itoa(len(args))
	}
	if forUpdate {
		query += ` FOR UPDATE`
	}
	rows, err := r.conn().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Invoice
	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(&inv.ID, &inv.Code, &inv.OrderID, &inv.SubtotalMinor, &inv.TotalMinor,
			&inv.BalanceMinor, &inv.Currency, &inv.Status, &inv.IssuedAt, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (r *CommerceRepository) UpdateInvoiceStatus(ctx context.Context, id int64, status string) (Invoice, error) {
	var inv Invoice
	err := r.conn().QueryRow(ctx, `
		UPDATE commerce.invoice
		SET status = $2::varchar,
		    issued_at = CASE WHEN $2::varchar = 'issued' THEN COALESCE(issued_at, NOW()) ELSE issued_at END,
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, order_id, subtotal_minor, total_minor, balance_minor, currency, status, issued_at, created_at, updated_at
	`, id, status).Scan(&inv.ID, &inv.Code, &inv.OrderID, &inv.SubtotalMinor, &inv.TotalMinor,
		&inv.BalanceMinor, &inv.Currency, &inv.Status, &inv.IssuedAt, &inv.CreatedAt, &inv.UpdatedAt)
	return inv, err
}

// RestoreInvoiceBalance credits an amount back onto an invoice balance,
// capped at the invoice total. Used by refund application.
func (r *CommerceRepository) RestoreInvoiceBalance(ctx context.Context, id, amountMinor int64) (Invoice, error) {
	var inv Invoice
	err := r.conn().QueryRow(ctx, `
		UPDATE commerce.invoice
		SET balance_minor = LEAST(total_minor, balance_minor + $2), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, order_id, subtotal_minor, total_minor, balance_minor, currency, status, issued_at, created_at, updated_at
	`, id, amountMinor).Scan(&inv.ID, &inv.Code, &inv.OrderID, &inv.SubtotalMinor, &inv.TotalMinor,
		&inv.BalanceMinor, &inv.Currency, &inv.Status, &inv.IssuedAt, &inv.CreatedAt, &inv.UpdatedAt)
	return inv, err
}

// ApplyInvoicePayment reduces invoice balances oldest-first, returning the
// unallocated remainder. Caller must run inside a transaction for atomicity.
func (r *CommerceRepository) ApplyInvoicePayment(ctx context.Context, orderID, amountMinor int64) (int64, error) {
	invoices, err := r.ListInvoices(ctx, orderID, 0, 0)
	if err != nil {
		return amountMinor, err
	}
	remaining := amountMinor
	// ListInvoices returns newest-first; apply oldest-first.
	for i := len(invoices) - 1; i >= 0; i-- {
		if remaining <= 0 {
			break
		}
		inv := invoices[i]
		if inv.Status != "issued" || inv.BalanceMinor <= 0 {
			continue
		}
		apply := remaining
		if apply > inv.BalanceMinor {
			apply = inv.BalanceMinor
		}
		if _, err := r.conn().Exec(ctx, `
			UPDATE commerce.invoice
			SET balance_minor = balance_minor - $2, updated_at = NOW()
			WHERE id = $1
		`, inv.ID, apply); err != nil {
			return remaining, err
		}
		remaining -= apply
	}
	return remaining, nil
}

// ListShipments lists shipments, optionally filtered by order.
func (r *CommerceRepository) ListShipments(ctx context.Context, orderID int64, limit, offset int) ([]Shipment, error) {
	query := `
		SELECT id, code, order_id, carrier, tracking_code, status, created_at, updated_at
		FROM commerce.shipment
		WHERE deleted_at IS NULL
	`
	args := []any{}
	if orderID > 0 {
		args = append(args, orderID)
		query += ` AND order_id = $` + strconv.Itoa(len(args))
	}
	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		if limit > 200 {
			limit = 200
		}
		args = append(args, limit)
		query += ` LIMIT $` + strconv.Itoa(len(args))
	}
	if offset > 0 {
		args = append(args, offset)
		query += ` OFFSET $` + strconv.Itoa(len(args))
	}
	rows, err := r.conn().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Shipment
	for rows.Next() {
		var s Shipment
		if err := rows.Scan(&s.ID, &s.Code, &s.OrderID, &s.Carrier, &s.TrackingCode, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// CountShipments counts shipments for the admin list envelope.
func (r *CommerceRepository) CountShipments(ctx context.Context, orderID int64) (int, error) {
	query := `SELECT COUNT(*) FROM commerce.shipment WHERE deleted_at IS NULL`
	args := []any{}
	if orderID > 0 {
		args = append(args, orderID)
		query += ` AND order_id = $1`
	}
	var n int
	err := r.conn().QueryRow(ctx, query, args...).Scan(&n)
	return n, err
}

// CountInvoices counts invoices for the admin list envelope.
func (r *CommerceRepository) CountInvoices(ctx context.Context, orderID int64) (int, error) {
	query := `SELECT COUNT(*) FROM commerce.invoice WHERE deleted_at IS NULL`
	args := []any{}
	if orderID > 0 {
		args = append(args, orderID)
		query += ` AND order_id = $1`
	}
	var n int
	err := r.conn().QueryRow(ctx, query, args...).Scan(&n)
	return n, err
}
