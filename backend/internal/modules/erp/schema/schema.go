package schema

import "time"

// SyncJobResponse represents a sync job status.
type SyncJobResponse struct {
	ID             int64           `json:"id"`
	Code           string          `json:"code"`
	JobType        string          `json:"job_type"`
	Status         string          `json:"status"`
	TriggeredBy    *int64          `json:"triggered_by,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	TotalRecords   int             `json:"total_records"`
	MatchedRecords int             `json:"matched_records"`
	CreatedRecords int             `json:"created_records"`
	UpdatedRecords int             `json:"updated_records"`
	DriftCount     int             `json:"drift_count"`
	ErrorMessage   *string         `json:"error_message,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Drifts         []DriftResponse `json:"drifts,omitempty"`
}

// SyncJobDetailResponse is the response for sync job creation/detail with simplified fields.
type SyncJobDetailResponse struct {
	Code         string  `json:"code"`
	JobType      string  `json:"job_type"`
	Status       string  `json:"status"`
	TotalItems   int     `json:"total_items"`
	MatchedItems int     `json:"matched_items"`
	DriftCount   int     `json:"drift_count"`
	ErrorMessage *string `json:"error_message,omitempty"`
}

// DriftResponse represents a drift detection record.
type DriftResponse struct {
	ID            int64       `json:"id"`
	SyncJobID     int64       `json:"sync_job_id"`
	EntityType    string      `json:"entity_type"`
	EntityID      int64       `json:"entity_id"`
	EntityCode    *string     `json:"entity_code,omitempty"`
	DriftType     string      `json:"drift_type"`
	LocalValue    interface{} `json:"local_value,omitempty"`
	RemoteValue   interface{} `json:"remote_value,omitempty"`
	FieldsChanged []string    `json:"fields_changed,omitempty"`
	DetectedAt    time.Time   `json:"detected_at"`
	ResolvedAt    *time.Time  `json:"resolved_at,omitempty"`
	Resolution    *string     `json:"resolution,omitempty"`
}

// OrderDispatchResponse represents an order dispatch record.
type OrderDispatchResponse struct {
	ID            int64      `json:"id"`
	Code          string     `json:"code"`
	OrderID       int64      `json:"order_id"`
	Status        string     `json:"status"`
	AttemptCount  int        `json:"attempt_count"`
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty"`
	LastError     *string    `json:"last_error,omitempty"`
	NextRetryAt   *time.Time `json:"next_retry_at,omitempty"`
	DispatchedAt  *time.Time `json:"dispatched_at,omitempty"`
	TriggeredBy   *int64     `json:"triggered_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// TriggerSyncRequest is the request body for triggering a sync.
type TriggerSyncRequest struct {
	Force bool `json:"force,omitempty"`
}

// DispatchOrderRequest is the request body for forcing order dispatch.
type DispatchOrderRequest struct {
	Force bool `json:"force,omitempty"`
}

// WebhookEventResponse represents an inbound ERP webhook event.
type WebhookEventResponse struct {
	ID              int64      `json:"id"`
	Provider        string     `json:"provider"`
	ProviderEventID string     `json:"provider_event_id"`
	Topic           string     `json:"topic"`
	Status          string     `json:"status"`
	ReceivedAt      time.Time  `json:"received_at"`
	AppliedAt       *time.Time `json:"applied_at,omitempty"`
}
