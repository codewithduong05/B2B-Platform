package service

// TASK-007: provider webhooks (HMAC + dedupe), refunds with approval,
// reconciliation view, credit accounts. No real gateway: settlement stays
// manual, webhooks simulate provider callbacks (sim) or bank hooks.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/payments/events"
	"github.com/atlas-platform/backend/internal/modules/payments/repository"
	"github.com/jackc/pgx/v5"
)

var (
	ErrWebhookUnauthorized = errors.New("invalid webhook signature")
	ErrWebhookStale        = errors.New("stale webhook timestamp")
	ErrWebhookInvalid      = errors.New("invalid webhook payload")
)

// RefundApprovalThreshold: refunds at or above this amount (minor units)
// require an approver distinct from the requester (A7.3).
const RefundApprovalThreshold int64 = 1000000

// WebhookTimestampWindow bounds delivery skew (contract: reject stale).
const WebhookTimestampWindow = 5 * time.Minute

const (
	RefundStatusPending  = "pending_approval"
	RefundStatusApproved = "approved"
	RefundStatusRejected = "rejected"
	RefundStatusApplied  = "applied"
)

type WebhookOutcome struct {
	Replayed   bool   `json:"replayed"`
	Applied    bool   `json:"applied"`
	IntentCode string `json:"intent_code,omitempty"`
	Status     string `json:"status"`
}

// SetWebhookSecrets configures per-provider HMAC secrets (main.go wires
// env; tests inject directly).
func (s *PaymentService) SetWebhookSecrets(secrets map[string]string) {
	s.webhookSecrets = secrets
}

func (s *PaymentService) webhookSecret(provider string) (string, bool) {
	if s.webhookSecrets == nil {
		return "", false
	}
	secret, ok := s.webhookSecrets[provider]
	return secret, ok && secret != ""
}

type webhookPayload struct {
	EventID    string `json:"event_id"`
	Type       string `json:"type"`
	IntentCode string `json:"intent_code"`
}

func verifyWebhookSignature(secret, timestamp string, body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "."))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	sig, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	exp, err := hex.DecodeString(expected)
	if err != nil {
		return false
	}
	return hmac.Equal(sig, exp)
}

// HandleWebhook verifies, dedupes, and applies a provider callback.
// Replay semantics: an already-applied event ID replays the current intent
// outcome without re-applying; an in-flight (received) row is processed.
func (s *PaymentService) HandleWebhook(ctx context.Context, provider string, body []byte, timestampHeader, signature string) (*WebhookOutcome, error) {
	secret, ok := s.webhookSecret(provider)
	if !ok {
		return nil, ErrWebhookUnauthorized
	}
	ts, err := strconv.ParseInt(timestampHeader, 10, 64)
	if err != nil {
		return nil, ErrWebhookUnauthorized
	}
	if skew := time.Since(time.Unix(ts, 0)); skew < 0 {
		if -skew > WebhookTimestampWindow {
			return nil, ErrWebhookStale
		}
	} else if skew > WebhookTimestampWindow {
		return nil, ErrWebhookStale
	}
	if !verifyWebhookSignature(secret, timestampHeader, body, signature) {
		return nil, ErrWebhookUnauthorized
	}

	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil || payload.EventID == "" {
		return nil, ErrWebhookInvalid
	}
	if payload.Type != "intent.succeeded" && payload.Type != "intent.failed" {
		return nil, ErrWebhookInvalid
	}
	if len(payload.EventID) > 128 {
		return nil, ErrWebhookInvalid
	}
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])

	event, created, err := s.repo.InsertWebhookEvent(ctx, provider, payload.EventID, payload.Type, hash, payload.IntentCode)
	if err != nil {
		return nil, fmt.Errorf("record webhook event: %w", err)
	}
	if !created {
		return s.replayWebhook(ctx, &event)
	}
	return s.applyWebhook(ctx, &event, &payload)
}

func (s *PaymentService) replayWebhook(ctx context.Context, event *repository.WebhookEvent) (*WebhookOutcome, error) {
	switch event.Status {
	case "applied":
		intent, err := s.repo.GetIntentByCode(ctx, event.IntentCode)
		if err != nil {
			return &WebhookOutcome{Replayed: true, Applied: true, IntentCode: event.IntentCode, Status: "applied"}, nil
		}
		return &WebhookOutcome{Replayed: true, Applied: true, IntentCode: intent.Code, Status: intent.Status}, nil
	case "unmatched":
		return &WebhookOutcome{Replayed: true, Applied: false, IntentCode: event.IntentCode, Status: "unmatched"}, nil
	default: // received: a previous attempt crashed mid-apply; process it.
		var payload webhookPayload
		payload.EventID = event.ProviderEventID
		payload.Type = event.EventType
		payload.IntentCode = event.IntentCode
		return s.applyWebhook(ctx, event, &payload)
	}
}

