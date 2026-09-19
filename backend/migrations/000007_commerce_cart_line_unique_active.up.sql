-- Fix soft-delete ghost lines on commerce.cart_line.
--
-- The original UNIQUE (cart_id, product_id) constraint also matched
-- soft-deleted rows. Re-adding a product after DELETE therefore hit the
-- conflict path, accumulated quantity onto a row that kept deleted_at set,
-- and returned an invisible line. A partial unique index enforces one
-- ACTIVE line per cart/product while letting re-adds insert a fresh row.

ALTER TABLE commerce.cart_line DROP CONSTRAINT cart_line_cart_id_product_id_key;

CREATE UNIQUE INDEX uq_commerce_cart_line_cart_product_active
    ON commerce.cart_line (cart_id, product_id)
    WHERE deleted_at IS NULL;
