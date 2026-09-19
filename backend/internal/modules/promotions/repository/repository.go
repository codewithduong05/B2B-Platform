package repository

// Promotions repository (raw pgx, same convention as commerce). conn()
// honors an ambient transaction bound via NewPromotionRepositoryWithTx,
// which the checkout path uses to redeem atomically with order creation.

import (
	"context"
	"strconv"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PromotionRepository struct {
	db *database.DB
	tx *database.Tx
}

func NewPromotionRepository(db *database.DB) *PromotionRepository {
	return &PromotionRepository{db: db}
}

func NewPromotionRepositoryWithTx(tx *database.Tx) *PromotionRepository {
	return &PromotionRepository{tx: tx}
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (r *PromotionRepository) conn() querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db.Pool
}

type Promotion struct {
	ID             int64
	Code           string
	Name           string
	Kind           string
	ValueMinor     int64
	Currency       string
	Status         string
	ValidFrom      *time.Time
	ValidTo        *time.Time
	MaxRedemptions *int
	RedeemedCount  int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Redemption struct {
	ID          int64
	PromotionID int64
	BuyerID     int64
	OrderID     int64
	AmountMinor int64
	Currency    string
	CreatedAt   time.Time
}

const promotionColumns = `id, code, name, kind, value_minor, currency, status, valid_from, valid_to, max_redemptions, redeemed_count, created_at, updated_at`

func scanPromotion(p *Promotion, validFrom, validTo **time.Time, maxRedemptions **int) []any {
	return []any{&p.ID, &p.Code, &p.Name, &p.Kind, &p.ValueMinor, &p.Currency,
		&p.Status, validFrom, validTo, maxRedemptions, &p.RedeemedCount, &p.CreatedAt, &p.UpdatedAt}
}

func fillPromotion(p *Promotion, validFrom, validTo *time.Time, maxRedemptions *int) {
	p.ValidFrom = validFrom
	p.ValidTo = validTo
	p.MaxRedemptions = maxRedemptions
}

func (r *PromotionRepository) CreatePromotion(ctx context.Context, code, name, kind string, valueMinor int64, currency string, validFrom, validTo *time.Time, maxRedemptions *int) (Promotion, error) {
	var p Promotion
	var vf, vt *time.Time
	var mr *int
	err := r.conn().QueryRow(ctx, `
		INSERT INTO promotions.promotion (code, name, kind, value_minor, currency, status, valid_from, valid_to, max_redemptions)
		VALUES ($1, $2, $3, $4, $5, 'draft', $6, $7, $8)
		RETURNING `+promotionColumns+`
	`, code, name, kind, valueMinor, currency, validFrom, validTo, maxRedemptions).Scan(
		scanPromotion(&p, &vf, &vt, &mr)...)
	fillPromotion(&p, vf, vt, mr)
	return p, err
}

func (r *PromotionRepository) GetPromotionByID(ctx context.Context, id int64) (Promotion, error) {
	var p Promotion
	var vf, vt *time.Time
	var mr *int
	err := r.conn().QueryRow(ctx, `
		SELECT `+promotionColumns+`
		FROM promotions.promotion
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(scanPromotion(&p, &vf, &vt, &mr)...)
	fillPromotion(&p, vf, vt, mr)
	return p, err
}

func (r *PromotionRepository) GetPromotionByCode(ctx context.Context, code string) (Promotion, error) {
	var p Promotion
	var vf, vt *time.Time
	var mr *int
	err := r.conn().QueryRow(ctx, `
		SELECT `+promotionColumns+`
		FROM promotions.promotion
		WHERE code = $1 AND deleted_at IS NULL
	`, code).Scan(scanPromotion(&p, &vf, &vt, &mr)...)
	fillPromotion(&p, vf, vt, mr)
	return p, err
}

func (r *PromotionRepository) ListPromotions(ctx context.Context, status string, limit, offset int) ([]Promotion, error) {
	query := `SELECT ` + promotionColumns + ` FROM promotions.promotion WHERE deleted_at IS NULL`
	args := []any{}
	if status != "" {
		args = append(args, status)
		query += ` AND status = $` + strconv.Itoa(len(args)) + `::varchar`
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

	var out []Promotion
	for rows.Next() {
		var p Promotion
		var vf, vt *time.Time
		var mr *int
		if err := rows.Scan(scanPromotion(&p, &vf, &vt, &mr)...); err != nil {
			return nil, err
		}
		fillPromotion(&p, vf, vt, mr)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PromotionRepository) CountPromotions(ctx context.Context, status string) (int, error) {
	query := `SELECT COUNT(*) FROM promotions.promotion WHERE deleted_at IS NULL`
	args := []any{}
	if status != "" {
		args = append(args, status)
		query += ` AND status = $1::varchar`
	}
	var n int
	err := r.conn().QueryRow(ctx, query, args...).Scan(&n)
	return n, err
}

type UpdatePromotionParams struct {
	Name           *string
	ValueMinor     *int64
	ValidFrom      **time.Time
	ValidTo        **time.Time
	MaxRedemptions **int
	Status         *string
}

// UpdatePromotion applies a partial update; nil fields are untouched.
// Double-pointer time/int fields distinguish absent from explicit NULL.
func (r *PromotionRepository) UpdatePromotion(ctx context.Context, id int64, params UpdatePromotionParams) (Promotion, error) {
	setClause := `updated_at = NOW()`
	args := []any{id}
	if params.Name != nil {
		args = append(args, *params.Name)
		setClause += `, name = $` + strconv.Itoa(len(args))
	}
	if params.ValueMinor != nil {
		args = append(args, *params.ValueMinor)
		setClause += `, value_minor = $` + strconv.Itoa(len(args))
	}
	if params.ValidFrom != nil {
		args = append(args, *params.ValidFrom)
		setClause += `, valid_from = $` + strconv.Itoa(len(args))
	}
	if params.ValidTo != nil {
		args = append(args, *params.ValidTo)
		setClause += `, valid_to = $` + strconv.Itoa(len(args))
	}
	if params.MaxRedemptions != nil {
		args = append(args, *params.MaxRedemptions)
		setClause += `, max_redemptions = $` + strconv.Itoa(len(args))
	}
	if params.Status != nil {
		args = append(args, *params.Status)
		setClause += `, status = $` + strconv.Itoa(len(args)) + `::varchar`
	}
	var p Promotion
	var vf, vt *time.Time
	var mr *int
	err := r.conn().QueryRow(ctx, `
		UPDATE promotions.promotion SET `+setClause+`
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+promotionColumns+`
	`, args...).Scan(scanPromotion(&p, &vf, &vt, &mr)...)
	if err != nil {
		return Promotion{}, err
	}
	fillPromotion(&p, vf, vt, mr)
	return p, nil
}

// HasRedeemed reports whether the buyer already consumed the promotion.
func (r *PromotionRepository) HasRedeemed(ctx context.Context, buyerID, promotionID int64) (bool, error) {
	var n int
	err := r.conn().QueryRow(ctx, `
		SELECT COUNT(*) FROM promotions.voucher_redemption
		WHERE buyer_id = $1 AND promotion_id = $2
	`, buyerID, promotionID).Scan(&n)
	return n > 0, err
}

// ConsumeBudget atomically consumes one budget unit, guarded by status,
// validity window, and max_redemptions. Exactly-once per checkout: callers
// invoke it once per order set, then record one row per split order.
// Concurrent checkouts serialize on the row; losers get pgx.ErrNoRows.
func (r *PromotionRepository) ConsumeBudget(ctx context.Context, promotionID int64, now time.Time) (Promotion, error) {
	var updated Promotion
	var vf, vt *time.Time
	var mr *int
	err := r.conn().QueryRow(ctx, `
		UPDATE promotions.promotion
		SET redeemed_count = redeemed_count + 1, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		  AND status = 'published'
		  AND (valid_from IS NULL OR valid_from <= $2)
		  AND (valid_to IS NULL OR valid_to >= $2)
		  AND (max_redemptions IS NULL OR redeemed_count < max_redemptions)
		RETURNING `+promotionColumns+`
	`, promotionID, now).Scan(scanPromotion(&updated, &vf, &vt, &mr)...)
	if err != nil {
		return Promotion{}, err
	}
	fillPromotion(&updated, vf, vt, mr)
	return updated, nil
}

// CreateRedemption records one order's discount share. Unique per
// (buyer, promotion, order): a retried insert fails instead of duplicating.
func (r *PromotionRepository) CreateRedemption(ctx context.Context, promotionID, buyerID, orderID, amountMinor int64, currency string) (Redemption, error) {
	var red Redemption
	err := r.conn().QueryRow(ctx, `
		INSERT INTO promotions.voucher_redemption (promotion_id, buyer_id, order_id, amount_minor, currency)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, promotion_id, buyer_id, order_id, amount_minor, currency, created_at
	`, promotionID, buyerID, orderID, amountMinor, currency).Scan(
		&red.ID, &red.PromotionID, &red.BuyerID, &red.OrderID, &red.AmountMinor, &red.Currency, &red.CreatedAt)
	return red, err
}

// TryRedeem is the single-line convenience path (consume + one row).
// Prefer ConsumeBudget + CreateRedemption for split flows.
func (r *PromotionRepository) TryRedeem(ctx context.Context, promotionID, buyerID, orderID, amountMinor int64, currency string, now time.Time) (Redemption, error) {
	if _, err := r.ConsumeBudget(ctx, promotionID, now); err != nil {
		return Redemption{}, err
	}
	return r.CreateRedemption(ctx, promotionID, buyerID, orderID, amountMinor, currency)
}

// Report aggregates redeemed counts and total discount cost per promotion.
func (r *PromotionRepository) Report(ctx context.Context) ([]ReportRow, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT p.code, p.name, p.status, p.max_redemptions,
		       COUNT(r.id) AS redeemed,
		       COALESCE(SUM(r.amount_minor), 0) AS cost
		FROM promotions.promotion p
		LEFT JOIN promotions.voucher_redemption r ON r.promotion_id = p.id
		WHERE p.deleted_at IS NULL
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ReportRow
	for rows.Next() {
		var row ReportRow
		if err := rows.Scan(&row.Code, &row.Name, &row.Status, &row.MaxRedemptions, &row.Redeemed, &row.Cost); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

type ReportRow struct {
	Code           string
	Name           string
	Status         string
	MaxRedemptions *int
	Redeemed       int
	Cost           int64
}
