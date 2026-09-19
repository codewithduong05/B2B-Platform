-- Restore the original unconditional uniqueness (dev-only rollback path;
-- fails if a visible row and a soft-deleted row share (cart_id, product_id)).

DROP INDEX IF EXISTS commerce.uq_commerce_cart_line_cart_product_active;

ALTER TABLE commerce.cart_line
    ADD CONSTRAINT cart_line_cart_id_product_id_key UNIQUE (cart_id, product_id);
