-- TASK-009: promotions slice 1 — vouchers end-to-end.
--
-- One promotion carries exactly one redeemable code (1:1). Budget counts
-- successful checkouts (redeemed_count bumped once per checkout, guarded
-- atomically); per-order rows share the checkout and are unique per
-- (buyer, promotion, order) so a retried order insert can never duplicate.
-- Once-per-buyer reuse is enforced by ValidateVoucher pre-checks (a buyer
-- owns a single cart, so concurrent double-use cannot interleave).

CREATE SCHEMA IF NOT EXISTS promotions;

-- Kinds: percent (1-100) | fixed (minor units).
-- Status: draft -> published -> archived (forward only, service-enforced).
CREATE TABLE promotions.promotion (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    kind VARCHAR(16) NOT NULL CHECK (kind IN ('percent', 'fixed')),
    value_minor BIGINT NOT NULL CHECK (value_minor > 0),
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    status VARCHAR(16) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
    valid_from TIMESTAMPTZ,
    valid_to TIMESTAMPTZ,
    max_redemptions INT CHECK (max_redemptions IS NULL OR max_redemptions > 0),
    redeemed_count INT NOT NULL DEFAULT 0 CHECK (redeemed_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CHECK (valid_to IS NULL OR valid_from IS NULL OR valid_to >= valid_from)
);

CREATE INDEX idx_promotions_promotion_code ON promotions.promotion (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_promotions_promotion_status ON promotions.promotion (status) WHERE deleted_at IS NULL;

CREATE TABLE promotions.voucher_redemption (
    id BIGSERIAL PRIMARY KEY,
    promotion_id BIGINT NOT NULL REFERENCES promotions.promotion (id) ON DELETE RESTRICT,
    buyer_id BIGINT NOT NULL REFERENCES identity.buyer_profile (id) ON DELETE RESTRICT,
    order_id BIGINT NOT NULL REFERENCES commerce."order" (id) ON DELETE RESTRICT,
    amount_minor BIGINT NOT NULL CHECK (amount_minor >= 0),
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (buyer_id, promotion_id, order_id)
);

CREATE INDEX idx_promotions_redemption_promo ON promotions.voucher_redemption (promotion_id);
CREATE INDEX idx_promotions_redemption_buyer ON promotions.voucher_redemption (buyer_id);
CREATE INDEX idx_promotions_redemption_order ON promotions.voucher_redemption (order_id);

-- Cart-level applied voucher (opaque code, resolved at quote/checkout time).
ALTER TABLE commerce.cart ADD COLUMN voucher_code VARCHAR(64);

CREATE TRIGGER update_promotion_updated_at BEFORE UPDATE ON promotions.promotion FOR EACH ROW EXECUTE FUNCTION commerce.update_updated_at_column();
