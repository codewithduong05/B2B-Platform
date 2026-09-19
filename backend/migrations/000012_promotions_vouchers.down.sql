DROP TRIGGER IF EXISTS update_promotion_updated_at ON promotions.promotion;

ALTER TABLE commerce.cart DROP COLUMN IF EXISTS voucher_code;

DROP TABLE IF EXISTS promotions.voucher_redemption;
DROP TABLE IF EXISTS promotions.promotion;

DROP SCHEMA IF EXISTS promotions;
