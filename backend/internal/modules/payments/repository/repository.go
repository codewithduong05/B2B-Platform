package repository

// Payments repository (raw pgx, same convention as commerce). conn() honors
// an ambient transaction bound via NewPaymentRepositoryWithTx.

import (
	"context"
	"strconv"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PaymentRepository struct {
	db *database.DB
	tx *database.Tx
}

func NewPaymentRepository(db *database.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func NewPaymentRepositoryWithTx(tx *database.Tx) *PaymentRepository {
	return &PaymentRepository{tx: tx}
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (r *PaymentRepository) conn() querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db.Pool
}

type PaymentMethod struct {
	ID          int64
	Code        string
	BuyerID     int64
	MethodType  string
	DisplayName string
	IsActive    bool
	CreatedAt   time.Time
}

type PaymentIntent struct {
	ID              int64
	Code            string
	BuyerID         int64
	OrderID         int64
	OrderCode       string
	PaymentMethodID *int64
	MethodCode      *string
	AmountMinor     int64
	Currency        string
	Status          string
	IdemKey         string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type PaymentAttempt struct {
	ID         int64
	IntentID   int64
	Result     string
	GatewayRef *string
	Note       string
	RecordedBy *int64
	CreatedAt  time.Time
}

func (r *PaymentRepository) CreateMethod(ctx context.Context, code string, buyerID int64, methodType, displayName string) (PaymentMethod, error) {
	var m PaymentMethod
	err := r.conn().QueryRow(ctx, `
		INSERT INTO payments.payment_method (code, buyer_id, method_type, display_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, code, buyer_id, method_type, display_name, is_active, created_at
	`, code, buyerID, methodType, displayName).Scan(
		&m.ID, &m.Code, &m.BuyerID, &m.MethodType, &m.DisplayName, &m.IsActive, &m.CreatedAt)
	return m, err
}

func (r *PaymentRepository) ListMethods(ctx context.Context, buyerID int64) ([]PaymentMethod, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT id, code, buyer_id, method_type, display_name, is_active, created_at
		FROM payments.payment_method
		WHERE buyer_id = $1 AND deleted_at IS NULL AND is_active
		ORDER BY created_at ASC
	`, buyerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PaymentMethod
	for rows.Next() {
		var m PaymentMethod
		if err := rows.Scan(&m.ID, &m.Code, &m.BuyerID, &m.MethodType, &m.DisplayName, &m.IsActive, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PaymentRepository) GetMethodByCode(ctx context.Context, buyerID int64, code string) (PaymentMethod, error) {
	var m PaymentMethod
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, buyer_id, method_type, display_name, is_active, created_at
		FROM payments.payment_method
		WHERE buyer_id = $1 AND code = $2 AND deleted_at IS NULL AND is_active
	`, buyerID, code).Scan(&m.ID, &m.Code, &m.BuyerID, &m.MethodType, &m.DisplayName, &m.IsActive, &m.CreatedAt)
	return m, err
}

func scanIntent(i *PaymentIntent, methodID **int64, methodCode **string) []any {
	return []any{&i.ID, &i.Code, &i.BuyerID, &i.OrderID, &i.OrderCode, methodID,
		methodCode, &i.AmountMinor, &i.Currency, &i.Status, &i.IdemKey, &i.CreatedAt, &i.UpdatedAt}
}

const intentColumns = `pi.id, pi.code, pi.buyer_id, pi.order_id, pi.order_code, pi.payment_method_id, pm.code AS method_code, pi.amount_minor, pi.currency, pi.status, pi.idem_key, pi.created_at, pi.updated_at`

const intentFrom = `FROM payments.payment_intent pi LEFT JOIN payments.payment_method pm ON pm.id = pi.payment_method_id`

func (r *PaymentRepository) CreateIntent(ctx context.Context, code string, buyerID, orderID int64, orderCode string, methodID *int64, amountMinor int64, currency, idemKey string) (PaymentIntent, error) {
	var i PaymentIntent
	var gotMethodID *int64
	err := r.conn().QueryRow(ctx, `
		INSERT INTO payments.payment_intent (code, buyer_id, order_id, order_code, payment_method_id, amount_minor, currency, status, idem_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'requires_action', $8)
		ON CONFLICT (buyer_id, idem_key) DO NOTHING
		RETURNING id, code, buyer_id, order_id, order_code, payment_method_id, amount_minor, currency, status, idem_key, created_at, updated_at
	`, code, buyerID, orderID, orderCode, methodID, amountMinor, currency, idemKey).Scan(
		&i.ID, &i.Code, &i.BuyerID, &i.OrderID, &i.OrderCode, &gotMethodID,
		&i.AmountMinor, &i.Currency, &i.Status, &i.IdemKey, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return PaymentIntent{}, err
	}
	i.PaymentMethodID = gotMethodID
	if gotMethodID != nil {
		if m, merr := r.GetMethodByID(ctx, *gotMethodID); merr == nil {
			mc := m.Code
			i.MethodCode = &mc
		}
	}
	return i, nil
}

func (r *PaymentRepository) GetMethodByID(ctx context.Context, id int64) (PaymentMethod, error) {
	var m PaymentMethod
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, buyer_id, method_type, display_name, is_active, created_at
		FROM payments.payment_method
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&m.ID, &m.Code, &m.BuyerID, &m.MethodType, &m.DisplayName, &m.IsActive, &m.CreatedAt)
	return m, err
}

func (r *PaymentRepository) GetIntentByKey(ctx context.Context, buyerID int64, idemKey string) (PaymentIntent, error) {
	var i PaymentIntent
	var methodID *int64
	var methodCode *string
	err := r.conn().QueryRow(ctx, `
		SELECT `+intentColumns+`
		`+intentFrom+`
		WHERE pi.buyer_id = $1 AND pi.idem_key = $2 AND pi.deleted_at IS NULL
	`, buyerID, idemKey).Scan(scanIntent(&i, &methodID, &methodCode)...)
	i.PaymentMethodID = methodID
	i.MethodCode = methodCode
	return i, err
}

func (r *PaymentRepository) GetIntentByCode(ctx context.Context, code string) (PaymentIntent, error) {
	var i PaymentIntent
	var methodID *int64
	var methodCode *string
	err := r.conn().QueryRow(ctx, `
		SELECT `+intentColumns+`
		`+intentFrom+`
		WHERE pi.code = $1 AND pi.deleted_at IS NULL
	`, code).Scan(scanIntent(&i, &methodID, &methodCode)...)
	i.PaymentMethodID = methodID
	i.MethodCode = methodCode
	return i, err
}

func (r *PaymentRepository) GetIntentByID(ctx context.Context, id int64) (PaymentIntent, error) {
	var i PaymentIntent
	var methodID *int64
	var methodCode *string
	err := r.conn().QueryRow(ctx, `
		SELECT `+intentColumns+`
		`+intentFrom+`
		WHERE pi.id = $1 AND pi.deleted_at IS NULL
	`, id).Scan(scanIntent(&i, &methodID, &methodCode)...)
	i.PaymentMethodID = methodID
	i.MethodCode = methodCode
	return i, err
}

func (r *PaymentRepository) GetIntentByIDForUpdate(ctx context.Context, id int64) (PaymentIntent, error) {
	var i PaymentIntent
	var methodID *int64
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, buyer_id, order_id, order_code, payment_method_id, amount_minor, currency, status, idem_key, created_at, updated_at
		FROM payments.payment_intent
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, id).Scan(&i.ID, &i.Code, &i.BuyerID, &i.OrderID, &i.OrderCode, &methodID,
		&i.AmountMinor, &i.Currency, &i.Status, &i.IdemKey, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return PaymentIntent{}, err
	}
	i.PaymentMethodID = methodID
	if methodID != nil {
		if m, merr := r.GetMethodByID(ctx, *methodID); merr == nil {
			mc := m.Code
			i.MethodCode = &mc
		}
	}
	return i, nil
}

