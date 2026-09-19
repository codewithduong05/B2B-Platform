package service

// Minimal B2B payments module (docs-faithful: payments owns methods,
// intents, attempts). No gateway integration: settlement is recorded
// manually by staff (bank transfer / COD / credit-terms confirmation).
// Money stays honest via order-total snapshots and terminal-state gates.

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	commerce_service "github.com/atlas-platform/backend/internal/modules/commerce/service"
	"github.com/atlas-platform/backend/internal/modules/payments/events"
	"github.com/atlas-platform/backend/internal/modules/payments/repository"
	"github.com/atlas-platform/backend/internal/modules/payments/schema"
	"github.com/jackc/pgx/v5"
)

var (
	ErrIntentNotFound = errors.New("payment intent not found")
	ErrMethodNotFound = errors.New("payment method not found")
	ErrIntentConflict = errors.New("idempotency key reused for a different intent")
	ErrIntentTerminal = errors.New("payment intent is already terminal")
	ErrInvalidMethod  = errors.New("invalid payment method")
	ErrInvalidIntent  = errors.New("invalid payment intent request")
	ErrOrderNotFound  = errors.New("order not found")
	ErrExceedsBalance = errors.New("payment exceeds outstanding balance")
)

const (
	IntentStatusRequiresAction = "requires_action"
	IntentStatusProcessing     = "processing"
	IntentStatusSucceeded      = "succeeded"
	IntentStatusFailed         = "failed"
	IntentStatusCancelled      = "cancelled"
)

var validMethodTypes = map[string]bool{
	"credit_terms": true, "bank_transfer": true, "cod": true, "card": true,
}

func isTerminalIntent(status string) bool {
	return status == IntentStatusSucceeded || status == IntentStatusFailed || status == IntentStatusCancelled
}

type EventPublisher interface {
	Publish(ctx context.Context, routingKey string, payload interface{}) error
}

type PaymentService struct {
	db          *database.DB
	repo        *repository.PaymentRepository
	commerceSvc *commerce_service.CommerceService
	publisher   EventPublisher
}

func NewPaymentService(
	db *database.DB,
	commerceSvc *commerce_service.CommerceService,
	publisher EventPublisher,
) *PaymentService {
	return &PaymentService{
		db:          db,
		repo:        repository.NewPaymentRepository(db),
		commerceSvc: commerceSvc,
		publisher:   publisher,
	}
}

func newCodeSuffix() string {
	return fmt.Sprintf("%s%s",
		strconv.FormatInt(time.Now().UnixNano(), 36),
		strconv.FormatUint(uint64(rand.Uint32()), 36),
	)
}

func newMethodCode() string {
	return "pm_" + newCodeSuffix()
}

func newIntentCode() string {
	return "pi_" + newCodeSuffix()
}

func (s *PaymentService) CreateMethod(ctx context.Context, buyerID int64, methodType, displayName string) (*schema.PaymentMethodResponse, error) {
	if !validMethodTypes[methodType] || displayName == "" {
		return nil, ErrInvalidMethod
	}
	m, err := s.repo.CreateMethod(ctx, newMethodCode(), buyerID, methodType, displayName)
	if err != nil {
		return nil, fmt.Errorf("create payment method: %w", err)
	}
	return &schema.PaymentMethodResponse{
		Code: m.Code, MethodType: m.MethodType, DisplayName: m.DisplayName,
		IsActive: m.IsActive, CreatedAt: m.CreatedAt,
	}, nil
}

func (s *PaymentService) ListMethods(ctx context.Context, buyerID int64) ([]schema.PaymentMethodResponse, error) {
	methods, err := s.repo.ListMethods(ctx, buyerID)
	if err != nil {
		return nil, fmt.Errorf("list payment methods: %w", err)
	}
	out := make([]schema.PaymentMethodResponse, 0, len(methods))
	for _, m := range methods {
		out = append(out, schema.PaymentMethodResponse{
			Code: m.Code, MethodType: m.MethodType, DisplayName: m.DisplayName,
			IsActive: m.IsActive, CreatedAt: m.CreatedAt,
		})
	}
	return out, nil
}

