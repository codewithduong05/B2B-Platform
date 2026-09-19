package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/atlas-platform/backend/internal/database"
)

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// ERPRepository provides data access for the ERP module.
type ERPRepository struct {
	db *database.DB
	tx *database.Tx
}

// NewERPRepository creates a repository backed by the connection pool.
func NewERPRepository(db *database.DB) *ERPRepository {
	return &ERPRepository{db: db}
}

// NewERPRepositoryWithTx creates a repository bound to a transaction.
func NewERPRepositoryWithTx(db *database.DB, tx *database.Tx) *ERPRepository {
	return &ERPRepository{db: db, tx: tx}
}

func (r *ERPRepository) conn() querier {
	if r.tx != nil {
		return r.tx
	}
	return r.db.Pool
}

// --- Domain models ---

// WebhookEvent represents an inbound ERP webhook event.
type WebhookEvent struct {
	ID              int64
	Provider        string
	ProviderEventID string
	Topic           string
	Payload         string
	Status          string
	ReceivedAt      time.Time
	AppliedAt       *time.Time
	CreatedAt       time.Time
}

// SyncJob represents a sync job record.
type SyncJob struct {
	ID             int64
	Code           string
	JobType        string
	Status         string
	TriggeredBy    *int64
	StartedAt      *time.Time
	CompletedAt    *time.Time
	TotalRecords   int
	MatchedRecords int
	CreatedRecords int
	UpdatedRecords int
	DriftCount     int
	ErrorMessage   *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SyncDrift represents a drift detection record.
type SyncDrift struct {
	ID            int64
	SyncJobID     int64
	EntityType    string
	EntityID      int64
	EntityCode    *string
	DriftType     string
	LocalValue    *string
	RemoteValue   *string
	FieldsChanged []string
	DetectedAt    time.Time
	ResolvedAt    *time.Time
	Resolution    *string
}

// OrderDispatch represents an order dispatch record.
type OrderDispatch struct {
	ID            int64
	Code          string
	OrderID       int64
	Status        string
	AttemptCount  int
	LastAttemptAt *time.Time
	LastError     *string
	NextRetryAt   *time.Time
	DispatchedAt  *time.Time
	TriggeredBy   *int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// --- Webhook events ---

// InsertWebhookEvent inserts a webhook event with deduplication.
func (r *ERPRepository) InsertWebhookEvent(ctx context.Context, provider, eventID, topic, payload string) (*WebhookEvent, bool, error) {
	query := `
		INSERT INTO erp.erp_webhook_event (provider, provider_event_id, topic, payload, status)
		VALUES ($1, $2, $3, $4, 'received')
		ON CONFLICT (provider, provider_event_id) DO NOTHING
		RETURNING id, provider, provider_event_id, topic, status, received_at, created_at
	`
	var e WebhookEvent
	err := r.conn().QueryRow(ctx, query, provider, eventID, topic, payload).Scan(
		&e.ID, &e.Provider, &e.ProviderEventID, &e.Topic, &e.Status, &e.ReceivedAt, &e.CreatedAt,
	)
	if err == nil {
		return &e, true, nil
	}
	if err == pgx.ErrNoRows {
		// Conflict — fetch existing
		err = r.conn().QueryRow(ctx, `
			SELECT id, provider, provider_event_id, topic, payload, status, received_at, applied_at, created_at
			FROM erp.erp_webhook_event WHERE provider = $1 AND provider_event_id = $2
		`, provider, eventID).Scan(
			&e.ID, &e.Provider, &e.ProviderEventID, &e.Topic, &e.Payload, &e.Status, &e.ReceivedAt, &e.AppliedAt, &e.CreatedAt,
		)
		if err != nil {
			return nil, false, fmt.Errorf("fetch existing webhook event: %w", err)
		}
		return &e, false, nil
	}
	return nil, false, fmt.Errorf("insert webhook event: %w", err)
}

// UpdateWebhookStatus updates the status of a webhook event.
func (r *ERPRepository) UpdateWebhookStatus(ctx context.Context, id int64, status string) error {
	var query string
	if status == "applied" {
		query = `UPDATE erp.erp_webhook_event SET status = $1, applied_at = NOW() WHERE id = $2`
	} else {
		query = `UPDATE erp.erp_webhook_event SET status = $1 WHERE id = $2`
	}
	_, err := r.conn().Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update webhook status: %w", err)
	}
	return nil
}

// --- Sync jobs ---

const syncJobColumns = "id, code, job_type, status, triggered_by, started_at, completed_at, total_records, matched_records, created_records, updated_records, drift_count, error_message, created_at, updated_at"

func scanSyncJob(row pgx.Row) (*SyncJob, error) {
	var j SyncJob
	var triggeredBy pgtype.Int8
	var startedAt, completedAt pgtype.Timestamptz
	var errMsg pgtype.Text
	err := row.Scan(
		&j.ID, &j.Code, &j.JobType, &j.Status, &triggeredBy,
		&startedAt, &completedAt, &j.TotalRecords, &j.MatchedRecords,
		&j.CreatedRecords, &j.UpdatedRecords, &j.DriftCount, &errMsg,
		&j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if triggeredBy.Valid {
		j.TriggeredBy = &triggeredBy.Int64
	}
	if startedAt.Valid {
		j.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		j.CompletedAt = &completedAt.Time
	}
	if errMsg.Valid {
		j.ErrorMessage = &errMsg.String
	}
	return &j, nil
}

// CreateSyncJob creates a new sync job.
func (r *ERPRepository) CreateSyncJob(ctx context.Context, code, jobType string, triggeredBy *int64) (*SyncJob, error) {
	query := `
		INSERT INTO erp.sync_job (code, job_type, triggered_by, status)
		VALUES ($1, $2, $3, 'pending')
		RETURNING ` + syncJobColumns
	var tb pgtype.Int8
	if triggeredBy != nil {
		tb.Int64 = *triggeredBy
		tb.Valid = true
	}
	j, err := scanSyncJob(r.conn().QueryRow(ctx, query, code, jobType, tb))
	if err != nil {
		return nil, fmt.Errorf("create sync job: %w", err)
	}
	return j, nil
}

// GetSyncJobByCode retrieves a sync job by code.
func (r *ERPRepository) GetSyncJobByCode(ctx context.Context, code string) (*SyncJob, error) {
	query := `SELECT ` + syncJobColumns + ` FROM erp.sync_job WHERE code = $1`
	j, err := scanSyncJob(r.conn().QueryRow(ctx, query, code))
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get sync job: %w", err)
	}
	return j, nil
}

// GetSyncJobByID retrieves a sync job by ID.
func (r *ERPRepository) GetSyncJobByID(ctx context.Context, id int64) (*SyncJob, error) {
	query := `SELECT ` + syncJobColumns + ` FROM erp.sync_job WHERE id = $1`
	j, err := scanSyncJob(r.conn().QueryRow(ctx, query, id))
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get sync job by id: %w", err)
	}
	return j, nil
}