// CountIntents counts intents for the admin list envelope.
func (r *PaymentRepository) CountIntents(ctx context.Context, buyerID int64, status string) (int, error) {
	query := `SELECT COUNT(*) FROM payments.payment_intent WHERE deleted_at IS NULL`
	args := []any{}
	if buyerID > 0 {
		args = append(args, buyerID)
		query += ` AND buyer_id = $` + strconv.Itoa(len(args)) + `::bigint`
	}
	if status != "" {
		args = append(args, status)
		query += ` AND status = $` + strconv.Itoa(len(args)) + `::varchar`
	}
	var n int
	err := r.conn().QueryRow(ctx, query, args...).Scan(&n)
	return n, err
}

func (r *PaymentRepository) ListIntents(ctx context.Context, buyerID int64, status string, limit, offset int) ([]PaymentIntent, error) {
	query := `
		SELECT ` + intentColumns + `
		` + intentFrom + `
		WHERE pi.deleted_at IS NULL
	`
	args := []any{}
	// nextParam numbers the just-appended argument (call after append).
	nextParam := func() string {
		return "$" + strconv.Itoa(len(args))
	}
	if buyerID > 0 {
		args = append(args, buyerID)
		query += ` AND pi.buyer_id = ` + nextParam()
	}
	if status != "" {
		args = append(args, status)
		query += ` AND pi.status = ` + nextParam() + `::varchar`
	}
	query += ` ORDER BY pi.created_at DESC`
	if limit > 0 {
		if limit > 200 {
			limit = 200
		}
		args = append(args, limit)
		query += ` LIMIT ` + nextParam() + `::int`
	}
	if offset > 0 {
		args = append(args, offset)
		query += ` OFFSET ` + nextParam() + `::int`
	}
	rows, err := r.conn().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PaymentIntent
	for rows.Next() {
		var i PaymentIntent
		var methodID *int64
		var methodCode *string
		if err := rows.Scan(scanIntent(&i, &methodID, &methodCode)...); err != nil {
			return nil, err
		}
		i.PaymentMethodID = methodID
		i.MethodCode = methodCode
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *PaymentRepository) CreateAttempt(ctx context.Context, intentID int64, result string, gatewayRef *string, note string, recordedBy *int64) (PaymentAttempt, error) {
	var a PaymentAttempt
	err := r.conn().QueryRow(ctx, `
		INSERT INTO payments.payment_attempt (intent_id, result, gateway_ref, note, recorded_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, intent_id, result, gateway_ref, note, recorded_by, created_at
	`, intentID, result, gatewayRef, note, recordedBy).Scan(
		&a.ID, &a.IntentID, &a.Result, &a.GatewayRef, &a.Note, &a.RecordedBy, &a.CreatedAt)
	return a, err
}

func (r *PaymentRepository) UpdateIntentStatus(ctx context.Context, id int64, status string) (PaymentIntent, error) {
	var i PaymentIntent
	var methodID *int64
	err := r.conn().QueryRow(ctx, `
		UPDATE payments.payment_intent
		SET status = $2::varchar, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, buyer_id, order_id, order_code, payment_method_id, amount_minor, currency, status, idem_key, created_at, updated_at
	`, id, status).Scan(&i.ID, &i.Code, &i.BuyerID, &i.OrderID, &i.OrderCode, &methodID,
		&i.AmountMinor, &i.Currency, &i.Status, &i.IdemKey, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return PaymentIntent{}, err
	}
	i.PaymentMethodID = methodID
	if methodID != nil {
		if m, merr := r.GetMethodByID(ctx, *methodID); merr == nil {
			mc := m.Code
			i.MethodCode = &mc
		}
	}
	return i, nil
}

func (r *PaymentRepository) GetAttempts(ctx context.Context, intentID int64) ([]PaymentAttempt, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT id, intent_id, result, gateway_ref, note, recorded_by, created_at
		FROM payments.payment_attempt
		WHERE intent_id = $1
		ORDER BY created_at ASC
	`, intentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PaymentAttempt
	for rows.Next() {
		var a PaymentAttempt
		if err := rows.Scan(&a.ID, &a.IntentID, &a.Result, &a.GatewayRef, &a.Note, &a.RecordedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