// CreateIntent opens a payment intent for the buyer's own order. The amount
// is snapshotted from the authoritative order total; the client supplies no
// money fields. Same (buyer, idem_key) replays; different target → 409.
// Cancelled orders cannot take new intents.
func (s *PaymentService) CreateIntent(ctx context.Context, buyerID int64, req schema.CreateIntentRequest) (*schema.IntentResponse, bool, error) {
	if req.OrderCode == "" || req.IdemKey == "" || len(req.IdemKey) > 100 {
		return nil, false, ErrInvalidIntent
	}
	order, err := s.commerceSvc.GetOrder(ctx, buyerID, req.OrderCode)
	if err != nil {
		if errors.Is(err, commerce_service.ErrOrderNotFound) {
			return nil, false, ErrOrderNotFound
		}
		return nil, false, fmt.Errorf("get order: %w", err)
	}
	if order.Status == "cancelled" {
		return nil, false, ErrInvalidIntent
	}
	orderID, totalMinor, currency, err := s.commerceSvc.ResolveOrderRef(ctx, buyerID, req.OrderCode)
	if err != nil {
		if errors.Is(err, commerce_service.ErrOrderNotFound) {
			return nil, false, ErrOrderNotFound
		}
		return nil, false, fmt.Errorf("resolve order: %w", err)
	}

	var methodID *int64
	if req.MethodCode != nil && *req.MethodCode != "" {
		m, err := s.repo.GetMethodByCode(ctx, buyerID, *req.MethodCode)
		if err != nil {
			return nil, false, ErrMethodNotFound
		}
		methodID = &m.ID
	}

	intent, err := s.repo.CreateIntent(ctx, newIntentCode(), buyerID, orderID, order.Code, methodID, totalMinor, currency, req.IdemKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			existing, gerr := s.repo.GetIntentByKey(ctx, buyerID, req.IdemKey)
			if gerr != nil {
				return nil, false, fmt.Errorf("get existing intent: %w", gerr)
			}
			if existing.OrderID != orderID || existing.AmountMinor != totalMinor || !sameMethod(existing.PaymentMethodID, methodID) {
				return nil, false, ErrIntentConflict
			}
			resp, err := s.withAttempts(ctx, existing)
			return resp, false, err
		}
		return nil, false, fmt.Errorf("create intent: %w", err)
	}
	resp, err := s.withAttempts(ctx, intent)
	return resp, true, err
}

func sameMethod(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func (s *PaymentService) withAttempts(ctx context.Context, intent repository.PaymentIntent) (*schema.IntentResponse, error) {
	attempts, err := s.repo.GetAttempts(ctx, intent.ID)
	if err != nil {
		return nil, fmt.Errorf("get attempts: %w", err)
	}
	resp := &schema.IntentResponse{
		Code: intent.Code, OrderCode: intent.OrderCode, AmountMinor: intent.AmountMinor,
		Currency: intent.Currency, MethodCode: intent.MethodCode, Status: intent.Status,
		CreatedAt: intent.CreatedAt, UpdatedAt: intent.UpdatedAt,
	}
	for _, a := range attempts {
		resp.Attempts = append(resp.Attempts, schema.AttemptResponse{
			Result: a.Result, GatewayRef: a.GatewayRef, Note: a.Note, CreatedAt: a.CreatedAt,
		})
	}
	return resp, nil
}

// GetIntent returns the buyer's own intent; foreign codes are 404.
func (s *PaymentService) GetIntent(ctx context.Context, buyerID int64, code string) (*schema.IntentResponse, error) {
	intent, err := s.repo.GetIntentByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrIntentNotFound
		}
		return nil, fmt.Errorf("get intent: %w", err)
	}
	if intent.BuyerID != buyerID {
		return nil, ErrIntentNotFound
	}
	return s.withAttempts(ctx, intent)
}

