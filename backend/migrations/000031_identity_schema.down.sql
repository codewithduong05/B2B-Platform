-- Rollback M1 Identity: Identity schema and tables

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
DROP SCHEMA IF EXISTS identity CASCADE;