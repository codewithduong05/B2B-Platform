-- Identity module schema
-- Creates tables for users, profiles, verification, roles, permissions

-- Ensure identity schema exists
CREATE SCHEMA IF NOT EXISTS identity;

-- Enum types
CREATE TYPE identity.user_type AS ENUM ('buyer', 'supplier', 'staff', 'platform');
CREATE TYPE identity.verification_status AS ENUM ('pending', 'approved', 'rejected', 'resubmitted');
CREATE TYPE identity.verification_decision AS ENUM ('approved', 'rejected');

-- Users table
CREATE TABLE identity.user (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,  -- opaque public identifier (ULID)
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    user_type identity.user_type NOT NULL DEFAULT 'buyer',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at TIMESTAMPTZ,
    failed_login_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_identity_user_email ON identity.user (email) WHERE deleted_at IS NULL;
CREATE INDEX idx_identity_user_code ON identity.user (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_identity_user_type ON identity.user (user_type) WHERE deleted_at IS NULL;

-- Buyer profiles
CREATE TABLE identity.buyer_profile (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES identity.user (id) ON DELETE CASCADE,
    code VARCHAR(26) NOT NULL UNIQUE,  -- opaque public identifier
    business_name VARCHAR(255) NOT NULL,
    trading_name VARCHAR(255),
    tax_id VARCHAR(50),
    registration_number VARCHAR(50),
    phone VARCHAR(50),
    website VARCHAR(255),
    industry VARCHAR(100),
    employee_count INT,
    annual_revenue_minor BIGINT,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    credit_limit_minor BIGINT,
    credit_terms_days INT,
    is_on_credit_hold BOOLEAN NOT NULL DEFAULT FALSE,
    credit_hold_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_identity_buyer_profile_user_id ON identity.buyer_profile (user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_identity_buyer_profile_code ON identity.buyer_profile (code) WHERE deleted_at IS NULL;

-- Supplier profiles
CREATE TABLE identity.supplier_profile (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES identity.user (id) ON DELETE CASCADE,
    code VARCHAR(26) NOT NULL UNIQUE,
    company_name VARCHAR(255) NOT NULL,
    trading_name VARCHAR(255),
    tax_id VARCHAR(50),
    registration_number VARCHAR(50),
    phone VARCHAR(50),
    email VARCHAR(255),
    website VARCHAR(255),
    address_line1 VARCHAR(255),
    address_line2 VARCHAR(255),
    city VARCHAR(100),
    state_province VARCHAR(100),
    postal_code VARCHAR(20),
    country CHAR(2) NOT NULL DEFAULT 'US',
    contact_person VARCHAR(255),
    contact_phone VARCHAR(50),
    contact_email VARCHAR(255),
    is_approved BOOLEAN NOT NULL DEFAULT FALSE,
    approved_at TIMESTAMPTZ,
    approved_by BIGINT REFERENCES identity.user (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_identity_supplier_profile_user_id ON identity.supplier_profile (user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_identity_supplier_profile_code ON identity.supplier_profile (code) WHERE deleted_at IS NULL;

-- Addresses (shared by buyers and suppliers)
CREATE TABLE identity.address (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    owner_user_id BIGINT NOT NULL REFERENCES identity.user (id) ON DELETE CASCADE,
    owner_type identity.user_type NOT NULL,
    label VARCHAR(100) NOT NULL,  -- e.g., 'default', 'warehouse', 'billing'
    recipient_name VARCHAR(255),
    company_name VARCHAR(255),
    line1 VARCHAR(255) NOT NULL,
    line2 VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    state_province VARCHAR(100),
    postal_code VARCHAR(20) NOT NULL,
    country CHAR(2) NOT NULL DEFAULT 'US',
    phone VARCHAR(50),
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    handling_class VARCHAR(20),  -- ambient, chilled, frozen
    delivery_instructions TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_identity_address_owner ON identity.address (owner_user_id, owner_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_identity_address_code ON identity.address (code) WHERE deleted_at IS NULL;

-- Verification applications
CREATE TABLE identity.verification_application (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    buyer_profile_id BIGINT NOT NULL REFERENCES identity.buyer_profile (id) ON DELETE CASCADE,
    status identity.verification_status NOT NULL DEFAULT 'pending',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at TIMESTAMPTZ,
    decided_by BIGINT REFERENCES identity.user (id),
    decision_reason TEXT,
    rejection_reason TEXT,
    licence_number VARCHAR(100),
    licence_expiry DATE,
    trading_name VARCHAR(255),
    business_address_line1 VARCHAR(255),
    business_address_line2 VARCHAR(255),
    business_city VARCHAR(100),
    business_state_province VARCHAR(100),
    business_postal_code VARCHAR(20),
    business_country CHAR(2) NOT NULL DEFAULT 'US',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_identity_verification_buyer ON identity.verification_application (buyer_profile_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_identity_verification_status ON identity.verification_application (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_identity_verification_code ON identity.verification_application (code) WHERE deleted_at IS NULL;

-- Verification documents
CREATE TABLE identity.verification_document (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    verification_application_id BIGINT NOT NULL REFERENCES identity.verification_application (id) ON DELETE CASCADE,
    document_type VARCHAR(50) NOT NULL,  -- 'licence', 'certificate', 'tax_id', 'other'
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_by BIGINT REFERENCES identity.user (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_identity_verification_doc_app ON identity.verification_document (verification_application_id) WHERE deleted_at IS NULL;

-- Purchase scopes (what a buyer is eligible to purchase)
CREATE TABLE identity.purchase_scope (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    buyer_profile_id BIGINT NOT NULL REFERENCES identity.buyer_profile (id) ON DELETE CASCADE,
    handling_class VARCHAR(20) NOT NULL,  -- 'ambient', 'chilled', 'frozen', 'restricted'
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by BIGINT REFERENCES identity.user (id),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_identity_purchase_scope_buyer ON identity.purchase_scope (buyer_profile_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_identity_purchase_scope_handling ON identity.purchase_scope (handling_class) WHERE deleted_at IS NULL;

-- Price list assignments
CREATE TABLE identity.price_list_assignment (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    buyer_profile_id BIGINT NOT NULL REFERENCES identity.buyer_profile (id) ON DELETE CASCADE,
    price_list_id BIGINT NOT NULL,  -- references pricing.price_list
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by BIGINT REFERENCES identity.user (id),
    effective_from DATE NOT NULL DEFAULT CURRENT_DATE,
    effective_to DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_identity_price_list_assignment_buyer ON identity.price_list_assignment (buyer_profile_id) WHERE deleted_at IS NULL;

-- Credit accounts
CREATE TABLE identity.credit_account (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    buyer_profile_id BIGINT NOT NULL REFERENCES identity.buyer_profile (id) ON DELETE CASCADE,
    limit_minor BIGINT NOT NULL DEFAULT 0,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    terms_days INT NOT NULL DEFAULT 30,
    current_exposure_minor BIGINT NOT NULL DEFAULT 0,
    available_minor BIGINT GENERATED ALWAYS AS (limit_minor - current_exposure_minor) STORED,
    is_on_hold BOOLEAN NOT NULL DEFAULT FALSE,
    hold_reason TEXT,
    hold_since TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_identity_credit_account_buyer ON identity.credit_account (buyer_profile_id) WHERE deleted_at IS NULL;

-- Roles
CREATE TABLE identity.role (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Permissions
CREATE TABLE identity.permission (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    resource VARCHAR(50) NOT NULL,  -- e.g., 'orders', 'products', 'users'
    action VARCHAR(50) NOT NULL,    -- e.g., 'view', 'create', 'update', 'delete'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- User roles (many-to-many)
CREATE TABLE identity.user_role (
    user_id BIGINT NOT NULL REFERENCES identity.user (id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES identity.role (id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by BIGINT REFERENCES identity.user (id),
    PRIMARY KEY (user_id, role_id)
);

-- Role permissions (many-to-many)
CREATE TABLE identity.role_permission (
    role_id BIGINT NOT NULL REFERENCES identity.role (id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES identity.permission (id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Refresh tokens
CREATE TABLE identity.refresh_token (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES identity.user (id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,  -- hashed refresh token
    family_id VARCHAR(26) NOT NULL,    -- for token rotation detection
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    revoked_reason VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    replaced_by_id BIGINT REFERENCES identity.refresh_token (id)
);

CREATE INDEX idx_identity_refresh_token_user ON identity.refresh_token (user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_identity_refresh_token_family ON identity.refresh_token (family_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_identity_refresh_token_expires ON identity.refresh_token (expires_at) WHERE revoked_at IS NULL;

-- Password reset tokens
CREATE TABLE identity.password_reset_token (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES identity.user (id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_identity_password_reset_user ON identity.password_reset_token (user_id) WHERE used_at IS NULL;
CREATE INDEX idx_identity_password_reset_expires ON identity.password_reset_token (expires_at) WHERE used_at IS NULL;

-- OTP codes (for registration verification, 2FA, etc.)
CREATE TABLE identity.otp_code (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES identity.user (id) ON DELETE CASCADE,
    otp_hash VARCHAR(255) NOT NULL,
    purpose VARCHAR(50) NOT NULL,  -- 'registration', 'login_2fa', 'password_reset'
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_identity_otp_user_purpose ON identity.otp_code (user_id, purpose) WHERE used_at IS NULL;

-- Sessions (for tracking active sessions)
CREATE TABLE identity.session (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES identity.user (id) ON DELETE CASCADE,
    refresh_token_id BIGINT REFERENCES identity.refresh_token (id) ON DELETE SET NULL,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_identity_session_user ON identity.session (user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_identity_session_expires ON identity.session (expires_at) WHERE revoked_at IS NULL;

-- Trigger to update updated_at timestamps
CREATE OR REPLACE FUNCTION identity.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_user_updated_at BEFORE UPDATE ON identity.user FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();
CREATE TRIGGER update_buyer_profile_updated_at BEFORE UPDATE ON identity.buyer_profile FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();
CREATE TRIGGER update_supplier_profile_updated_at BEFORE UPDATE ON identity.supplier_profile FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();
CREATE TRIGGER update_address_updated_at BEFORE UPDATE ON identity.address FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();
CREATE TRIGGER update_verification_application_updated_at BEFORE UPDATE ON identity.verification_application FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();
CREATE TRIGGER update_credit_account_updated_at BEFORE UPDATE ON identity.credit_account FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();
CREATE TRIGGER update_role_updated_at BEFORE UPDATE ON identity.role FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();
CREATE TRIGGER update_permission_updated_at BEFORE UPDATE ON identity.permission FOR EACH ROW EXECUTE FUNCTION identity.update_updated_at_column();