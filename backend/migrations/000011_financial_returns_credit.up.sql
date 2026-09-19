-- TASK-008: financial lifecycle — returns, credit notes, invoice reissue chain.
--
-- Returns are financial only: completing a return creates a credit note but
-- never touches inventory (deliberate scope decision, documented in code).

-- Reissue chain reference on invoices.
ALTER TABLE commerce.invoice ADD COLUMN replaces_code VARCHAR(26);

-- Return requests: buyer- or staff-initiated, fulfilled financially.
CREATE TABLE commerce.return_request (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    order_id BIGINT NOT NULL REFERENCES commerce."order" (id) ON DELETE RESTRICT,
    buyer_id BIGINT NOT NULL REFERENCES identity.buyer_profile (id) ON DELETE RESTRICT,
    reason TEXT NOT NULL CHECK (char_length(reason) > 0),
    status VARCHAR(16) NOT NULL DEFAULT 'requested' CHECK (status IN ('requested', 'approved', 'rejected', 'completed')),
    requested_by BIGINT,
    decided_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_commerce_return_order ON commerce.return_request (order_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_commerce_return_buyer ON commerce.return_request (buyer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_commerce_return_code ON commerce.return_request (code) WHERE deleted_at IS NULL;

CREATE TABLE commerce.return_line (
    id BIGSERIAL PRIMARY KEY,
    return_id BIGINT NOT NULL REFERENCES commerce.return_request (id) ON DELETE CASCADE,
    order_line_id BIGINT NOT NULL REFERENCES commerce.order_line (id) ON DELETE RESTRICT,
    quantity INT NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (return_id, order_line_id)
);

CREATE INDEX idx_commerce_return_line_return ON commerce.return_line (return_id);
CREATE INDEX idx_commerce_return_line_order_line ON commerce.return_line (order_line_id);

-- Credit notes: one row per invoice touched by an approval, grouped by
-- batch_code (a single approval may spread across several invoices).
-- Applied oldest-first on approval; void reverses via capped restore.
-- Rows are never deleted (R10).
CREATE TABLE commerce.credit_note (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(26) NOT NULL UNIQUE,
    batch_code VARCHAR(26) NOT NULL,
    order_id BIGINT NOT NULL REFERENCES commerce."order" (id) ON DELETE RESTRICT,
    invoice_id BIGINT REFERENCES commerce.invoice (id) ON DELETE SET NULL,
    return_id BIGINT REFERENCES commerce.return_request (id) ON DELETE SET NULL,
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    reason TEXT NOT NULL CHECK (char_length(reason) > 0),
    status VARCHAR(16) NOT NULL DEFAULT 'applied' CHECK (status IN ('applied', 'void')),
    actor BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_commerce_credit_note_order ON commerce.credit_note (order_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_commerce_credit_note_code ON commerce.credit_note (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_commerce_credit_note_batch ON commerce.credit_note (batch_code) WHERE deleted_at IS NULL;

CREATE TRIGGER update_return_updated_at BEFORE UPDATE ON commerce.return_request FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();
CREATE TRIGGER update_credit_note_updated_at BEFORE UPDATE ON commerce.credit_note FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();
