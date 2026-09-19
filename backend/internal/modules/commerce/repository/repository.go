package repository

import (
	"context"
	"time"

	"github.com/atlas-platform/backend/internal/database"
)

type Cart struct {
	ID        int64
	Code      string
	BuyerID   int64
	Currency  string
	CreatedAt time.Time
	UpdatedAt time.Time
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
}

func NewCommerceRepository(db *database.DB) *CommerceRepository {
	return &CommerceRepository{db: db}
}

func (r *CommerceRepository) GetCartByBuyerID(ctx context.Context, buyerID int64) (Cart, error) {
	var c Cart
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, code, buyer_id, currency, created_at, updated_at
		FROM commerce.cart
		WHERE buyer_id = $1 AND deleted_at IS NULL
	`, buyerID).Scan(&c.ID, &c.Code, &c.BuyerID, &c.Currency, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *CommerceRepository) CreateCart(ctx context.Context, code string, buyerID int64, currency string) (Cart, error) {
	var c Cart
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO commerce.cart (code, buyer_id, currency)
		VALUES ($1, $2, $3)
		ON CONFLICT (buyer_id) DO UPDATE SET updated_at = NOW()
		RETURNING id, code, buyer_id, currency, created_at, updated_at
	`, code, buyerID, currency).Scan(&c.ID, &c.Code, &c.BuyerID, &c.Currency, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (r *CommerceRepository) GetCartLines(ctx context.Context, cartID int64) ([]CartLine, error) {
	rows, err := r.db.Pool.Query(ctx, `
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
	err := r.db.Pool.QueryRow(ctx, `
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
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, code, cart_id, product_id, supplier_id, unit_id, quantity, created_at, updated_at
		FROM commerce.cart_line
		WHERE code = $1 AND deleted_at IS NULL
	`, code).Scan(&l.ID, &l.Code, &l.CartID, &l.ProductID, &l.SupplierID, &l.UnitID, &l.Quantity, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

func (r *CommerceRepository) UpdateCartLineQuantity(ctx context.Context, id int64, quantity int) (CartLine, error) {
	var l CartLine
	err := r.db.Pool.QueryRow(ctx, `
		UPDATE commerce.cart_line
		SET quantity = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, cart_id, product_id, supplier_id, unit_id, quantity, created_at, updated_at
	`, id, quantity).Scan(&l.ID, &l.Code, &l.CartID, &l.ProductID, &l.SupplierID, &l.UnitID, &l.Quantity, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

func (r *CommerceRepository) DeleteCartLine(ctx context.Context, id int64) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE commerce.cart_line
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	return err
}
