-- Rollback M1 Identity: Identity schema and tables
-- Safe rollback: only drops objects that this migration created.
-- Uses IF EXISTS to be safe when 000002 tables exist.

DROP TABLE IF EXISTS identity.user_role;
DROP TABLE IF EXISTS identity.role_permission;
DROP TABLE IF EXISTS identity.permission;
DROP TABLE IF EXISTS identity.role;
DROP TABLE IF EXISTS identity.password_reset_token;
DROP TABLE IF EXISTS identity.otp_code;
DROP TABLE IF EXISTS identity.verification;
DROP TABLE IF EXISTS identity.buyer_address;
DROP TABLE IF EXISTS identity.buyer_profile;
DROP TABLE IF EXISTS identity."user";
-- Note: Do not DROP SCHEMA here as 000002 owns the identity schema.