// UpdateSyncJobRunning marks a job as running.
func (r *ERPRepository) UpdateSyncJobRunning(ctx context.Context, id int64) error {
	_, err := r.conn().Exec(ctx, `
		UPDATE erp.sync_job SET status = 'running', started_at = NOW(), updated_at = NOW() WHERE id = $1
	`, id)
	return err
}

// CompleteSyncJob marks a job as completed with counters.
func (r *ERPRepository) CompleteSyncJob(ctx context.Context, id int64, total, matched, created, updated, drifts int) error {
	_, err := r.conn().Exec(ctx, `
		UPDATE erp.sync_job
		SET status = 'completed', completed_at = NOW(), updated_at = NOW(),
		    total_records = $2, matched_records = $3, created_records = $4,
		    updated_records = $5, drift_count = $6
		WHERE id = $1
	`, id, total, matched, created, updated, drifts)
	return err
}

// FailSyncJob marks a job as failed.
func (r *ERPRepository) FailSyncJob(ctx context.Context, id int64, errMsg string) error {
	_, err := r.conn().Exec(ctx, `
		UPDATE erp.sync_job
		SET status = 'failed', completed_at = NOW(), updated_at = NOW(), error_message = $2
		WHERE id = $1
	`, id, errMsg)
	return err
}

