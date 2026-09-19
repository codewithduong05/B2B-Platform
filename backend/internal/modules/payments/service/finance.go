package service

// TASK-007 finance: refunds with approval, reconciliation view, credit
// accounts. Refunds restore invoice balances via commerce.ReversePayment;
// reconciliation is a derived read over intents, invoices, and webhooks.

import (
	"context"
	"errors"
	"fmt"

	"github.com/atlas-platform/backend/internal/database"
	commerce_service "github.com/atlas-platform/backend/internal/modules/commerce/service"
	"github.com/atlas-platform/backend/internal/modules/payments/events"
	"github.com/atlas-platform/backend/internal/modules/payments/repository"
	"github.com/atlas-platform/backend/internal/modules/payments/schema"
	"github.com/jackc/pgx/v5"
)

var (
	ErrRefundNotFound     = errors.New("refund not found")
	ErrRefundState        = errors.New("refund is not actionable")
	ErrRefundSelfApproval = errors.New("second approver required")
	ErrCreditNotFound     = errors.New("credit account not found")
)

func newRefundCode() string {
	return "rf_" + newCodeSuffix()
}

// RequestRefund opens a refund against a succeeded intent. Amounts at or
// above RefundApprovalThreshold stay pending_approval; smaller amounts are
// approved immediately by the requesting staffer.
func (s *PaymentService) RequestRefund(ctx context.Context, actor int64, intentCode string, amountMinor int64, reason string) (*schema.RefundResponse, error) {
	if amountMinor <= 0 || reason == "" {
		return nil, ErrInvalidIntent
	}
	intent, err := s.repo.GetIntentByCode(ctx, intentCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrIntentNotFound
		}
		return nil, fmt.Errorf("get intent: %w", err)
	}
	if intent.Status != IntentStatusSucceeded {
		return nil, ErrRefundState
	}
	refunded, err := s.repo.RefundedTotal(ctx, intent.ID)
	if err != nil {
		return nil, fmt.Errorf("refunded total: %w", err)
	}
	if amountMinor > intent.AmountMinor-refunded {
		return nil, ErrExceedsBalance
	}

	status := RefundStatusApproved
	if amountMinor >= RefundApprovalThreshold {
		status = RefundStatusPending
	}
	rf, err := s.repo.CreateRefund(ctx, newRefundCode(), intent.ID, amountMinor, intent.Currency, reason, status, &actor)
	if err != nil {
		return nil, fmt.Errorf("create refund: %w", err)
	}
	return toRefundResponse(rf, intent.Code), nil
}

// ApproveRefund approves (and applies) a pending refund, or retries
// application of an approved-but-unapplied one. Large refunds require an
// approver distinct from the requester.
func (s *PaymentService) ApproveRefund(ctx context.Context, approver int64, refundID int64) (*schema.RefundResponse, error) {
	var rf repository.PaymentRefund
	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewPaymentRepositoryWithTx(tx)
		locked, err := txRepo.GetRefundByIDForUpdate(ctx, refundID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrRefundNotFound
			}
			return fmt.Errorf("lock refund: %w", err)
		}
		switch locked.Status {
		case RefundStatusPending:
			if locked.AmountMinor >= RefundApprovalThreshold {
				if locked.RequestedBy == nil || *locked.RequestedBy == approver {
					return ErrRefundSelfApproval
				}
			}
		case RefundStatusApproved:
			// Retry path: re-apply an approved refund whose application
			// failed previously.
		default:
			return ErrRefundState
		}
		u, err := txRepo.UpdateRefund(ctx, locked.ID, RefundStatusApproved, &approver)
		if err != nil {
			return fmt.Errorf("approve refund: %w", err)
		}
		rf = u
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Resolve the intent for allocation + response (locked read above
	// guarantees it still exists).
	intent, err := s.repo.GetIntentByID(ctx, rf.IntentID)
	if err != nil {
		return nil, fmt.Errorf("get intent: %w", err)
	}
	if _, err := s.commerceSvc.ReversePayment(ctx, intent.OrderID, rf.AmountMinor); err != nil {
		return nil, fmt.Errorf("restore invoice balances: %w", err)
	}
	applied, err := s.repo.UpdateRefund(ctx, rf.ID, "applied", &approver)
	if err != nil {
		return nil, fmt.Errorf("mark refund applied: %w", err)
	}
	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, events.EventRefundIssued, events.NewEnvelope(
			events.EventRefundIssued,
			events.RefundPayload{
				RefundCode: applied.Code, IntentCode: intent.Code,
				BuyerID: intent.BuyerID, OrderCode: intent.OrderCode,
				AmountMinor: applied.AmountMinor, Currency: applied.Currency,
			},
			"",
		))
	}
	_ = intent
	return toRefundResponse(applied, intent.Code), nil
}

