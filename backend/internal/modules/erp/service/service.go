package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/erp/repository"
)

var (
	ErrNotFound             = errors.New("erp: not found")
	ErrInvalidInput         = errors.New("erp: invalid input")
	ErrWebhookUnauthorized  = errors.New("erp: webhook unauthorized")
	ErrWebhookStale         = errors.New("erp: webhook stale")
	ErrWebhookInvalid       = errors.New("erp: webhook invalid")
	ErrOrderNotFound        = errors.New("erp: order not found")
	ErrOrderNotDispatchable = errors.New("erp: order not dispatchable")
	ErrAlreadyDispatched    = errors.New("erp: already dispatched")
)

const (
	WebhookTimestampWindow = 5 * time.Minute
	MaxRetryAttempts       = 5
	RetryBackoffBase       = 30 * time.Second
)

// ERPService provides ERP integration business logic.
type ERPService struct {
	db             *database.DB
	repo           *repository.ERPRepository
	webhookSecrets map[string]string
}

// NewERPService creates a new ERP service.
func NewERPService(db *database.DB) *ERPService {
	return &ERPService{
		db:             db,
		repo:           repository.NewERPRepository(db),
		webhookSecrets: make(map[string]string),
	}
}

// SetWebhookSecrets configures webhook verification secrets per provider.
func (s *ERPService) SetWebhookSecrets(secrets map[string]string) {
	s.webhookSecrets = secrets
}

// SetWebhookSecret configures a single default webhook secret.
func (s *ERPService) SetWebhookSecret(secret string) {
	s.webhookSecrets["default"] = secret
}

// --- Webhook handling ---

// VerifyAndProcessWebhook verifies the HMAC signature and processes the event.
func (s *ERPService) VerifyAndProcessWebhook(ctx context.Context, provider, topic string, timestamp int64, body []byte, signature string) (*repository.WebhookEvent, error) {
	// Check timestamp freshness
	now := time.Now()
	ts := time.Unix(timestamp, 0)
	if now.Sub(ts) > WebhookTimestampWindow {
		return nil, ErrWebhookStale
	}

	// Verify HMAC signature
	secret, ok := s.webhookSecrets[provider]
	if !ok {
		secret = s.webhookSecrets["default"]
	}
	if secret == "" {
		return nil, ErrWebhookUnauthorized
	}

	if !verifyHMACSignature(secret, fmt.Sprintf("%d", timestamp), body, signature) {
		return nil, ErrWebhookUnauthorized
	}

	// Extract event ID from payload for deduplication
	var payload struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.EventID == "" {
		payload.EventID = uuid.NewString()
	}
	eventID := payload.EventID

	// Insert with dedup
	event, created, err := s.repo.InsertWebhookEvent(ctx, provider, eventID, topic, string(body))
	if err != nil {
		return nil, fmt.Errorf("insert webhook event: %w", err)
	}

	if !created {
		// Replay - return existing
		return event, nil
	}

	// Process the event (simulated)
	if err := s.processWebhookEvent(ctx, event, topic, body); err != nil {
		_ = s.repo.UpdateWebhookStatus(ctx, event.ID, "failed")
		return nil, fmt.Errorf("process webhook: %w", err)
	}

	_ = s.repo.UpdateWebhookStatus(ctx, event.ID, "applied")
	event.Status = "applied"
	nowApplied := time.Now()
	event.AppliedAt = &nowApplied
	return event, nil
}

func (s *ERPService) processWebhookEvent(ctx context.Context, event *repository.WebhookEvent, topic string, body []byte) error {
	// Simulated processing - in real implementation would handle different topics
	// e.g., stock updates, order status changes, product updates
	return nil
}

