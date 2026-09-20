-- M1 Identity: Identity schema and tables

CREATE SCHEMA IF NOT EXISTS identity;

-- Users table (both platform staff and buyers)
CREATE TABLE identity."user" (
    id              BIGSERIAL PRIMARY KEY,
    code            VARCHAR(26) NOT NULL UNIQUE,  -- usr_ prefix
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    user_type       VARCHAR(20) NOT NULL CHECK (user_type IN ('buyer', 'staff', 'admin')),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_identity_user_email ON identity."user" (email) WHERE deleted_at IS NULL;
CREATE INDEX idx_identity_user_code ON identity."user" (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_identity_user_type ON identity."user" (user_type) WHERE deleted_at IS NULL;

-- Buyer profiles (extends user for business information)
CREATE TABLE identity.buyer_profile (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES identity."user" (id) ON DELETE CASCADE,
    code            VARCHAR(26) NOT NULL UNIQUE,  -- buy_ prefix
    business_name   VARCHAR(255) NOT NULL,
    tax_id          VARCHAR(50),
    phone           VARCHAR(50),
    currency        CHAR(3) NOT NULL DEFAULT 'VND',
    purchase_scope  VARCHAR(100),  -- restricted/general/special
    verification_status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (verification_status IN ('pending', 'approved', 'rejected', 'resubmit')),
    credit_limit_minor BIGINT NOT NULL DEFAULT 0,
    credit_terms_days INT NOT NULL DEFAULT 0,
    price_list_id   BIGINT,  -- FK to pricing.price_list (added later)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_identity_buyer_profile_user_id ON identity.buyer_profile (user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_identity_buyer_profile_code ON identity.buyer_profile (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_identity_buyer_profile_verification ON identity.buyer_profile (verification_status) WHERE deleted_at IS NULL;

-- Buyer addresses
CREATE TABLE identity.buyer_address (
    id              BIGSERIAL PRIMARY KEY,
    buyer_id        BIGINT NOT NULL REFERENCES identity.buyer_profile (id) ON DELETE CASCADE,
    code            VARCHAR(26) NOT NULL UNIQUE,  -- addr_ prefix
    label           VARCHAR(100) NOT NULL,  -- e.g., "HQ", "Warehouse 1"
    recipient_name  VARCHAR(255) NOT NULL,
    phone           VARCHAR(50) NOT NULL,
    address_line1   VARCHAR(255) NOT NULL,
    address_line2   VARCHAR(255),
    ward            VARCHAR(100),
    district        VARCHAR(100),
    province        VARCHAR(100) NOT NULL,
    postal_code     VARCHAR(20),
    country         VARCHAR(100) NOT NULL DEFAULT 'VN',
    is_default      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_identity_buyer_address_buyer_id ON identity.buyer_address (buyer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_identity_buyer_address_default ON identity.buyer_address (buyer_id, is_default) WHERE deleted_at IS NULL AND is_default = TRUE;

-- Business verification submissions
CREATE TABLE identity.verification (
    id                  BIGSERIAL PRIMARY KEY,
    buyer_id            BIGINT NOT NULL REFERENCES identity.buyer_profile (id) ON DELETE CASCADE,
    code                VARCHAR(26) NOT NULL UNIQUE,  -- ver_ prefix
    business_license    VARCHAR(100) NOT NULL,
    license_image_url   VARCHAR(500),
    tax_certificate_url VARCHAR(500),
    representative_name VARCHAR(255) NOT NULL,
    representative_id   VARCHAR(50) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'resubmit')),
    rejection_reason    TEXT,
    reviewed_by         BIGINT REFERENCES identity."user" (id) ON DELETE SET NULL,
    reviewed_at         TIMESTAMPTZ,
    submitted_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX idx_identity_verification_buyer_id ON identity.verification (buyer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_identity_verification_status ON identity.verification (status) WHERE deleted_at IS NULL;

-- OTP codes for registration/login/password reset
CREATE TABLE identity.otp_code (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES identity."user" (id) ON DELETE CASCADE,
    code_hash       VARCHAR(64) NOT NULL,  -- SHA-256 of the OTP
    purpose         VARCHAR(20) NOT NULL CHECK (purpose IN ('register', 'login', 'password_reset')),
    expires_at      TIMESTAMPTZ NOT NULL,
    consumed_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_identity_otp_user_purpose ON identity.otp_code (user_id, purpose) WHERE consumed_at IS NULL;

-- Password reset tokens
CREATE TABLE identity.password_reset_token (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES identity."user" (id) ON DELETE CASCADE,
    token_hash      VARCHAR(64) NOT NULL,  -- SHA-256 of the token
    expires_at      TIMESTAMPTZ NOT NULL,
    consumed_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_identity_pwd_reset_user ON identity.password_reset_token (user_id) WHERE consumed_at IS NULL;

-- Roles and permissions (for staff/admin)
CREATE TABLE identity.role (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    is_system   BOOLEAN NOT NULL DEFAULT FALSE,  -- system roles cannot be deleted
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE identity.permission (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,  -- e.g., 'buyers.view', 'orders.edit'
    description TEXT,
    category    VARCHAR(50) NOT NULL,  -- buyers, orders, inventory, etc.
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE identity.role_permission (
    role_id       BIGINT NOT NULL REFERENCES identity.role (id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES identity.permission (id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE identity.user_role (
    user_id   BIGINT NOT NULL REFERENCES identity."user" (id) ON DELETE CASCADE,
    role_id   BIGINT NOT NULL REFERENCES identity.role (id) ON DELETE CASCADE,
    assigned_by BIGINT REFERENCES identity."user" (id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

-- Seed default roles and permissions
INSERT INTO identity.role (name, description, is_system) VALUES
    ('super_admin', 'Full system access', TRUE),
    ('admin', 'Administrative access', TRUE),
    ('staff', 'Standard staff access', TRUE),
    ('manager', 'Managerial access', TRUE)
ON CONFLICT (name) DO NOTHING;

-- Seed permissions (aligned with feature inventory)
INSERT INTO identity.permission (name, description, category) VALUES
    -- Buyers
    ('buyers.view', 'View buyer list and details', 'buyers'),
    ('buyers.manage', 'Manage buyer accounts', 'buyers'),
    ('buyers.scope.manage', 'Manage purchase scopes', 'buyers'),
    -- Verification
    ('verification.review', 'Review business verifications', 'verification'),
    ('verification.configure', 'Configure verification rules', 'verification'),
    -- Orders
    ('orders.view', 'View orders', 'orders'),
    ('orders.edit', 'Edit orders', 'orders'),
    ('orders.hold', 'Hold/release orders', 'orders'),
    ('orders.dispatch', 'Dispatch orders', 'orders'),
    -- Shipments
    ('shipments.manage', 'Manage shipments', 'shipments'),
    -- Returns
    ('returns.manage', 'Manage returns and credit notes', 'returns'),
    -- Credit
    ('credit.manage', 'Manage credit accounts', 'credit'),
    -- Inventory
    ('inventory.view', 'View inventory', 'inventory'),
    ('inventory.adjust', 'Adjust stock levels', 'inventory'),
    ('inventory.quarantine', 'Quarantine/release lots', 'inventory'),
    -- Promotions
    ('promotions.view', 'View promotions', 'promotions'),
    ('promotions.manage', 'Manage promotions', 'promotions'),
    -- Partners
    ('partners.manage', 'Manage partners and referrals', 'partners'),
    -- Leads
    ('leads.view', 'View leads', 'leads'),
    ('leads.configure', 'Configure lead routing', 'leads'),
    -- Suppliers
    ('suppliers.view', 'View suppliers', 'suppliers'),
    ('suppliers.manage', 'Manage supplier profiles', 'suppliers'),
    -- CMS
    ('cms.view', 'View CMS content', 'cms'),
    ('cms.edit', 'Edit CMS content', 'cms'),
    ('cms.legal', 'Manage legal documents', 'cms'),
    -- Reports
    ('reports.view', 'View reports', 'reports'),
    ('reports.export', 'Export reports', 'reports'),
    -- Platform
    ('feature_flags.view', 'View feature flags', 'platform'),
    ('feature_flags.manage', 'Manage feature flags', 'platform'),
    ('audit_log.view', 'View audit log', 'platform'),
    ('integration_traffic.view', 'View integration traffic', 'platform'),
    -- AI
    ('system.ai', 'AI system administration', 'ai'),
    -- ERP
    ('erp.view', 'View ERP sync jobs', 'erp'),
    ('erp.manage', 'Manage ERP sync', 'erp'),
    -- System
    ('system.view', 'System administration', 'system')
ON CONFLICT (name) DO NOTHING;

-- Assign all permissions to super_admin
INSERT INTO identity.role_permission (role_id, permission_id)
SELECT r.id, p.id FROM identity.role r, identity.permission p
WHERE r.name = 'super_admin'
ON CONFLICT DO NOTHING;

-- Assign common permissions to admin
INSERT INTO identity.role_permission (role_id, permission_id)
SELECT r.id, p.id FROM identity.role r, identity.permission p
WHERE r.name = 'admin' AND p.category IN ('buyers', 'verification', 'orders', 'shipments', 'returns', 'credit', 'inventory', 'promotions', 'partners', 'leads', 'suppliers', 'cms', 'reports', 'platform', 'erp')
ON CONFLICT DO NOTHING;

-- Assign staff permissions to staff role
INSERT INTO identity.role_permission (role_id, permission_id)
SELECT r.id, p.id FROM identity.role r, identity.permission p
WHERE r.name = 'staff' AND p.category IN ('buyers', 'orders', 'shipments', 'returns', 'inventory', 'promotions', 'suppliers', 'cms', 'reports')
ON CONFLICT DO NOTHING;

-- Assign manager permissions
INSERT INTO identity.role_permission (role_id, permission_id)
SELECT r.id, p.id FROM identity.role r, identity.permission p
WHERE r.name = 'manager' AND p.category IN ('buyers', 'verification', 'orders', 'shipments', 'returns', 'credit', 'inventory', 'promotions', 'partners', 'leads', 'suppliers', 'cms', 'reports')
ON CONFLICT DO NOTHING;