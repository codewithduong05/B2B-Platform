CREATE TABLE IF NOT EXISTS cms.legal_document (
    doc_type TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    current_version INT,
    effective_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by TEXT,
    draft_title TEXT,
    draft_body TEXT,
    draft_body_format TEXT NOT NULL DEFAULT 'markdown' CHECK (draft_body_format IN ('markdown', 'html', 'plaintext')),
    draft_effective_at TIMESTAMPTZ,
    draft_updated_at TIMESTAMPTZ,
    draft_updated_by TEXT
);

CREATE TABLE IF NOT EXISTS cms.legal_document_version (
    id BIGSERIAL PRIMARY KEY,
    doc_type TEXT NOT NULL REFERENCES cms.legal_document(doc_type),
    version INT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    body_format TEXT NOT NULL DEFAULT 'markdown' CHECK (body_format IN ('markdown', 'html', 'plaintext')),
    status TEXT NOT NULL DEFAULT 'published' CHECK (status IN ('published', 'superseded')),
    effective_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT,
    UNIQUE (doc_type, version)
);

CREATE INDEX IF NOT EXISTS idx_legal_document_version_doc_type ON cms.legal_document_version (doc_type);
CREATE INDEX IF NOT EXISTS idx_legal_document_version_status ON cms.legal_document_version (status);
CREATE INDEX IF NOT EXISTS idx_legal_document_version_version ON cms.legal_document_version (doc_type, version DESC);