func verifyHMACSignature(secret, timestamp string, body []byte, signature string) bool {
	message := timestamp + "." + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// --- Sync jobs ---

// TriggerCatalogSync creates a catalog sync job.
func (s *ERPService) TriggerCatalogSync(ctx context.Context, triggeredBy *int64, force bool) (*repository.SyncJob, error) {
	code := newCode("erp_cat")
	job, err := s.repo.CreateSyncJob(ctx, code, "catalog", triggeredBy)
	if err != nil {
		return nil, fmt.Errorf("create catalog sync job: %w", err)
	}

	// Execute sync synchronously (in real impl would be async)
	if err := s.executeCatalogSync(ctx, job); err != nil {
		_ = s.repo.FailSyncJob(ctx, job.ID, err.Error())
		return nil, fmt.Errorf("execute catalog sync: %w", err)
	}

	return s.repo.GetSyncJobByID(ctx, job.ID)
}

// TriggerStockSync creates a stock sync job.
func (s *ERPService) TriggerStockSync(ctx context.Context, triggeredBy *int64, force bool) (*repository.SyncJob, error) {
	code := newCode("erp_stk")
	job, err := s.repo.CreateSyncJob(ctx, code, "stock", triggeredBy)
	if err != nil {
		return nil, fmt.Errorf("create stock sync job: %w", err)
	}

	// Execute sync synchronously (in real impl would be async)
	if err := s.executeStockSync(ctx, job); err != nil {
		_ = s.repo.FailSyncJob(ctx, job.ID, err.Error())
		return nil, fmt.Errorf("execute stock sync: %w", err)
	}

	return s.repo.GetSyncJobByID(ctx, job.ID)
}

func (s *ERPService) executeCatalogSync(ctx context.Context, job *repository.SyncJob) error {
	if err := s.repo.UpdateSyncJobRunning(ctx, job.ID); err != nil {
		return err
	}

	// Simulated catalog sync - compare local products with ERP
	totalProducts, err := s.repo.CountProducts(ctx)
	if err != nil {
		return fmt.Errorf("count products: %w", err)
	}

	// Simulate: all products match (no drift in simulation)
	matched := totalProducts
	created := 0
	updated := 0
	drifts := 0

	return s.repo.CompleteSyncJob(ctx, job.ID, totalProducts, matched, created, updated, drifts)
}

func (s *ERPService) executeStockSync(ctx context.Context, job *repository.SyncJob) error {
	if err := s.repo.UpdateSyncJobRunning(ctx, job.ID); err != nil {
		return err
	}

	// Simulated stock sync - compare local stock with ERP
	totalStock, err := s.repo.CountStockLevels(ctx)
	if err != nil {
		return fmt.Errorf("count stock levels: %w", err)
	}

	// Simulate: all stock levels match (no drift in simulation)
	matched := totalStock
	created := 0
	updated := 0
	drifts := 0

	return s.repo.CompleteSyncJob(ctx, job.ID, totalStock, matched, created, updated, drifts)
}

// GetSyncJob retrieves a sync job by ID with drift details.
func (s *ERPService) GetSyncJob(ctx context.Context, id int64) (*repository.SyncJob, []repository.SyncDrift, error) {
	job, err := s.repo.GetSyncJobByID(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, fmt.Errorf("get sync job: %w", err)
	}

	drifts, err := s.repo.ListDriftsByJobID(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("list drifts: %w", err)
	}

	return job, drifts, nil
}

// GetSyncJobByCode retrieves a sync job by code.
func (s *ERPService) GetSyncJobByCode(ctx context.Context, code string) (*repository.SyncJob, []repository.SyncDrift, error) {
	job, err := s.repo.GetSyncJobByCode(ctx, code)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, fmt.Errorf("get sync job: %w", err)
	}

	drifts, err := s.repo.ListDriftsByJobID(ctx, job.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("list drifts: %w", err)
	}

	return job, drifts, nil
}

// --- Order dispatch ---

// DispatchOrder creates or retries an order dispatch to ERP.
func (s *ERPService) DispatchOrder(ctx context.Context, orderID int64, triggeredBy *int64, force bool) (*repository.OrderDispatch, error) {
	// Check order exists and is in a dispatchable state
	exists, status, err := s.repo.OrderExists(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("check order: %w", err)
	}
	if !exists {
		return nil, ErrOrderNotFound
	}

	// Only certain statuses can be dispatched
	if !isDispatchableStatus(status) && !force {
		return nil, ErrOrderNotDispatchable
	}

	// Check if dispatch already exists
	existing, err := s.repo.GetDispatchByOrderID(ctx, orderID)
	if err == nil {
		// Dispatch exists - check if we can retry
		if existing.Status == "dispatched" {
			return nil, ErrAlreadyDispatched
		}
		if existing.Status == "failed" || existing.Status == "retry" {
			// Retry the dispatch
			if err := s.executeDispatch(ctx, existing); err != nil {
				nextRetry := calculateNextRetry(existing.AttemptCount)
				_ = s.repo.UpdateDispatchFailed(ctx, existing.ID, err.Error(), nextRetry)
				return s.repo.GetDispatchByOrderID(ctx, orderID)
			}
			return s.repo.GetDispatchByOrderID(ctx, orderID)
		}
		return existing, nil
	}

	// Create new dispatch
	code := newCode("erp_dsp")
	dispatch, err := s.repo.CreateOrderDispatch(ctx, code, orderID, triggeredBy)
	if err != nil {
		return nil, fmt.Errorf("create dispatch: %w", err)
	}

	// Execute dispatch
	if err := s.executeDispatch(ctx, dispatch); err != nil {
		nextRetry := calculateNextRetry(dispatch.AttemptCount)
		_ = s.repo.UpdateDispatchFailed(ctx, dispatch.ID, err.Error(), nextRetry)
		return s.repo.GetDispatchByOrderID(ctx, orderID)
	}

	return s.repo.GetDispatchByOrderID(ctx, orderID)
}

func (s *ERPService) executeDispatch(ctx context.Context, dispatch *repository.OrderDispatch) error {
	// Simulated dispatch - in real implementation would call ERP API
	// For now, always succeed
	return s.repo.UpdateDispatchDispatched(ctx, dispatch.ID)
}

func isDispatchableStatus(status string) bool {
	// Orders that are confirmed/paid can be dispatched
	switch status {
	case "confirmed", "paid", "processing", "awaiting_dispatch":
		return true
	}
	return false
}

func calculateNextRetry(attemptCount int) *time.Time {
	if attemptCount >= MaxRetryAttempts {
		return nil // No more retries
	}
	// Exponential backoff: 30s, 1m, 2m, 4m, 8m
	backoff := RetryBackoffBase * time.Duration(1<<uint(attemptCount))
	next := time.Now().Add(backoff)
	return &next
}

// --- Helpers ---

func newCode(prefix string) string {
	ts := time.Now().UnixNano()
	return fmt.Sprintf("%s_%s", prefix, encodeBase36(ts))
}

func encodeBase36(n int64) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{alphabet[n%36]}, result...)
		n /= 36
	}
	return string(result)
}
