package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/atlas-platform/backend/internal/database"
)

type AIRepository struct {
	db *database.DB
}

func NewAIRepository(db *database.DB) *AIRepository {
	return &AIRepository{db: db}
}

type Prompt struct {
	ID          int64
	Key         string
	Name        string
	Description *string
	Owner       *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PromptVersion struct {
	ID        int64
	PromptID  int64
	Version   int
	Content   string
	Status    string
	CreatedAt time.Time
}

type ModelRouting struct {
	ID         int64
	FeatureKey string
	Provider   string
	ModelName  string
	Config     []byte
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Proposal struct {
	ID          int64
	Code        string
	Feature     string
	Status      string
	Confidence  float64
	Items       []byte
	Unresolved  []byte
	Provenance  []byte
	ExpiresAt   time.Time
	TenantID    *int64
	CreatedAt   time.Time
	ConfirmedAt *time.Time
	RejectedAt  *time.Time
}

type UsageRecord struct {
	ID            int64
	ProposalID    *int64
	PromptKey     string
	PromptVersion int
	Model         string
	InputTokens   int
	OutputTokens  int
	LatencyMS     int
	CostMicros    int64
	TenantID      *int64
	Feature       string
	CreatedAt     time.Time
}

type Budget struct {
	ID            int64
	ScopeKey      string
	PeriodStart   time.Time
	PeriodEnd     time.Time
	CeilingMicros int64
	UsedMicros    int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (r *AIRepository) CreatePrompt(ctx context.Context, key, name string, description, owner *string) (*Prompt, error) {
	var p Prompt
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO ai.prompt (key, name, description, owner)
		VALUES ($1, $2, $3, $4)
		RETURNING id, key, name, description, owner, created_at, updated_at
	`, key, name, description, owner).Scan(
		&p.ID, &p.Key, &p.Name, &p.Description, &p.Owner, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *AIRepository) GetPromptByKey(ctx context.Context, key string) (*Prompt, error) {
	var p Prompt
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, key, name, description, owner, created_at, updated_at
		FROM ai.prompt
		WHERE key = $1
	`, key).Scan(&p.ID, &p.Key, &p.Name, &p.Description, &p.Owner, &p.CreatedAt, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *AIRepository) ListPrompts(ctx context.Context, limit, offset int) ([]Prompt, int, error) {
	var total int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM ai.prompt`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, key, name, description, owner, created_at, updated_at
		FROM ai.prompt
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Prompt
	for rows.Next() {
		var p Prompt
		if err := rows.Scan(&p.ID, &p.Key, &p.Name, &p.Description, &p.Owner, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, nil
}

func (r *AIRepository) CreatePromptVersion(ctx context.Context, promptID int64, version int, content, status string) (*PromptVersion, error) {
	var pv PromptVersion
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO ai.prompt_version (prompt_id, version, content, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, prompt_id, version, content, status, created_at
	`, promptID, version, content, status).Scan(
		&pv.ID, &pv.PromptID, &pv.Version, &pv.Content, &pv.Status, &pv.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &pv, nil
}

func (r *AIRepository) GetLatestPromptVersion(ctx context.Context, promptID int64) (*PromptVersion, error) {
	var pv PromptVersion
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, prompt_id, version, content, status, created_at
		FROM ai.prompt_version
		WHERE prompt_id = $1
		ORDER BY version DESC
		LIMIT 1
	`, promptID).Scan(&pv.ID, &pv.PromptID, &pv.Version, &pv.Content, &pv.Status, &pv.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &pv, nil
}

func (r *AIRepository) CreateProposal(ctx context.Context, code, feature string, confidence float64, items, unresolved, provenance []byte, expiresAt time.Time, tenantID *int64) (*Proposal, error) {
	var p Proposal
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO ai.proposal (code, feature, confidence, items, unresolved, provenance, expires_at, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, code, feature, status, confidence, items, unresolved, provenance, expires_at, tenant_id, created_at
	`, code, feature, confidence, items, unresolved, provenance, expiresAt, tenantID).Scan(
		&p.ID, &p.Code, &p.Feature, &p.Status, &p.Confidence, &p.Items, &p.Unresolved, &p.Provenance, &p.ExpiresAt, &p.TenantID, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *AIRepository) GetProposalByCode(ctx context.Context, code string) (*Proposal, error) {
	var p Proposal
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, code, feature, status, confidence, items, unresolved, provenance, expires_at, tenant_id, created_at, confirmed_at, rejected_at
		FROM ai.proposal
		WHERE code = $1
	`, code).Scan(
		&p.ID, &p.Code, &p.Feature, &p.Status, &p.Confidence, &p.Items, &p.Unresolved, &p.Provenance, &p.ExpiresAt, &p.TenantID, &p.CreatedAt, &p.ConfirmedAt, &p.RejectedAt)
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *AIRepository) ConfirmProposal(ctx context.Context, code string) (*Proposal, error) {
	var p Proposal
	err := r.db.Pool.QueryRow(ctx, `
		UPDATE ai.proposal
		SET status = 'confirmed', confirmed_at = NOW()
		WHERE code = $1 AND status = 'proposed'
		RETURNING id, code, feature, status, confidence, items, unresolved, provenance, expires_at, tenant_id, created_at, confirmed_at, rejected_at
	`, code).Scan(
		&p.ID, &p.Code, &p.Feature, &p.Status, &p.Confidence, &p.Items, &p.Unresolved, &p.Provenance, &p.ExpiresAt, &p.TenantID, &p.CreatedAt, &p.ConfirmedAt, &p.RejectedAt)
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *AIRepository) RejectProposal(ctx context.Context, code string) (*Proposal, error) {
	var p Proposal
	err := r.db.Pool.QueryRow(ctx, `
		UPDATE ai.proposal
		SET status = 'rejected', rejected_at = NOW()
		WHERE code = $1 AND status = 'proposed'
		RETURNING id, code, feature, status, confidence, items, unresolved, provenance, expires_at, tenant_id, created_at, confirmed_at, rejected_at
	`, code).Scan(
		&p.ID, &p.Code, &p.Feature, &p.Status, &p.Confidence, &p.Items, &p.Unresolved, &p.Provenance, &p.ExpiresAt, &p.TenantID, &p.CreatedAt, &p.ConfirmedAt, &p.RejectedAt)
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *AIRepository) RecordUsage(ctx context.Context, proposalID *int64, promptKey string, promptVersion int, model string, inputTokens, outputTokens, latencyMS int, costMicros int64, tenantID *int64, feature string) (*UsageRecord, error) {
	var u UsageRecord
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO ai.usage_record (proposal_id, prompt_key, prompt_version, model, input_tokens, output_tokens, latency_ms, cost_micros, tenant_id, feature)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, proposal_id, prompt_key, prompt_version, model, input_tokens, output_tokens, latency_ms, cost_micros, tenant_id, feature, created_at
	`, proposalID, promptKey, promptVersion, model, inputTokens, outputTokens, latencyMS, costMicros, tenantID, feature).Scan(
		&u.ID, &u.ProposalID, &u.PromptKey, &u.PromptVersion, &u.Model, &u.InputTokens, &u.OutputTokens, &u.LatencyMS, &u.CostMicros, &u.TenantID, &u.Feature, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *AIRepository) ListUsage(ctx context.Context, feature string, tenantID *int64, limit, offset int) ([]UsageRecord, int, error) {
	where := "WHERE ($1 = '' OR feature = $1) AND ($2 IS NULL OR tenant_id = $2)"
	var total int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM ai.usage_record `+where, feature, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, proposal_id, prompt_key, prompt_version, model, input_tokens, output_tokens, latency_ms, cost_micros, tenant_id, feature, created_at
		FROM ai.usage_record
		`+where+`
		ORDER BY created_at DESC, id DESC
		LIMIT $3 OFFSET $4
	`, feature, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []UsageRecord
	for rows.Next() {
		var u UsageRecord
		if err := rows.Scan(&u.ID, &u.ProposalID, &u.PromptKey, &u.PromptVersion, &u.Model, &u.InputTokens, &u.OutputTokens, &u.LatencyMS, &u.CostMicros, &u.TenantID, &u.Feature, &u.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	return out, total, nil
}

func (r *AIRepository) CreateModelRouting(ctx context.Context, featureKey, provider, modelName string, config []byte) (*ModelRouting, error) {
	var m ModelRouting
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO ai.model_routing (feature_key, provider, model_name, config)
		VALUES ($1, $2, $3, $4)
		RETURNING id, feature_key, provider, model_name, config, created_at, updated_at
	`, featureKey, provider, modelName, config).Scan(
		&m.ID, &m.FeatureKey, &m.Provider, &m.ModelName, &m.Config, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *AIRepository) GetModelRoutingByFeature(ctx context.Context, featureKey string) (*ModelRouting, error) {
	var m ModelRouting
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, feature_key, provider, model_name, config, created_at, updated_at
		FROM ai.model_routing
		WHERE feature_key = $1
	`, featureKey).Scan(&m.ID, &m.FeatureKey, &m.Provider, &m.ModelName, &m.Config, &m.CreatedAt, &m.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *AIRepository) ListModelRouting(ctx context.Context, limit, offset int) ([]ModelRouting, int, error) {
	var total int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM ai.model_routing`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, feature_key, provider, model_name, config, created_at, updated_at
		FROM ai.model_routing
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []ModelRouting
	for rows.Next() {
		var m ModelRouting
		if err := rows.Scan(&m.ID, &m.FeatureKey, &m.Provider, &m.ModelName, &m.Config, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, m)
	}
	return out, total, nil
}

func (r *AIRepository) UpdateModelRouting(ctx context.Context, featureKey string, provider, modelName *string, config []byte) (*ModelRouting, error) {
	var m ModelRouting
	err := r.db.Pool.QueryRow(ctx, `
		UPDATE ai.model_routing
		SET provider = COALESCE($2, provider),
		    model_name = COALESCE($3, model_name),
		    config = COALESCE($4, config),
		    updated_at = NOW()
		WHERE feature_key = $1
		RETURNING id, feature_key, provider, model_name, config, created_at, updated_at
	`, featureKey, provider, modelName, config).Scan(
		&m.ID, &m.FeatureKey, &m.Provider, &m.ModelName, &m.Config, &m.CreatedAt, &m.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *AIRepository) CreateBudget(ctx context.Context, scopeKey string, periodStart, periodEnd time.Time, ceilingMicros int64) (*Budget, error) {
	var b Budget
	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO ai.budget (scope_key, period_start, period_end, ceiling_micros)
		VALUES ($1, $2, $3, $4)
		RETURNING id, scope_key, period_start, period_end, ceiling_micros, used_micros, created_at, updated_at
	`, scopeKey, periodStart, periodEnd, ceilingMicros).Scan(
		&b.ID, &b.ScopeKey, &b.PeriodStart, &b.PeriodEnd, &b.CeilingMicros, &b.UsedMicros, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *AIRepository) GetBudgetByScope(ctx context.Context, scopeKey string) (*Budget, error) {
	var b Budget
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, scope_key, period_start, period_end, ceiling_micros, used_micros, created_at, updated_at
		FROM ai.budget
		WHERE scope_key = $1
	`, scopeKey).Scan(&b.ID, &b.ScopeKey, &b.PeriodStart, &b.PeriodEnd, &b.CeilingMicros, &b.UsedMicros, &b.CreatedAt, &b.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *AIRepository) UpdateBudgetUsage(ctx context.Context, scopeKey string, costMicros int64) (*Budget, error) {
	var b Budget
	err := r.db.Pool.QueryRow(ctx, `
		UPDATE ai.budget
		SET used_micros = used_micros + $2, updated_at = NOW()
		WHERE scope_key = $1
		RETURNING id, scope_key, period_start, period_end, ceiling_micros, used_micros, created_at, updated_at
	`, scopeKey, costMicros).Scan(
		&b.ID, &b.ScopeKey, &b.PeriodStart, &b.PeriodEnd, &b.CeilingMicros, &b.UsedMicros, &b.CreatedAt, &b.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, database.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *AIRepository) ListBudgets(ctx context.Context, limit, offset int) ([]Budget, int, error) {
	var total int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM ai.budget`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, scope_key, period_start, period_end, ceiling_micros, used_micros, created_at, updated_at
		FROM ai.budget
		ORDER BY period_start DESC, id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Budget
	for rows.Next() {
		var b Budget
		if err := rows.Scan(&b.ID, &b.ScopeKey, &b.PeriodStart, &b.PeriodEnd, &b.CeilingMicros, &b.UsedMicros, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, b)
	}
	return out, total, nil
}

// Helper to marshal JSON for provenance
func MarshalProvenance(promptKey string, promptVersion int, model, inputHash string) ([]byte, error) {
	p := map[string]interface{}{
		"prompt_key":     promptKey,
		"prompt_version": promptVersion,
		"model":          model,
		"input_hash":     inputHash,
	}
	return json.Marshal(p)
}