func (s *PaymentService) ListIntents(ctx context.Context, buyerID int64, status string, limit, offset int) ([]schema.IntentResponse, error) {
	return s.listIntents(ctx, buyerID, status, limit, offset)
}

func (s *PaymentService) AdminListIntents(ctx context.Context, status string, limit, offset int) ([]schema.IntentResponse, error) {
	return s.listIntents(ctx, 0, status, limit, offset)
}

func (s *PaymentService) AdminCountIntents(ctx context.Context, status string) (int, error) {
	n, err := s.repo.CountIntents(ctx, 0, status)
	if err != nil {
		return 0, fmt.Errorf("count intents: %w", err)
	}
	return n, nil
}

func (s *PaymentService) listIntents(ctx context.Context, buyerID int64, status string, limit, offset int) ([]schema.IntentResponse, error) {
	intents, err := s.repo.ListIntents(ctx, buyerID, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list intents: %w", err)
	}
	out := make([]schema.IntentResponse, 0, len(intents))
	for _, intent := range intents {
		resp, err := s.withAttempts(ctx, intent)
		if err != nil {
			return nil, err
		}
		out = append(out, *resp)
	}
	return out, nil
}

// MarkIntent records a manual B2B settlement outcome (staff only at the
// router layer). Terminal intents reject further marks. On success the
// amount is allocated across the order's issued invoices.
func (s *PaymentService) MarkIntent(ctx context.Context, actor int64, code string, succeeded bool, note string) (*schema.IntentResponse, error) {
	intent, err := s.repo.GetIntentByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrIntentNotFound
		}
		return nil, fmt.Errorf("get intent: %w", err)
	}

	target := IntentStatusFailed
	result := "failed"
	eventType := events.EventIntentFailed
	if succeeded {
		target = IntentStatusSucceeded
		result = "succeeded"
		eventType = events.EventIntentSucceeded
	}

	var updated repository.PaymentIntent
	err = s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewPaymentRepositoryWithTx(tx)
		locked, err := txRepo.GetIntentByIDForUpdate(ctx, intent.ID)
		if err != nil {
			return fmt.Errorf("lock intent: %w", err)
		}
		if isTerminalIntent(locked.Status) {
			return ErrIntentTerminal
		}
		if _, err := txRepo.CreateAttempt(ctx, locked.ID, result, nil, note, &actor); err != nil {
			return fmt.Errorf("record attempt: %w", err)
		}
		u, err := txRepo.UpdateIntentStatus(ctx, locked.ID, target)
		if err != nil {
			return fmt.Errorf("update intent status: %w", err)
		}
		updated = u
		return nil
	})
	if err != nil {
		return nil, err
	}

	if succeeded {
		// Crash window (documented): intent is durably succeeded while the
		// invoice allocation below runs in its own transaction. Allocation
		// is validated against outstanding balances, so a retry surfaces as
		// overpayment instead of a double-apply.
		if _, err := s.commerceSvc.RecordPayment(ctx, updated.OrderID, updated.AmountMinor); err != nil {
			if errors.Is(err, commerce_service.ErrOverpayment) {
				return nil, ErrExceedsBalance
			}
			return nil, fmt.Errorf("allocate payment to invoices: %w", err)
		}
	}

	if s.publisher != nil {
		_ = s.publisher.Publish(ctx, eventType, events.NewEnvelope(
			eventType,
			events.IntentPayload{
				IntentCode: updated.Code, BuyerID: updated.BuyerID,
				OrderCode: updated.OrderCode, AmountMinor: updated.AmountMinor,
				Currency: updated.Currency,
			},
			"",
		))
	}

	return s.withAttemptsResponse(ctx, updated)
}

func (s *PaymentService) withAttemptsResponse(ctx context.Context, intent repository.PaymentIntent) (*schema.IntentResponse, error) {
	return s.withAttempts(ctx, intent)
}