func toRefundResponse(rf repository.PaymentRefund, intentCode string) *schema.RefundResponse {
	return &schema.RefundResponse{
		Code: rf.Code, IntentCode: intentCode, AmountMinor: rf.AmountMinor,
		Currency: rf.Currency, Reason: rf.Reason, Status: rf.Status,
		RequestedBy: rf.RequestedBy, ApprovedBy: rf.ApprovedBy,
		CreatedAt: rf.CreatedAt, UpdatedAt: rf.UpdatedAt,
	}
}

// RejectRefund rejects a pending refund. Approved/applied/rejected rows are
// terminal.
func (s *PaymentService) RejectRefund(ctx context.Context, approver int64, refundID int64) (*schema.RefundResponse, error) {
	var rf repository.PaymentRefund
	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewPaymentRepositoryWithTx(tx)
		locked, err := txRepo.GetRefundByIDForUpdate(ctx, refundID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrRefundNotFound
			}
			return fmt.Errorf("lock refund: %w", err)
		}
		if locked.Status != RefundStatusPending {
			return ErrRefundState
		}
		u, err := txRepo.UpdateRefund(ctx, locked.ID, RefundStatusRejected, &approver)
		if err != nil {
			return fmt.Errorf("reject refund: %w", err)
		}
		rf = u
		return nil
	})
	if err != nil {
		return nil, err
	}
	intent, err := s.repo.GetIntentByID(ctx, rf.IntentID)
	if err != nil {
		return nil, fmt.Errorf("get intent: %w", err)
	}
	return toRefundResponse(rf, intent.Code), nil
}

func (s *PaymentService) GetRefund(ctx context.Context, refundID int64) (*schema.RefundResponse, error) {
	rf, err := s.repo.GetRefundByID(ctx, refundID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRefundNotFound
		}
		return nil, fmt.Errorf("get refund: %w", err)
	}
	intent, err := s.repo.GetIntentByID(ctx, rf.IntentID)
	if err != nil {
		return nil, fmt.Errorf("get intent: %w", err)
	}
	return toRefundResponse(rf, intent.Code), nil
}

// ReconRow is one order's money reconciliation state. PaidApplied is derived
// (issued totals minus issued balances); SucceededTotal sums succeeded
// intents. Matched means every captured unit is accounted for on invoices.
type ReconRow struct {
	OrderID         int64
	OrderCode       string
	InvoicedTotal   int64
	PaidApplied     int64
	SucceededTotal  int64
	Unmatched       bool
}

