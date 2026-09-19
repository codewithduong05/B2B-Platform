-- TASK-011: M4 suppliers slice 1 — operational profiles, contracts.
--
-- suppliers.supplier_profile is the operational record (portal login,
-- approval, contracts). identity.supplier_profile (M1 onboarding) is left
-- untouched. catalog.supplier.supplier_id references suppliers profiles
-- per the catalog migration comment (wired when SKU mapping lands).

CREATE SCHEMA IF NOT EXISTS suppliers;

-- Status: applied -> approved. No other transitions in this slice.
CREATE TABLE suppliers.supplier_profile (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    company_name VARCHAR(255) NOT NULL,
    contact_name VARCHAR(255),
    contact_email VARCHAR(255),
    contact_phone VARCHAR(64),
    status VARCHAR(16) NOT NULL DEFAULT 'applied' CHECK (status IN ('applied', 'approved', 'rejected', 'suspended')),
    user_id BIGINT REFERENCES identity.user (id) ON DELETE SET NULL,
    approved_by BIGINT,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_suppliers_profile_code ON suppliers.supplier_profile (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_suppliers_profile_user ON suppliers.supplier_profile (user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_suppliers_profile_status ON suppliers.supplier_profile (status) WHERE deleted_at IS NULL;

-- Contracts are immutable rows; PUT mints a new version and deactivates the
-- previous current one, preserving full history.
CREATE TABLE suppliers.supplier_contract (
    id BIGSERIAL PRIMARY KEY,
    supplier_id BIGINT NOT NULL REFERENCES suppliers.supplier_profile (id) ON DELETE CASCADE,
    version INT NOT NULL CHECK (version > 0),
    terms TEXT NOT NULL CHECK (char_length(terms) > 0),
    valid_from TIMESTAMPTZ,
    valid_to TIMESTAMPTZ,
    is_current BOOLEAN NOT NULL DEFAULT TRUE,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (supplier_id, version),
    CHECK (valid_to IS NULL OR valid_from IS NULL OR valid_to >= valid_from)
);

CREATE INDEX idx_suppliers_contract_supplier ON suppliers.supplier_contract (supplier_id);
CREATE INDEX idx_suppliers_contract_current ON suppliers.supplier_contract (supplier_id) WHERE is_current;
