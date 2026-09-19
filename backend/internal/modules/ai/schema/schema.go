package schema

import (
	"encoding/json"
	"time"
)

type Provenance struct {
	PromptKey     string `json:"prompt_key"`
	PromptVersion int    `json:"prompt_version"`
	Model         string `json:"model"`
	InputHash     string `json:"input_hash"`
}

type ProposalResponse struct {
	ProposalID  string          `json:"proposal_id"`
	Status      string          `json:"status"`
	Confidence  float64         `json:"confidence"`
	Items       json.RawMessage `json:"items"`
	Unresolved  json.RawMessage `json:"unresolved"`
	Provenance  Provenance      `json:"provenance"`
	ExpiresAt   time.Time       `json:"expires_at"`
	CreatedAt   time.Time       `json:"created_at"`
	ConfirmedAt *time.Time      `json:"confirmed_at,omitempty"`
	RejectedAt  *time.Time      `json:"rejected_at,omitempty"`
}

type PromptResponse struct {
	ID          int64     `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Owner       *string   `json:"owner,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PromptVersionResponse struct {
	ID        int64     `json:"id"`
	PromptID  int64     `json:"prompt_id"`
	Version   int       `json:"version"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type ModelRoutingResponse struct {
	ID         int64           `json:"id"`
	FeatureKey string          `json:"feature_key"`
	Provider   string          `json:"provider"`
	ModelName  string          `json:"model_name"`
	Config     json.RawMessage `json:"config,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type UsageRecordResponse struct {
	ID            int64     `json:"id"`
	ProposalID    *int64    `json:"proposal_id,omitempty"`
	PromptKey     string    `json:"prompt_key"`
	PromptVersion int       `json:"prompt_version"`
	Model         string    `json:"model"`
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	LatencyMS     int       `json:"latency_ms"`
	CostMicros    int64     `json:"cost_micros"`
	TenantID      *int64    `json:"tenant_id,omitempty"`
	Feature       string    `json:"feature"`
	CreatedAt     time.Time `json:"created_at"`
}

type BudgetResponse struct {
	ID            int64     `json:"id"`
	ScopeKey      string    `json:"scope_key"`
	PeriodStart   time.Time `json:"period_start"`
	PeriodEnd     time.Time `json:"period_end"`
	CeilingMicros int64     `json:"ceiling_micros"`
	UsedMicros    int64     `json:"used_micros"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreatePromptRequest struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Owner       *string `json:"owner,omitempty"`
}

type CreatePromptVersionRequest struct {
	Content string `json:"content"`
}

type CreateModelRoutingRequest struct {
	FeatureKey string          `json:"feature_key"`
	Provider   string          `json:"provider"`
	ModelName  string          `json:"model_name"`
	Config     json.RawMessage `json:"config,omitempty"`
}

type UpdateModelRoutingRequest struct {
	Provider  *string         `json:"provider,omitempty"`
	ModelName *string         `json:"model_name,omitempty"`
	Config    json.RawMessage `json:"config,omitempty"`
}

type CreateBudgetRequest struct {
	ScopeKey      string `json:"scope_key"`
	PeriodStart   string `json:"period_start"`
	PeriodEnd     string `json:"period_end"`
	CeilingMicros int64  `json:"ceiling_micros"`
}

type UpdateBudgetRequest struct {
	CeilingMicros *int64 `json:"ceiling_micros,omitempty"`
}
