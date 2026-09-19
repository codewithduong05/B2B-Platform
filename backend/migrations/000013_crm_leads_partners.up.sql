-- TASK-010: M4 CRM slice 1 — partners, referral codes, leads.
--
-- Attribution is derived (leads counted per code/partner), never stored as
-- counters. Lead assignment/notes/buyer-link live on the lead row (no
-- extra tables). Buyer links are FK-guarded; no cross-module reads needed.

CREATE SCHEMA IF NOT EXISTS crm;

CREATE TABLE crm.partner (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_crm_partner_slug ON crm.partner (slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_crm_partner_code ON crm.partner (code) WHERE deleted_at IS NULL;

CREATE TABLE crm.referral_code (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    partner_id BIGINT NOT NULL REFERENCES crm.partner (id) ON DELETE RESTRICT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_crm_referral_code ON crm.referral_code (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_crm_referral_partner ON crm.referral_code (partner_id) WHERE deleted_at IS NULL;

-- Status: new -> assigned -> contacted -> converted | closed.
-- converted/closed are terminal. buyer_id is set on conversion (FK-guarded).
CREATE TABLE crm.lead (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    contact_name VARCHAR(255) NOT NULL,
    business_name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(64),
    message TEXT,
    status VARCHAR(16) NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'assigned', 'contacted', 'converted', 'closed')),
    partner_id BIGINT REFERENCES crm.partner (id) ON DELETE SET NULL,
    referral_id BIGINT REFERENCES crm.referral_code (id) ON DELETE SET NULL,
    assigned_to BIGINT,
    buyer_id BIGINT REFERENCES identity.buyer_profile (id) ON DELETE SET NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_crm_lead_status ON crm.lead (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_crm_lead_code ON crm.lead (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_crm_lead_partner ON crm.lead (partner_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_crm_lead_referral ON crm.lead (referral_id) WHERE deleted_at IS NULL;