// --- Sync drifts ---

// InsertSyncDrift records a drift detection.
func (r *ERPRepository) InsertSyncDrift(ctx context.Context, jobID int64, entityType string, entityID int64, entityCode *string, driftType string, localVal, remoteVal *string, fieldsChanged []string) error {
	_, err := r.conn().Exec(ctx, `
		INSERT INTO erp.sync_drift (sync_job_id, entity_type, entity_id, entity_code, drift_type, local_value, remote_value, fields_changed)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, jobID, entityType, entityID, entityCode, driftType, localVal, remoteVal, fieldsChanged)
	if err != nil {
		return fmt.Errorf("insert sync drift: %w", err)
	}
	return nil
}

// ListDriftsByJobID lists all drifts for a sync job.
func (r *ERPRepository) ListDriftsByJobID(ctx context.Context, jobID int64) ([]SyncDrift, error) {
	query := `
		SELECT id, sync_job_id, entity_type, entity_id, entity_code, drift_type,
		       local_value, remote_value, fields_changed, detected_at, resolved_at, resolution
		FROM erp.sync_drift WHERE sync_job_id = $1 ORDER BY detected_at
	`
	rows, err := r.conn().Query(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("list drifts: %w", err)
	}
	defer rows.Close()

	var drifts []SyncDrift
	for rows.Next() {
		var d SyncDrift
		var entityCode, localVal, remoteVal pgtype.Text
		var resolvedAt pgtype.Timestamptz
		var resolution pgtype.Text
		var fieldsChanged []string
		err := rows.Scan(&d.ID, &d.SyncJobID, &d.EntityType, &d.EntityID, &entityCode,
			&d.DriftType, &localVal, &remoteVal, &fieldsChanged, &d.DetectedAt, &resolvedAt, &resolution)
		if err != nil {
			return nil, fmt.Errorf("scan drift: %w", err)
		}
		if entityCode.Valid {
			d.EntityCode = &entityCode.String
		}
		if localVal.Valid {
			d.LocalValue = &localVal.String
		}
		if remoteVal.Valid {
			d.RemoteValue = &remoteVal.String
		}
		d.FieldsChanged = fieldsChanged
		if resolvedAt.Valid {
			d.ResolvedAt = &resolvedAt.Time
		}
		if resolution.Valid {
			d.Resolution = &resolution.String
		}
		drifts = append(drifts, d)
	}
	if drifts == nil {
		drifts = []SyncDrift{}
	}
	return drifts, nil
}

// --- Order dispatch ---

const orderDispatchColumns = "id, code, order_id, status, attempt_count, last_attempt_at, last_error, next_retry_at, dispatched_at, triggered_by, created_at, updated_at"

func scanOrderDispatch(row pgx.Row) (*OrderDispatch, error) {
	var d OrderDispatch
	var lastAttemptAt, dispatchedAt, nextRetryAt pgtype.Timestamptz
	var lastErr pgtype.Text
	var triggeredBy pgtype.Int8

	err := row.Scan(
		&d.ID, &d.Code, &d.OrderID, &d.Status, &d.AttemptCount,
		&lastAttemptAt, &lastErr, &nextRetryAt, &dispatchedAt, &triggeredBy,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if lastAttemptAt.Valid {
		d.LastAttemptAt = &lastAttemptAt.Time
	}
	if lastErr.Valid {
		d.LastError = &lastErr.String
	}
	if nextRetryAt.Valid {
		d.NextRetryAt = &nextRetryAt.Time
	}
	if dispatchedAt.Valid {
		d.DispatchedAt = &dispatchedAt.Time
	}
	if triggeredBy.Valid {
		d.TriggeredBy = &triggeredBy.Int64
	}
	return &d, nil
}

// CreateOrderDispatch creates a dispatch record for an order.
func (r *ERPRepository) CreateOrderDispatch(ctx context.Context, code string, orderID int64, triggeredBy *int64) (*OrderDispatch, error) {
	query := `
		INSERT INTO erp.order_dispatch (code, order_id, triggered_by, status)
		VALUES ($1, $2, $3, 'pending')
		RETURNING ` + orderDispatchColumns
	var tb pgtype.Int8
	if triggeredBy != nil {
		tb.Int64 = *triggeredBy
		tb.Valid = true
	}
	d, err := scanOrderDispatch(r.conn().QueryRow(ctx, query, code, orderID, tb))
	if err != nil {
		return nil, fmt.Errorf("create order dispatch: %w", err)
	}
	return d, nil
}

// GetDispatchByOrderID retrieves the dispatch record for an order.
func (r *ERPRepository) GetDispatchByOrderID(ctx context.Context, orderID int64) (*OrderDispatch, error) {
	query := `SELECT ` + orderDispatchColumns + ` FROM erp.order_dispatch WHERE order_id = $1 ORDER BY created_at DESC LIMIT 1`
	d, err := scanOrderDispatch(r.conn().QueryRow(ctx, query, orderID))
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dispatch by order: %w", err)
	}
	return d, nil
}

// GetDispatchByCode retrieves a dispatch record by code.
func (r *ERPRepository) GetDispatchByCode(ctx context.Context, code string) (*OrderDispatch, error) {
	query := `SELECT ` + orderDispatchColumns + ` FROM erp.order_dispatch WHERE code = $1`
	d, err := scanOrderDispatch(r.conn().QueryRow(ctx, query, code))
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dispatch by code: %w", err)
	}
	return d, nil
}

// UpdateDispatchDispatched marks a dispatch as successfully sent.
func (r *ERPRepository) UpdateDispatchDispatched(ctx context.Context, id int64) error {
	_, err := r.conn().Exec(ctx, `
		UPDATE erp.order_dispatch
		SET status = 'dispatched', dispatched_at = NOW(), updated_at = NOW(),
		    attempt_count = attempt_count + 1, last_attempt_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

// UpdateDispatchFailed marks a dispatch as failed with error.
func (r *ERPRepository) UpdateDispatchFailed(ctx context.Context, id int64, errMsg string, retryAt *time.Time) error {
	status := "failed"
	var nextRetry pgtype.Timestamptz
	if retryAt != nil {
		status = "retry"
		nextRetry.Time = *retryAt
		nextRetry.Valid = true
	}
	_, err := r.conn().Exec(ctx, `
		UPDATE erp.order_dispatch
		SET status = $2, last_error = $3, next_retry_at = $4, updated_at = NOW(),
		    attempt_count = attempt_count + 1, last_attempt_at = NOW()
		WHERE id = $1
	`, id, status, errMsg, nextRetry)
	return err
}

// --- Cross-module reads (for sync simulation) ---

// CountProducts returns the total number of products in the catalog.
func (r *ERPRepository) CountProducts(ctx context.Context) (int, error) {
	var count int
	err := r.conn().QueryRow(ctx, `SELECT COUNT(*) FROM catalog.product WHERE deleted_at IS NULL`).Scan(&count)
	return count, err
}

// CountStockLevels returns the total number of stock levels.
func (r *ERPRepository) CountStockLevels(ctx context.Context) (int, error) {
	var count int
	err := r.conn().QueryRow(ctx, `SELECT COUNT(*) FROM inventory.stock_level`).Scan(&count)
	return count, err
}

// OrderExists checks if an order exists and returns its status.
func (r *ERPRepository) OrderExists(ctx context.Context, orderID int64) (bool, string, error) {
	var status string
	err := r.conn().QueryRow(ctx, `SELECT status FROM commerce."order" WHERE id = $1`, orderID).Scan(&status)
	if err == pgx.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, status, nil
}
