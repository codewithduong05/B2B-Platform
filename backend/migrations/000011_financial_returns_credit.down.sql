DROP TRIGGER IF EXISTS update_credit_note_updated_at ON commerce.credit_note;
DROP TRIGGER IF EXISTS update_return_updated_at ON commerce.return_request;

DROP TABLE IF EXISTS commerce.credit_note;
DROP TABLE IF EXISTS commerce.return_line;
DROP TABLE IF EXISTS commerce.return_request;

ALTER TABLE commerce.invoice DROP COLUMN IF EXISTS replaces_code;
