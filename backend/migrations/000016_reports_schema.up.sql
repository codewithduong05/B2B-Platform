-- TASK-014: M5 reporting slice 1 — export centre metadata.
--
-- Reports are read-only projections over existing transactional schemas;
-- no reporting tables are materialised. This migration stores export job
-- requests only. Generation is asynchronous by contract (08: "Async export
-- with a download link"); the repository has no worker or file storage yet,
-- so jobs remain 'pending' until a worker claims them.

CREATE SCHEMA IF NOT EXISTS reports;

CREATE TABLE reports.export_job (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    report_type VARCHAR(64) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    format VARCHAR(16) NOT NULL DEFAULT 'csv' CHECK (format IN ('csv')),
    download_url VARCHAR(500),
    parameters TEXT,
    created_by BIGINT REFERENCES identity.user (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_reports_export_job_status ON reports.export_job (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_reports_export_job_type ON reports.export_job (report_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_reports_export_job_created ON reports.export_job (created_at DESC) WHERE deleted_at IS NULL;