func (s *PaymentService) applyWebhook(ctx context.Context, event *repository.WebhookEvent, payload *webhookPayload) (*WebhookOutcome, error) {
	intent, err := s.repo.GetIntentByCode(ctx, payload.IntentCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = s.repo.UpdateWebhookStatus(ctx, event.ID, "unmatched")
			return &WebhookOutcome{Replayed: false, Applied: false, IntentCode: payload.IntentCode, Status: "unmatched"}, nil
		}
		return nil, fmt.Errorf("get intent: %w", err)
	}
	// Idempotent outcome: already at the target state counts as applied.
	target := IntentStatusFailed
	if payload.Type == "intent.succeeded" {
		target = IntentStatusSucceeded
	}
	if intent.Status == target {
		_ = s.repo.UpdateWebhookStatus(ctx, event.ID, "applied")
		return &WebhookOutcome{Replayed: false, Applied: true, IntentCode: intent.Code, Status: intent.Status}, nil
	}
	if isTerminalIntent(intent.Status) {
		_ = s.repo.UpdateWebhookStatus(ctx, event.ID, "unmatched")
		return &WebhookOutcome{Replayed: false, Applied: false, IntentCode: intent.Code, Status: "terminal_conflict"}, nil
	}

	succeeded := payload.Type == "intent.succeeded"
	updated, err := s.markIntentLocked(ctx, 0, intent.ID, succeeded, "webhook "+event.ProviderEventID)
	if err != nil {
		// Lost a race with a concurrent delivery of the same (or another)
		// event: re-read and converge. If the intent landed on our target,
		// this delivery is satisfied; otherwise it genuinely conflicts.
		if errors.Is(err, ErrIntentTerminal) {
			current, gerr := s.repo.GetIntentByCode(ctx, intent.Code)
			if gerr == nil && current.Status == target {
				_ = s.repo.UpdateWebhookStatus(ctx, event.ID, "applied")
				return &WebhookOutcome{Replayed: false, Applied: true, IntentCode: current.Code, Status: current.Status}, nil
			}
			_ = s.repo.UpdateWebhookStatus(ctx, event.ID, "unmatched")
			return &WebhookOutcome{Replayed: false, Applied: false, IntentCode: intent.Code, Status: "terminal_conflict"}, nil
		}
		return nil, err
	}
	_ = s.repo.UpdateWebhookStatus(ctx, event.ID, "applied")
	s.publishIntentEvent(ctx, updated, succeeded)
	return &WebhookOutcome{Replayed: false, Applied: true, IntentCode: updated.Code, Status: updated.Status}, nil
}

// markIntentLocked performs the terminal-gated attempt+status update used by
// both manual settlement and webhook application.
func (s *PaymentService) markIntentLocked(ctx context.Context, actor int64, intentID int64, succeeded bool, note string) (repository.PaymentIntent, error) {
	target, result := IntentStatusFailed, "failed"
	if succeeded {
		target, result = IntentStatusSucceeded, "succeeded"
	}
	var updated repository.PaymentIntent
	err := s.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := repository.NewPaymentRepositoryWithTx(tx)
		locked, err := txRepo.GetIntentByIDForUpdate(ctx, intentID)
		if err != nil {
			return fmt.Errorf("lock intent: %w", err)
		}
		if isTerminalIntent(locked.Status) {
			return ErrIntentTerminal
		}
		var actorPtr *int64
		if actor != 0 {
			actorPtr = &actor
		}
		if _, err := txRepo.CreateAttempt(ctx, locked.ID, result, nil, note, actorPtr); err != nil {
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
		return repository.PaymentIntent{}, err
	}
	return updated, nil
}

func (s *PaymentService) publishIntentEvent(ctx context.Context, intent repository.PaymentIntent, succeeded bool) {
	if s.publisher == nil {
		return
	}
	eventType := events.EventIntentFailed
	if succeeded {
		eventType = events.EventIntentSucceeded
	}
	_ = s.publisher.Publish(ctx, eventType, events.NewEnvelope(
		eventType,
		events.IntentPayload{
			IntentCode: intent.Code, BuyerID: intent.BuyerID,
			OrderCode: intent.OrderCode, AmountMinor: intent.AmountMinor,
			Currency: intent.Currency,
		},
		"",
	))
}
