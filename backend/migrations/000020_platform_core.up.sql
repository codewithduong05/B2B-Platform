-- TASK-018: M5 Platform Core — Feature Flags, Audit Log, Integration Traffic, Media, Reference Data

CREATE TABLE platform.feature_flag (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    updated_by BIGINT REFERENCES identity."user" (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_feature_flag_key ON platform.feature_flag (key);
CREATE INDEX idx_feature_flag_enabled ON platform.feature_flag (enabled) WHERE enabled = TRUE;

CREATE TABLE platform.audit_log (
    id BIGSERIAL PRIMARY KEY,
    actor_id BIGINT,
    actor_type VARCHAR(20) NOT NULL CHECK (actor_type IN ('user', 'buyer', 'system')),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id BIGINT,
    resource_code VARCHAR(50),
    metadata JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_actor ON platform.audit_log (actor_type, actor_id);
CREATE INDEX idx_audit_log_resource ON platform.audit_log (resource_type, resource_id);
CREATE INDEX idx_audit_log_action ON platform.audit_log (action);
CREATE INDEX idx_audit_log_created ON platform.audit_log (created_at DESC);

CREATE TABLE platform.integration_traffic (
    id BIGSERIAL PRIMARY KEY,
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    provider VARCHAR(100) NOT NULL,
    endpoint VARCHAR(500) NOT NULL,
    method VARCHAR(10),
    status_code INTEGER,
    payload_hash VARCHAR(64),
    request_headers JSONB,
    response_headers JSONB,
    error_message TEXT,
    duration_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_integration_traffic_direction ON platform.integration_traffic (direction);
CREATE INDEX idx_integration_traffic_provider ON platform.integration_traffic (provider);
CREATE INDEX idx_integration_traffic_created ON platform.integration_traffic (created_at DESC);

CREATE TABLE platform.media_upload (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    filename VARCHAR(500) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT,
    upload_url VARCHAR(1000) NOT NULL,
    download_url VARCHAR(1000),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'uploaded', 'failed', 'expired')),
    expires_at TIMESTAMPTZ NOT NULL,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_media_upload_code ON platform.media_upload (code);
CREATE INDEX idx_media_upload_status ON platform.media_upload (status);

CREATE TABLE platform.reference_region (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    country VARCHAR(100) NOT NULL DEFAULT 'VN',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reference_region_active ON platform.reference_region (is_active, name);

INSERT INTO platform.reference_region (code, name, country) VALUES
    ('HN', 'Hà Nội', 'VN'),
    ('HCM', 'TP. Hồ Chí Minh', 'VN'),
    ('DN', 'Đà Nẵng', 'VN'),
    ('HP', 'Hải Phòng', 'VN'),
    ('CT', 'Cần Thơ', 'VN'),
    ('BD', 'Bình Dương', 'VN'),
    ('DNa', 'Đồng Nai', 'VN'),
    ('KH', 'Khánh Hòa', 'VN'),
    ('LD', 'Lâm Đồng', 'VN'),
    ('QNa', 'Quảng Nam', 'VN');
