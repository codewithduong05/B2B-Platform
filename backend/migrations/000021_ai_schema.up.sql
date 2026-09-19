CREATE TABLE IF NOT EXISTS ai.prompt (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    owner VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai.prompt_version (
    id BIGSERIAL PRIMARY KEY,
    prompt_id BIGINT NOT NULL REFERENCES ai.prompt(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(prompt_id, version)
);

CREATE TABLE IF NOT EXISTS ai.model_routing (
    id BIGSERIAL PRIMARY KEY,
    feature_key VARCHAR(100) NOT NULL UNIQUE,
    provider VARCHAR(100) NOT NULL,
    model_name VARCHAR(100) NOT NULL,
    config JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai.proposal (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    feature VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'proposed' CHECK (status IN ('proposed', 'confirmed', 'rejected', 'expired')),
    confidence DECIMAL(3,2) CHECK (confidence >= 0 AND confidence <= 1),
    items JSONB NOT NULL DEFAULT '[]'::jsonb,
    unresolved JSONB NOT NULL DEFAULT '[]'::jsonb,
    provenance JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    tenant_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_proposal_tenant ON ai.proposal(tenant_id);
CREATE INDEX IF NOT EXISTS idx_proposal_feature ON ai.proposal(feature);
CREATE INDEX IF NOT EXISTS idx_proposal_status ON ai.proposal(status);
CREATE INDEX IF NOT EXISTS idx_proposal_expires ON ai.proposal(expires_at);

CREATE TABLE IF NOT EXISTS ai.usage_record (
    id BIGSERIAL PRIMARY KEY,
    proposal_id BIGINT REFERENCES ai.proposal(id) ON DELETE SET NULL,
    prompt_key VARCHAR(100) NOT NULL,
    prompt_version INTEGER NOT NULL,
    model VARCHAR(200) NOT NULL,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    latency_ms INTEGER NOT NULL DEFAULT 0,
    cost_micros BIGINT NOT NULL DEFAULT 0,
    tenant_id BIGINT,
    feature VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_usage_tenant ON ai.usage_record(tenant_id);
CREATE INDEX IF NOT EXISTS idx_usage_feature ON ai.usage_record(feature);
CREATE INDEX IF NOT EXISTS idx_usage_created ON ai.usage_record(created_at);

CREATE TABLE IF NOT EXISTS ai.budget (
    id BIGSERIAL PRIMARY KEY,
    scope_key VARCHAR(200) NOT NULL UNIQUE,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    ceiling_micros BIGINT NOT NULL,
    used_micros BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_budget_period ON ai.budget(period_start, period_end);
