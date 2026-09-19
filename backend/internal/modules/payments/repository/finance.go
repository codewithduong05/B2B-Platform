package repository

// TASK-007: webhook deliveries, refunds, credit accounts.

import (
	"context"
	"time"
)

type WebhookEvent struct {
	ID              int64
	Provider        string
	ProviderEventID string
	EventType       string
	PayloadHash     string
	IntentCode      string
	Status          string
	CreatedAt       time.Time
}

type PaymentRefund struct {
	ID          int64
	Code        string
	IntentID    int64
	AmountMinor int64
	Currency    string
	Reason      string
	Status      string
	RequestedBy *int64
	ApprovedBy  *int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreditAccount struct {
	BuyerID          int64
	CreditLimitMinor int64
	Terms            string
	OnHold           bool
	HoldReason       *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// InsertWebhookEvent records a delivery; on provider-event conflict the
// existing row is returned (created=false) for replay semantics.
func (r *PaymentRepository) InsertWebhookEvent(ctx context.Context, provider, eventID, eventType, payloadHash, intentCode string) (WebhookEvent, bool, error) {
	var e WebhookEvent
	err := r.conn().QueryRow(ctx, `
		INSERT INTO payments.webhook_event (provider, provider_event_id, event_type, payload_hash, intent_code, status)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), 'received')
		ON CONFLICT (provider, provider_event_id) DO NOTHING
		RETURNING id, provider, provider_event_id, event_type, payload_hash, COALESCE(intent_code, ''), status, created_at
	`, provider, eventID, eventType, payloadHash, intentCode).Scan(
		&e.ID, &e.Provider, &e.ProviderEventID, &e.EventType, &e.PayloadHash, &e.IntentCode, &e.Status, &e.CreatedAt)
	if err == nil {
		return e, true, nil
	}
	var existing WebhookEvent
	gerr := r.conn().QueryRow(ctx, `
		SELECT id, provider, provider_event_id, event_type, payload_hash, COALESCE(intent_code, ''), status, created_at
		FROM payments.webhook_event
		WHERE provider = $1 AND provider_event_id = $2
	`, provider, eventID).Scan(
		&existing.ID, &existing.Provider, &existing.ProviderEventID, &existing.EventType,
		&existing.PayloadHash, &existing.IntentCode, &existing.Status, &existing.CreatedAt)
	if gerr != nil {
		return WebhookEvent{}, false, gerr
	}
	return existing, false, nil
}

func (r *PaymentRepository) UpdateWebhookStatus(ctx context.Context, id int64, status string) error {
	_, err := r.conn().Exec(ctx, `
		UPDATE payments.webhook_event SET status = $2 WHERE id = $1
	`, id, status)
	return err
}

func (r *PaymentRepository) ListUnmatchedWebhooks(ctx context.Context, limit int) ([]WebhookEvent, error) {
	rows, err := r.conn().Query(ctx, `
		SELECT id, provider, provider_event_id, event_type, payload_hash, COALESCE(intent_code, ''), status, created_at
		FROM payments.webhook_event
		WHERE status = 'unmatched'
		ORDER BY created_at DESC
		LIMIT $1::int
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []WebhookEvent
	for rows.Next() {
		var e WebhookEvent
		if err := rows.Scan(&e.ID, &e.Provider, &e.ProviderEventID, &e.EventType,
			&e.PayloadHash, &e.IntentCode, &e.Status, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PaymentRepository) CreateRefund(ctx context.Context, code string, intentID, amountMinor int64, currency, reason, status string, requestedBy *int64) (PaymentRefund, error) {
	var rf PaymentRefund
	err := r.conn().QueryRow(ctx, `
		INSERT INTO payments.payment_refund (code, intent_id, amount_minor, currency, reason, status, requested_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, code, intent_id, amount_minor, currency, reason, status, requested_by, approved_by, created_at, updated_at
	`, code, intentID, amountMinor, currency, reason, status, requestedBy).Scan(
		&rf.ID, &rf.Code, &rf.IntentID, &rf.AmountMinor, &rf.Currency, &rf.Reason,
		&rf.Status, &rf.RequestedBy, &rf.ApprovedBy, &rf.CreatedAt, &rf.UpdatedAt)
	return rf, err
}

func (r *PaymentRepository) GetRefundByID(ctx context.Context, id int64) (PaymentRefund, error) {
	var rf PaymentRefund
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, intent_id, amount_minor, currency, reason, status, requested_by, approved_by, created_at, updated_at
		FROM payments.payment_refund
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&rf.ID, &rf.Code, &rf.IntentID, &rf.AmountMinor, &rf.Currency, &rf.Reason,
		&rf.Status, &rf.RequestedBy, &rf.ApprovedBy, &rf.CreatedAt, &rf.UpdatedAt)
	return rf, err
}

func (r *PaymentRepository) GetRefundByIDForUpdate(ctx context.Context, id int64) (PaymentRefund, error) {
	var rf PaymentRefund
	err := r.conn().QueryRow(ctx, `
		SELECT id, code, intent_id, amount_minor, currency, reason, status, requested_by, approved_by, created_at, updated_at
		FROM payments.payment_refund
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, id).Scan(&rf.ID, &rf.Code, &rf.IntentID, &rf.AmountMinor, &rf.Currency, &rf.Reason,
		&rf.Status, &rf.RequestedBy, &rf.ApprovedBy, &rf.CreatedAt, &rf.UpdatedAt)
	return rf, err
}

// RefundedTotal returns approved+applied refunds for one intent.
func (r *PaymentRepository) RefundedTotal(ctx context.Context, intentID int64) (int64, error) {
	var total int64
	err := r.conn().QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_minor), 0)
		FROM payments.payment_refund
		WHERE intent_id = $1 AND status IN ('approved', 'applied') AND deleted_at IS NULL
	`, intentID).Scan(&total)
	return total, err
}

func (r *PaymentRepository) UpdateRefund(ctx context.Context, id int64, status string, approvedBy *int64) (PaymentRefund, error) {
	var rf PaymentRefund
	err := r.conn().QueryRow(ctx, `
		UPDATE payments.payment_refund
		SET status = $2::varchar, approved_by = COALESCE($3, approved_by), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, code, intent_id, amount_minor, currency, reason, status, requested_by, approved_by, created_at, updated_at
	`, id, status, approvedBy).Scan(&rf.ID, &rf.Code, &rf.IntentID, &rf.AmountMinor, &rf.Currency, &rf.Reason,
		&rf.Status, &rf.RequestedBy, &rf.ApprovedBy, &rf.CreatedAt, &rf.UpdatedAt)
	return rf, err
}

func (r *PaymentRepository) GetCreditAccount(ctx context.Context, buyerID int64) (CreditAccount, error) {
	var a CreditAccount
	err := r.conn().QueryRow(ctx, `
		SELECT buyer_id, credit_limit_minor, terms, on_hold, hold_reason, created_at, updated_at
		FROM payments.credit_account
		WHERE buyer_id = $1
	`, buyerID).Scan(&a.BuyerID, &a.CreditLimitMinor, &a.Terms, &a.OnHold, &a.HoldReason, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (r *PaymentRepository) UpsertCreditAccount(ctx context.Context, buyerID, limitMinor int64, terms string, onHold bool, holdReason *string) (CreditAccount, error) {
	var a CreditAccount
	err := r.conn().QueryRow(ctx, `
		INSERT INTO payments.credit_account (buyer_id, credit_limit_minor, terms, on_hold, hold_reason)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (buyer_id) DO UPDATE
		SET credit_limit_minor = EXCLUDED.credit_limit_minor,
		    terms = EXCLUDED.terms,
		    on_hold = EXCLUDED.on_hold,
		    hold_reason = EXCLUDED.hold_reason,
		    updated_at = NOW()
		RETURNING buyer_id, credit_limit_minor, terms, on_hold, hold_reason, created_at, updated_at
	`, buyerID, limitMinor, terms, onHold, holdReason).Scan(
		&a.BuyerID, &a.CreditLimitMinor, &a.Terms, &a.OnHold, &a.HoldReason, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}
