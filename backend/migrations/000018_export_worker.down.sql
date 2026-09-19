ALTER TABLE reports.export_job
    DROP COLUMN IF EXISTS file_path,
    DROP COLUMN IF EXISTS file_size,
    DROP COLUMN IF EXISTS file_checksum,
    DROP COLUMN IF EXISTS started_at,
    DROP COLUMN IF EXISTS completed_at,
    DROP COLUMN IF EXISTS error_message;

DROP INDEX IF EXISTS idx_reports_export_job_file;
