-- TASK-016: M5 export worker — file storage metadata.
--
-- Extends reports.export_job with file storage fields. The worker generates
-- files and stores them; this tracks the storage location and metadata.

ALTER TABLE reports.export_job
    ADD COLUMN IF NOT EXISTS file_path VARCHAR(500),
    ADD COLUMN IF NOT EXISTS file_size BIGINT,
    ADD COLUMN IF NOT EXISTS file_checksum VARCHAR(64),
    ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS error_message TEXT;

CREATE INDEX idx_reports_export_job_file ON reports.export_job (file_path) WHERE file_path IS NOT NULL AND deleted_at IS NULL;
