-- TASK-017: M5 platform notifications — templates, delivery, tracking.
--
-- Notification templates define reusable message formats for email and SMS.
-- Notification records track individual deliveries with status and metadata.

CREATE TABLE platform.notification_template (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('email', 'sms')),
    subject VARCHAR(500),
    body_template TEXT NOT NULL,
    variables JSONB NOT NULL DEFAULT '[]'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_by BIGINT REFERENCES identity.user (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_notification_template_code ON platform.notification_template (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_notification_template_channel ON platform.notification_template (channel) WHERE deleted_at IS NULL;

CREATE TABLE platform.notification (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    template_id BIGINT REFERENCES platform.notification_template (id) ON DELETE RESTRICT,
    recipient_type VARCHAR(20) NOT NULL CHECK (recipient_type IN ('buyer', 'user')),
    recipient_id BIGINT NOT NULL,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('email', 'sms')),
    recipient_address VARCHAR(500) NOT NULL,
    subject VARCHAR(500),
    body TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'delivered', 'failed', 'bounced')),
    provider_message_id VARCHAR(255),
    provider_response TEXT,
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_notification_recipient ON platform.notification (recipient_type, recipient_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_notification_status ON platform.notification (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_notification_template ON platform.notification (template_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_notification_created ON platform.notification (created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE platform.notification_event (
    id BIGSERIAL PRIMARY KEY,
    notification_id BIGINT NOT NULL REFERENCES platform.notification (id) ON DELETE CASCADE,
    event_type VARCHAR(30) NOT NULL CHECK (event_type IN ('created', 'sent', 'delivered', 'failed', 'bounced', 'retry')),
    provider_event_id VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notification_event_notification ON platform.notification_event (notification_id);
CREATE INDEX idx_notification_event_type ON platform.notification_event (event_type);
