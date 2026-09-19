-- TASK-014: M5 Reporting Slice 1 — Reports & Export Centre

CREATE SCHEMA IF NOT EXISTS reports;

CREATE TABLE reports.export_job (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    report_type VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'completed',
    format VARCHAR(16) NOT NULL DEFAULT 'csv',
    download_url VARCHAR(500),
    parameters TEXT,
    created_by BIGINT REFERENCES identity.user (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_reports_export_job_status ON reports.export_job (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_reports_export_job_type ON reports.export_job (report_type) WHERE deleted_at IS NULL;
