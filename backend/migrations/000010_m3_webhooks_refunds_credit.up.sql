-- TASK-007: M3 remainder — webhook dedupe store, refunds, credit accounts.
--
-- Refund approval policy (A7.3): amounts at or above
-- payments.refund_approval_threshold() minor units require a second
-- approver distinct from the requester. Threshold lives in code
-- (RefundApprovalThreshold = 1000000); the table records both actors.

-- Provider webhook deliveries, deduplicated by provider event ID.
CREATE TABLE payments.webhook_event (
    id BIGSERIAL PRIMARY KEY,
    provider VARCHAR(64) NOT NULL,
    provider_event_id VARCHAR(128) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload_hash CHAR(64) NOT NULL,
    intent_code VARCHAR(26),
    status VARCHAR(16) NOT NULL DEFAULT 'received' CHECK (status IN ('received', 'applied', 'unmatched', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_event_id)
);

CREATE INDEX idx_payments_webhook_provider ON payments.webhook_event (provider, created_at DESC);

-- B2B refunds against succeeded intents. Balance restoration happens on
-- approval (oldest-first invoice order is NOT re-opened; instead the refund
-- amount is credited back onto issued invoices with remaining... see
-- service notes: refunds restore balance up to the invoice total).
CREATE TABLE payments.payment_refund (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    intent_id BIGINT NOT NULL REFERENCES payments.payment_intent (id) ON DELETE RESTRICT,
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    reason TEXT NOT NULL CHECK (char_length(reason) > 0),
    status VARCHAR(16) NOT NULL DEFAULT 'pending_approval' CHECK (status IN ('pending_approval', 'approved', 'rejected', 'applied')),
    requested_by BIGINT,
    approved_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_payments_refund_intent ON payments.payment_refund (intent_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payments_refund_code ON payments.payment_refund (code) WHERE deleted_at IS NULL;

-- Buyer credit terms. Exposure is always derived (sum of issued invoice
-- balances) and never stored.
CREATE TABLE payments.credit_account (
    buyer_id BIGINT PRIMARY KEY REFERENCES identity.buyer_profile (id) ON DELETE CASCADE,
    credit_limit_minor BIGINT NOT NULL DEFAULT 0 CHECK (credit_limit_minor >= 0),
    terms VARCHAR(64) NOT NULL DEFAULT 'prepaid',
    on_hold BOOLEAN NOT NULL DEFAULT FALSE,
    hold_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