// Reconcile builds the admin reconciliation view: per-order money rows plus
// webhook deliveries that reference unknown intents.
func (s *PaymentService) Reconcile(ctx context.Context, limit int) ([]schema.ReconResponse, []schema.WebhookResponse, error) {
	intents, err := s.repo.ListIntents(ctx, 0, "", 0, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("list intents: %w", err)
	}
	byOrder := make(map[int64][]repository.PaymentIntent)
	var orderIDs []int64
	for _, in := range intents {
		if _, ok := byOrder[in.OrderID]; !ok {
			orderIDs = append(orderIDs, in.OrderID)
		}
		byOrder[in.OrderID] = append(byOrder[in.OrderID], in)
	}

	var rows []schema.ReconResponse
	for _, orderID := range orderIDs {
		if limit > 0 && len(rows) >= limit {
			break
		}
		orderCode := ""
		var succeeded int64
		for _, in := range byOrder[orderID] {
			orderCode = in.OrderCode
			if in.Status == IntentStatusSucceeded {
				succeeded += in.AmountMinor
			}
		}
		invoiced, paid, err := s.orderMoney(ctx, orderID)
		if err != nil {
			return nil, nil, err
		}
		status := "matched"
		if paid != succeeded {
			status = "discrepancy"
		}
		rows = append(rows, schema.ReconResponse{
			OrderID: orderID, OrderCode: orderCode,
			InvoicedTotal: invoiced, PaidApplied: paid,
			SucceededTotal: succeeded, Status: status,
		})
	}

	unmatched, err := s.repo.ListUnmatchedWebhooks(ctx, 100)
	if err != nil {
		return nil, nil, fmt.Errorf("list unmatched webhooks: %w", err)
	}
	var hooks []schema.WebhookResponse
	for _, e := range unmatched {
		hooks = append(hooks, schema.WebhookResponse{
			Provider: e.Provider, EventID: e.ProviderEventID, EventType: e.EventType,
			IntentCode: e.IntentCode, Status: e.Status, CreatedAt: e.CreatedAt,
		})
	}
	if hooks == nil {
		hooks = []schema.WebhookResponse{}
	}
	return rows, hooks, nil
}

func (s *PaymentService) orderMoney(ctx context.Context, orderID int64) (invoiced, paid int64, err error) {
	return s.commerceSvc.OrderMoney(ctx, orderID)
}

// CheckCredit implements commerce_service.CreditChecker: missing account
// means no terms (unlimited); holds and limit breaches are rejected.
func (s *PaymentService) CheckCredit(ctx context.Context, buyerID, orderTotalMinor int64) error {
	acct, err := s.repo.GetCreditAccount(ctx, buyerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("get credit account: %w", err)
	}
	if acct.OnHold {
		return commerce_service.ErrCreditHold
	}
	exposure, err := s.commerceSvc.OutstandingExposure(ctx, buyerID)
	if err != nil {
		return fmt.Errorf("credit exposure: %w", err)
	}
	if exposure+orderTotalMinor > acct.CreditLimitMinor {
		return commerce_service.ErrCreditLimitExceeded
	}
	return nil
}

func (s *PaymentService) GetCredit(ctx context.Context, buyerID int64) (*schema.CreditResponse, error) {
	acct, err := s.repo.GetCreditAccount(ctx, buyerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCreditNotFound
		}
		return nil, fmt.Errorf("get credit account: %w", err)
	}
	exposure, err := s.commerceSvc.OutstandingExposure(ctx, buyerID)
	if err != nil {
		return nil, fmt.Errorf("credit exposure: %w", err)
	}
	return toCreditResponse(acct, exposure), nil
}

func (s *PaymentService) SetCredit(ctx context.Context, buyerID, limitMinor int64, terms string, onHold bool, holdReason *string) (*schema.CreditResponse, error) {
	if limitMinor < 0 || terms == "" {
		return nil, ErrInvalidIntent
	}
	acct, err := s.repo.UpsertCreditAccount(ctx, buyerID, limitMinor, terms, onHold, holdReason)
	if err != nil {
		return nil, fmt.Errorf("upsert credit account: %w", err)
	}
	exposure, err := s.commerceSvc.OutstandingExposure(ctx, buyerID)
	if err != nil {
		return nil, fmt.Errorf("credit exposure: %w", err)
	}
	return toCreditResponse(acct, exposure), nil
}

func toCreditResponse(acct repository.CreditAccount, exposure int64) *schema.CreditResponse {
	available := acct.CreditLimitMinor - exposure
	if available < 0 {
		available = 0
	}
	return &schema.CreditResponse{
		BuyerID: acct.BuyerID, CreditLimitMinor: acct.CreditLimitMinor,
		Terms: acct.Terms, OnHold: acct.OnHold, HoldReason: acct.HoldReason,
		ExposureMinor: exposure, AvailableMinor: available,
		CreatedAt: acct.CreatedAt, UpdatedAt: acct.UpdatedAt,
	}
}
