CREATE SCHEMA IF NOT EXISTS cms;

CREATE TABLE IF NOT EXISTS cms.contact_enquiry (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    company VARCHAR(200),
    subject VARCHAR(200) NOT NULL,
    message TEXT NOT NULL,
    source VARCHAR(50) NOT NULL DEFAULT 'contact_form',
    status VARCHAR(50) NOT NULL DEFAULT 'new',
    lead_id BIGINT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    routed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_contact_enquiry_status ON cms.contact_enquiry(status);
CREATE INDEX IF NOT EXISTS idx_contact_enquiry_source ON cms.contact_enquiry(source);
CREATE INDEX IF NOT EXISTS idx_contact_enquiry_created_at ON cms.contact_enquiry(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_contact_enquiry_email ON cms.contact_enquiry(email);
