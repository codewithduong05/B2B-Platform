package repository

import (
	"context"
	"errors"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Cart struct {
	ID          int64
	Code        string
	BuyerID     int64
	Currency    string
	VoucherCode *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CartLine struct {
	ID         int64
	Code       string
	CartID     int64
	ProductID  int64
	SupplierID int64
	UnitID     int64
	Quantity   int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CommerceRepository struct {
	db *database.DB
	tx *database.Tx
}

func NewCommerceRepository(db *database.DB) *CommerceRepository {
	return &CommerceRepository{db: db}
}

// NewCommerceRepositoryWithTx binds the repository to an ongoing transaction.
// database.Tx embeds pgx.Tx, so all queries below run inside that transaction.
func NewCommerceRepositoryWithTx(tx *database.Tx) *CommerceRepository {
	return &CommerceRepository{tx: tx}
}

// querier is satisfied by both *pgxpool.Pool and database.Tx (via pgx.Tx).
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (r *CommerceRepository) conn() querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db.Pool
}

func (r *CommerceRepository) GetCartByBuyerID(ctx context.Context, buyerID int64) (Cart, error) {
	var c Cart
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, buyer_id, currency, voucher_code, created_at, updated_at
		FROM commerce.cart
		WHERE buyer_id = $1 AND deleted_at IS NULL
	`, buyerID).Scan(&c.ID, &c.Code, &c.BuyerID, &c.Currency, &c.VoucherCode, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *CommerceRepository) CreateCart(ctx context.Context, code string, buyerID int64, currency string) (Cart, error) {
	var c Cart
	err := r.conn().QueryRow(ctx, `
		INSERT INTO commerce.cart (code, buyer_id, currency)
		VALUES ($1, $2, $3)
		ON CONFLICT (buyer_id) DO UPDATE SET updated_at = NOW()
		RETURNING id, code, buyer_id, currency, voucher_code, created_at, updated_at
	`, code, buyerID, currency).Scan(&c.ID, &c.Code, &c.BuyerID, &c.Currency, &c.VoucherCode, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

// SetCartVoucher applies (or replaces) the cart-level voucher code. Last
// write wins: exactly one voucher per cart (no stacking).
func (r *CommerceRepository) SetCartVoucher(ctx context.Context, cartID int64, code string) (Cart, error) {
	var c Cart
	err := r.conn().QueryRow(ctx, `
		UPDATE commerce.cart
		SET voucher_code = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, buyer_id, currency, voucher_code, created_at, updated_at
	`, cartID, code).Scan(&c.ID, &c.Code, &c.BuyerID, &c.Currency, &c.VoucherCode, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

// ClearCartVoucher removes the applied voucher (post-checkout consumption).
func (r *CommerceRepository) ClearCartVoucher(ctx context.Context, cartID int64) error {
	_, err := r.conn().Exec(ctx, `
		UPDATE commerce.cart
		SET voucher_code = NULL, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, cartID)
	return err
}

func (r *CommerceRepository) GetCartLines(ctx context.Context, cartID int64) ([]CartLine, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT id, code, cart_id, product_id, supplier_id, unit_id, quantity, created_at, updated_at
		FROM commerce.cart_line
		WHERE cart_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []CartLine
	for rows.Next() {
		var l CartLine
		if err := rows.Scan(&l.ID, &l.Code, &l.CartID, &l.ProductID, &l.SupplierID, &l.UnitID, &l.Quantity, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

func (r *CommerceRepository) UpsertCartLine(ctx context.Context, code string, cartID, productID, supplierID, unitID int64, quantity int) (CartLine, error) {
	var l CartLine
	// Arbiter matches the partial unique index
	// uq_commerce_cart_line_cart_product_active, so re-adding a product
	// whose previous line was soft-deleted inserts a fresh visible row
	// instead of accumulating onto the deleted one.
	err := r.conn().QueryRow(ctx, `
		INSERT INTO commerce.cart_line (code, cart_id, product_id, supplier_id, unit_id, quantity)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (cart_id, product_id) WHERE deleted_at IS NULL DO UPDATE
		SET quantity = commerce.cart_line.quantity + EXCLUDED.quantity, updated_at = NOW()
		RETURNING id, code, cart_id, product_id, supplier_id, unit_id, quantity, created_at, updated_at
	`, code, cartID, productID, supplierID, unitID, quantity).Scan(&l.ID, &l.Code, &l.CartID, &l.ProductID, &l.SupplierID, &l.UnitID, &l.Quantity, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

func (r *CommerceRepository) GetCartLineByCode(ctx context.Context, code string) (CartLine, error) {
	var l CartLine
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, cart_id, product_id, supplier_id, unit_id, quantity, created_at, updated_at
		FROM commerce.cart_line
		WHERE code = $1 AND deleted_at IS NULL
	`, code).Scan(&l.ID, &l.Code, &l.CartID, &l.ProductID, &l.SupplierID, &l.UnitID, &l.Quantity, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

func (r *CommerceRepository) UpdateCartLineQuantity(ctx context.Context, id int64, quantity int) (CartLine, error) {
	var l CartLine
	err := r.conn().QueryRow(ctx, `
		UPDATE commerce.cart_line
		SET quantity = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, cart_id, product_id, supplier_id, unit_id, quantity, created_at, updated_at
	`, id, quantity).Scan(&l.ID, &l.Code, &l.CartID, &l.ProductID, &l.SupplierID, &l.UnitID, &l.Quantity, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

func (r *CommerceRepository) DeleteCartLine(ctx context.Context, id int64) error {
	_, err := r.conn().Exec(ctx, `
		UPDATE commerce.cart_line
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	return err
}

// LockCartByBuyerID loads the buyer's active cart holding a row lock.
// It serializes concurrent checkouts against the same cart.
func (r *CommerceRepository) LockCartByBuyerID(ctx context.Context, buyerID int64) (Cart, error) {
	var c Cart
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, buyer_id, currency, voucher_code, created_at, updated_at
		FROM commerce.cart
		WHERE buyer_id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, buyerID).Scan(&c.ID, &c.Code, &c.BuyerID, &c.Currency, &c.VoucherCode, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

// ClearCartLinesByIDs soft-deletes exactly the given lines (post-checkout
// consumption). Clearing by ID — not by cart — guarantees a line added
// concurrently with checkout is never silently swallowed: it is either part
// of the validated snapshot (and cleared) or survives in the cart.
func (r *CommerceRepository) ClearCartLinesByIDs(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.conn().Exec(ctx, `
		UPDATE commerce.cart_line
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = ANY($1) AND deleted_at IS NULL
	`, ids)
	return err
}

// GetCartLinesForUpdate re-reads a cart's active lines holding row locks,
// so concurrent line updates serialize against the checkout transaction
// instead of interleaving with order creation.
func (r *CommerceRepository) GetCartLinesForUpdate(ctx context.Context, cartID int64) ([]CartLine, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT id, code, cart_id, product_id, supplier_id, unit_id, quantity, created_at, updated_at
		FROM commerce.cart_line
		WHERE cart_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
		FOR UPDATE
	`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []CartLine
	for rows.Next() {
		var l CartLine
		if err := rows.Scan(&l.ID, &l.Code, &l.CartID, &l.ProductID, &l.SupplierID, &l.UnitID, &l.Quantity, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

type Order struct {
	ID             int64
	Code           string
	BuyerID        int64
	SupplierID     int64
	CartID         *int64
	CartCode       string
	Currency       string
	SubtotalMinor  int64
	DiscountsMinor int64
	TotalMinor     int64
	Status         string
	OnHold         bool
	HoldReason     *string
	IdemKey        *string
	PlacedAt       time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type OrderLine struct {
	ID              int64
	Code            string
	OrderID         int64
	ProductID       int64
	SupplierID      int64
	UnitID          int64
	Quantity        int
	UnitPriceMinor  int64
	TotalPriceMinor int64
	Currency        string
	ProductCode     string
	ProductName     string
	UnitCode        string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type OrderHistory struct {
	ID         int64
	OrderID    int64
	FromStatus *string
	ToStatus   string
	Actor      *int64
	Reason     string
	CreatedAt  time.Time
}

type IdempotencyClaim struct {
	ID          int64
	BuyerID     int64
	Key         string
	PayloadHash string
	Status      string
	Result      []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r *CommerceRepository) CreateOrder(ctx context.Context, code string, buyerID, supplierID int64, cartID int64, cartCode, currency string, subtotalMinor, discountsMinor, totalMinor int64) (Order, error) {
	var o Order
	var cartIDPtr *int64
	var cartCodePtr *string
	err := r.conn().QueryRow(ctx, `
		INSERT INTO commerce."order" (code, buyer_id, supplier_id, cart_id, cart_code, currency, subtotal_minor, discounts_minor, total_minor, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'placed')
		RETURNING id, code, buyer_id, supplier_id, cart_id, cart_code, currency, subtotal_minor, discounts_minor, total_minor, status, placed_at, created_at, updated_at
	`, code, buyerID, supplierID, cartID, cartCode, currency, subtotalMinor, discountsMinor, totalMinor).Scan(
		&o.ID, &o.Code, &o.BuyerID, &o.SupplierID, &cartIDPtr, &cartCodePtr,
		&o.Currency, &o.SubtotalMinor, &o.DiscountsMinor, &o.TotalMinor,
		&o.Status, &o.PlacedAt, &o.CreatedAt, &o.UpdatedAt)
	if cartIDPtr != nil {
		o.CartID = cartIDPtr
	}
	if cartCodePtr != nil {
		o.CartCode = *cartCodePtr
	}
	return o, err
}

func (r *CommerceRepository) CreateOrderLine(ctx context.Context, code string, orderID, productID, supplierID, unitID int64, quantity int, unitPriceMinor, totalPriceMinor int64, currency, productCode, productName, unitCode string) (OrderLine, error) {
	var l OrderLine
	err := r.conn().QueryRow(ctx, `
		INSERT INTO commerce.order_line (code, order_id, product_id, supplier_id, unit_id, quantity, unit_price_minor, total_price_minor, currency, product_code, product_name, unit_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, code, order_id, product_id, supplier_id, unit_id, quantity, unit_price_minor, total_price_minor, currency, product_code, product_name, unit_code, created_at, updated_at
	`, code, orderID, productID, supplierID, unitID, quantity, unitPriceMinor, totalPriceMinor, currency, productCode, productName, unitCode).Scan(
		&l.ID, &l.Code, &l.OrderID, &l.ProductID, &l.SupplierID, &l.UnitID,
		&l.Quantity, &l.UnitPriceMinor, &l.TotalPriceMinor, &l.Currency,
		&l.ProductCode, &l.ProductName, &l.UnitCode, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

func (r *CommerceRepository) CreateOrderHistory(ctx context.Context, orderID int64, fromStatus *string, toStatus string, actor *int64, reason string) (OrderHistory, error) {
	var h OrderHistory
	err := r.conn().QueryRow(ctx, `
		INSERT INTO commerce.order_history (order_id, from_status, to_status, actor, reason)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, order_id, from_status, to_status, actor, reason, created_at
	`, orderID, fromStatus, toStatus, actor, reason).Scan(
		&h.ID, &h.OrderID, &h.FromStatus, &h.ToStatus, &h.Actor, &h.Reason, &h.CreatedAt)
	return h, err
}

// ClaimIdempotency inserts a pending claim for (buyer, key). created=true
// means this caller owns the attempt; created=false returns the existing row.
// If the conflicting row vanishes between the insert and the re-read (a
// concurrent failure path deleted it), the claim is retried: the key is free
// again, so INSERT may now succeed. Bounded; a residual miss surfaces as
// pgx.ErrNoRows for the caller to map.
func (r *CommerceRepository) ClaimIdempotency(ctx context.Context, buyerID int64, key, payloadHash string) (IdempotencyClaim, bool, error) {
	for attempt := 0; attempt < 3; attempt++ {
		var c IdempotencyClaim
		err := r.conn().QueryRow(ctx, `
			INSERT INTO commerce.checkout_idempotency (buyer_id, idem_key, payload_hash, status)
			VALUES ($1, $2, $3, 'pending')
			ON CONFLICT (buyer_id, idem_key) DO NOTHING
			RETURNING id, buyer_id, idem_key, payload_hash, status, result, created_at, updated_at
		`, buyerID, key, payloadHash).Scan(
			&c.ID, &c.BuyerID, &c.Key, &c.PayloadHash, &c.Status, &c.Result, &c.CreatedAt, &c.UpdatedAt)
		if err == nil {
			return c, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return IdempotencyClaim{}, false, err
		}
		existing, err := r.GetIdempotency(ctx, buyerID, key)
		if err == nil {
			return existing, false, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return IdempotencyClaim{}, false, err
		}
	}
	return IdempotencyClaim{}, false, pgx.ErrNoRows
}

func (r *CommerceRepository) GetIdempotency(ctx context.Context, buyerID int64, key string) (IdempotencyClaim, error) {
	var c IdempotencyClaim
	err := r.conn().QueryRow(ctx, `
		SELECT id, buyer_id, idem_key, payload_hash, status, result, created_at, updated_at
		FROM commerce.checkout_idempotency
		WHERE buyer_id = $1 AND idem_key = $2
	`, buyerID, key).Scan(
		&c.ID, &c.BuyerID, &c.Key, &c.PayloadHash, &c.Status, &c.Result, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *CommerceRepository) CompleteIdempotency(ctx context.Context, id int64, resultJSON []byte) error {
	_, err := r.conn().Exec(ctx, `
		UPDATE commerce.checkout_idempotency
		SET status = 'completed', result = $2, updated_at = NOW()
		WHERE id = $1
	`, id, resultJSON)
	return err
}

func (r *CommerceRepository) DeleteIdempotency(ctx context.Context, id int64) error {
	_, err := r.conn().Exec(ctx, `
		DELETE FROM commerce.checkout_idempotency
		WHERE id = $1
	`, id)
	return err
}

func (r *CommerceRepository) ListOrdersByBuyer(ctx context.Context, buyerID int64) ([]Order, error) {
	return r.listOrdersWhere(ctx, `buyer_id = $1`, buyerID)
}

// ListOrdersBySupplier lists a catalog supplier's directed orders (portal).
func (r *CommerceRepository) ListOrdersBySupplier(ctx context.Context, supplierID int64) ([]Order, error) {
	return r.listOrdersWhere(ctx, `supplier_id = $1`, supplierID)
}

func (r *CommerceRepository) listOrdersWhere(ctx context.Context, cond string, arg int64) ([]Order, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT id, code, buyer_id, supplier_id, cart_id, cart_code, currency, subtotal_minor, discounts_minor, total_minor, status, placed_at, created_at, updated_at
		FROM commerce."order"
		WHERE `+cond+` AND deleted_at IS NULL
		ORDER BY placed_at DESC
	`, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var cartID *int64
		var cartCode *string
		if err := rows.Scan(&o.ID, &o.Code, &o.BuyerID, &o.SupplierID, &cartID,
			&cartCode, &o.Currency, &o.SubtotalMinor, &o.DiscountsMinor,
			&o.TotalMinor, &o.Status, &o.PlacedAt, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		o.CartID = cartID
		if cartCode != nil {
			o.CartCode = *cartCode
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *CommerceRepository) GetOrderByCode(ctx context.Context, code string) (Order, error) {
	var o Order
	var cartID *int64
	var cartCode *string
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, buyer_id, supplier_id, cart_id, cart_code, currency, subtotal_minor, discounts_minor, total_minor, status, placed_at, created_at, updated_at
		FROM commerce."order"
		WHERE code = $1 AND deleted_at IS NULL
	`, code).Scan(&o.ID, &o.Code, &o.BuyerID, &o.SupplierID, &cartID,
		&cartCode, &o.Currency, &o.SubtotalMinor, &o.DiscountsMinor,
		&o.TotalMinor, &o.Status, &o.PlacedAt, &o.CreatedAt, &o.UpdatedAt)
	if err == nil {
		o.CartID = cartID
		if cartCode != nil {
			o.CartCode = *cartCode
		}
	}
	return o, err
}

func (r *CommerceRepository) GetOrderLines(ctx context.Context, orderID int64) ([]OrderLine, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT id, code, order_id, product_id, supplier_id, unit_id, quantity, unit_price_minor, total_price_minor, currency, product_code, product_name, unit_code, created_at, updated_at
		FROM commerce.order_line
		WHERE order_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []OrderLine
	for rows.Next() {
		var l OrderLine
		if err := rows.Scan(&l.ID, &l.Code, &l.OrderID, &l.ProductID, &l.SupplierID,
			&l.UnitID, &l.Quantity, &l.UnitPriceMinor, &l.TotalPriceMinor, &l.Currency,
			&l.ProductCode, &l.ProductName, &l.UnitCode, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

func (r *CommerceRepository) GetOrderHistory(ctx context.Context, orderID int64) ([]OrderHistory, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT id, order_id, from_status, to_status, actor, reason, created_at
		FROM commerce.order_history
		WHERE order_id = $1
		ORDER BY created_at ASC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []OrderHistory
	for rows.Next() {
		var h OrderHistory
		if err := rows.Scan(&h.ID, &h.OrderID, &h.FromStatus, &h.ToStatus, &h.Actor, &h.Reason, &h.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
