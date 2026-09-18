-- Rollback identity module schema

-- Drop triggers
DROP TRIGGER IF EXISTS update_permission_updated_at ON identity.permission;
DROP TRIGGER IF EXISTS update_role_updated_at ON identity.role;
DROP TRIGGER IF EXISTS update_credit_account_updated_at ON identity.credit_account;
DROP TRIGGER IF EXISTS update_verification_application_updated_at ON identity.verification_application;
DROP TRIGGER IF EXISTS update_address_updated_at ON identity.address;
DROP TRIGGER IF EXISTS update_supplier_profile_updated_at ON identity.supplier_profile;
DROP TRIGGER IF EXISTS update_buyer_profile_updated_at ON identity.buyer_profile;
DROP TRIGGER IF EXISTS update_user_updated_at ON identity.user;

DROP FUNCTION IF EXISTS identity.update_updated_at_column();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS identity.session;
DROP TABLE IF EXISTS identity.otp_code;
DROP TABLE IF EXISTS identity.password_reset_token;
DROP TABLE IF EXISTS identity.refresh_token;
DROP TABLE IF EXISTS identity.role_permission;
DROP TABLE IF EXISTS identity.user_role;
DROP TABLE IF EXISTS identity.permission;
DROP TABLE IF EXISTS identity.role;
DROP TABLE IF EXISTS identity.credit_account;
DROP TABLE IF EXISTS identity.price_list_assignment;
DROP TABLE IF EXISTS identity.purchase_scope;
DROP TABLE IF EXISTS identity.verification_document;
DROP TABLE IF EXISTS identity.verification_application;
DROP TABLE IF EXISTS identity.address;
DROP TABLE IF EXISTS identity.supplier_profile;
DROP TABLE IF EXISTS identity.buyer_profile;
DROP TABLE IF EXISTS identity.user;

-- Drop enum types
DROP TYPE IF EXISTS identity.verification_decision;
DROP TYPE IF EXISTS identity.verification_status;
DROP TYPE IF EXISTS identity.user_type;