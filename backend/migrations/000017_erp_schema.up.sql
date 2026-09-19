-- M5 ERP Slice 1: Sync jobs, drift detection, order dispatch

CREATE SCHEMA IF NOT EXISTS erp;

-- Inbound ERP webhook events (deduplication and audit)
CREATE TABLE erp.erp_webhook_event (
    id BIGSERIAL PRIMARY KEY,
    provider VARCHAR(50) NOT NULL,
    provider_event_id VARCHAR(255) NOT NULL,
    topic VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'received' CHECK (status IN ('received', 'applied', 'unmatched', 'failed')),
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_event_id)
);

CREATE INDEX idx_webhook_event_status ON erp.erp_webhook_event (status) WHERE status IN ('received', 'failed');
CREATE INDEX idx_webhook_event_received ON erp.erp_webhook_event (received_at DESC);

-- Sync jobs (catalog and stock synchronization)
CREATE TABLE erp.sync_job (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    job_type VARCHAR(20) NOT NULL CHECK (job_type IN ('catalog', 'stock')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    triggered_by BIGINT REFERENCES identity.user (id) ON DELETE SET NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    total_records INTEGER NOT NULL DEFAULT 0,
    matched_records INTEGER NOT NULL DEFAULT 0,
    created_records INTEGER NOT NULL DEFAULT 0,
    updated_records INTEGER NOT NULL DEFAULT 0,
    drift_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sync_job_status ON erp.sync_job (status) WHERE status IN ('pending', 'running');
CREATE INDEX idx_sync_job_type ON erp.sync_job (job_type, created_at DESC);

-- Drift detection records (differences between local and ERP data)
CREATE TABLE erp.sync_drift (
    id BIGSERIAL PRIMARY KEY,
    sync_job_id BIGINT NOT NULL REFERENCES erp.sync_job (id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id BIGINT NOT NULL,
    entity_code VARCHAR(26),
    drift_type VARCHAR(30) NOT NULL CHECK (drift_type IN ('missing_local', 'missing_remote', 'field_mismatch')),
    local_value JSONB,
    remote_value JSONB,
    fields_changed TEXT[],
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolution VARCHAR(50) CHECK (resolution IN ('accepted_local', 'accepted_remote', 'manual'))
);

CREATE INDEX idx_sync_drift_job ON erp.sync_drift (sync_job_id);
CREATE INDEX idx_sync_drift_entity ON erp.sync_drift (entity_type, entity_id);
CREATE INDEX idx_sync_drift_unresolved ON erp.sync_drift (resolved_at) WHERE resolved_at IS NULL;

-- Order dispatch log (tracking dispatch attempts to ERP)
CREATE TABLE erp.order_dispatch (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    order_id BIGINT NOT NULL REFERENCES commerce."order" (id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'dispatched', 'failed', 'retry')),
    attempt_count INTEGER NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    last_error TEXT,
    next_retry_at TIMESTAMPTZ,
    dispatched_at TIMESTAMPTZ,
    triggered_by BIGINT REFERENCES identity.user (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_order_dispatch_order ON erp.order_dispatch (order_id);
CREATE INDEX idx_order_dispatch_status ON erp.order_dispatch (status) WHERE status IN ('pending', 'retry', 'failed');
CREATE INDEX idx_order_dispatch_retry ON erp.order_dispatch (next_retry_at) WHERE status = 'retry';
