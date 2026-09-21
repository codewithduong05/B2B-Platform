-- Identity module queries
-- User management

-- name: CreateUser :one
INSERT INTO identity.user (
    code, email, password_hash, user_type
) VALUES (
    $1, $2, $3, $4
) RETURNING id, code, email, user_type, is_active, is_verified, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, code, email, password_hash, user_type, is_active, is_verified,
       last_login_at, failed_login_attempts, locked_until,
       created_at, updated_at, deleted_at
FROM identity.user
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByCode :one
SELECT id, code, email, password_hash, user_type, is_active, is_verified,
       last_login_at, failed_login_attempts, locked_until,
       created_at, updated_at, deleted_at
FROM identity.user
WHERE code = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT id, code, email, password_hash, user_type, is_active, is_verified,
       last_login_at, failed_login_attempts, locked_until,
       created_at, updated_at, deleted_at
FROM identity.user
WHERE email = $1 AND deleted_at IS NULL;

-- name: UpdateUser :one
UPDATE identity.user
SET email = COALESCE($2, email),
    user_type = COALESCE($3, user_type),
    is_active = COALESCE($4, is_active),
    is_verified = COALESCE($5, is_verified),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, code, email, user_type, is_active, is_verified, created_at, updated_at;

-- name: UpdateUserPassword :exec
UPDATE identity.user
SET password_hash = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateUserLoginAttempts :exec
UPDATE identity.user
SET failed_login_attempts = $2,
    locked_until = $3,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateUserLastLogin :exec
UPDATE identity.user
SET last_login_at = NOW(),
    failed_login_attempts = 0,
    locked_until = NULL,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteUser :exec
UPDATE identity.user
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: ListUsers :many
SELECT id, code, email, user_type, is_active, is_verified,
       last_login_at, created_at, updated_at
FROM identity.user
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM identity.user WHERE deleted_at IS NULL;

-- Buyer Profiles

-- name: CreateBuyerProfile :one
INSERT INTO identity.buyer_profile (
    code, user_id, business_name, trading_name, tax_id,
    registration_number, phone, website, industry,
    employee_count, annual_revenue_minor, currency
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING id, code, user_id, business_name, created_at, updated_at;

-- name: GetBuyerProfileByUserID :one
SELECT id, code, user_id, business_name, trading_name, tax_id,
       registration_number, phone, website, industry,
       employee_count, annual_revenue_minor, currency,
       credit_limit_minor, credit_terms_days, is_on_credit_hold,
       credit_hold_reason, created_at, updated_at, deleted_at
FROM identity.buyer_profile
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: GetBuyerProfileByCode :one
SELECT id, code, user_id, business_name, trading_name, tax_id,
       registration_number, phone, website, industry,
       employee_count, annual_revenue_minor, currency,
       credit_limit_minor, credit_terms_days, is_on_credit_hold,
       credit_hold_reason, created_at, updated_at, deleted_at
FROM identity.buyer_profile
WHERE code = $1 AND deleted_at IS NULL;

-- name: UpdateBuyerProfile :one
UPDATE identity.buyer_profile
SET business_name = COALESCE($2, business_name),
    trading_name = COALESCE($3, trading_name),
    tax_id = COALESCE($4, tax_id),
    registration_number = COALESCE($5, registration_number),
    phone = COALESCE($6, phone),
    website = COALESCE($7, website),
    industry = COALESCE($8, industry),
    employee_count = COALESCE($9, employee_count),
    annual_revenue_minor = COALESCE($10, annual_revenue_minor),
    currency = COALESCE($11, currency),
    credit_limit_minor = COALESCE($12, credit_limit_minor),
    credit_terms_days = COALESCE($13, credit_terms_days),
    is_on_credit_hold = COALESCE($14, is_on_credit_hold),
    credit_hold_reason = COALESCE($15, credit_hold_reason),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, code, user_id, business_name, created_at, updated_at;

-- name: SoftDeleteBuyerProfile :exec
UPDATE identity.buyer_profile
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: UpdateBuyerProfileByUserID :one
UPDATE identity.buyer_profile
SET business_name = COALESCE($2, business_name),
    trading_name = COALESCE($3, trading_name),
    tax_id = COALESCE($4, tax_id),
    registration_number = COALESCE($5, registration_number),
    phone = COALESCE($6, phone),
    website = COALESCE($7, website),
    industry = COALESCE($8, industry),
    employee_count = COALESCE($9, employee_count),
    annual_revenue_minor = COALESCE($10, annual_revenue_minor),
    currency = COALESCE($11, currency),
    credit_limit_minor = COALESCE($12, credit_limit_minor),
    credit_terms_days = COALESCE($13, credit_terms_days),
    is_on_credit_hold = COALESCE($14, is_on_credit_hold),
    credit_hold_reason = COALESCE($15, credit_hold_reason),
    updated_at = NOW()
WHERE user_id = $1 AND deleted_at IS NULL
RETURNING id, code, user_id, business_name, trading_name, tax_id,
       registration_number, phone, website, industry,
       employee_count, annual_revenue_minor, currency,
       credit_limit_minor, credit_terms_days, is_on_credit_hold,
       credit_hold_reason, created_at, updated_at;

-- Supplier Profiles

-- name: CreateSupplierProfile :one
INSERT INTO identity.supplier_profile (
    code, user_id, company_name, trading_name, tax_id,
    registration_number, phone, email, website,
    address_line1, address_line2, city, state_province,
    postal_code, country, contact_person, contact_phone, contact_email
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
) RETURNING id, code, user_id, company_name, created_at, updated_at;

-- name: GetSupplierProfileByUserID :one
SELECT id, code, user_id, company_name, trading_name, tax_id,
       registration_number, phone, email, website,
       address_line1, address_line2, city, state_province,
       postal_code, country, contact_person, contact_phone, contact_email,
       is_approved, approved_at, approved_by, created_at, updated_at, deleted_at
FROM identity.supplier_profile
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: GetSupplierProfileByCode :one
SELECT id, code, user_id, company_name, trading_name, tax_id,
       registration_number, phone, email, website,
       address_line1, address_line2, city, state_province,
       postal_code, country, contact_person, contact_phone, contact_email,
       is_approved, approved_at, approved_by, created_at, updated_at, deleted_at
FROM identity.supplier_profile
WHERE code = $1 AND deleted_at IS NULL;

-- Addresses

-- name: CreateAddress :one
INSERT INTO identity.address (
    code, owner_user_id, owner_type, label,
    recipient_name, company_name, line1, line2,
    city, state_province, postal_code, country,
    phone, is_default, handling_class, delivery_instructions
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
) RETURNING id, code, owner_user_id, owner_type, label, created_at, updated_at;

-- name: GetAddressesByOwner :many
SELECT id, code, owner_user_id, owner_type, label,
       recipient_name, company_name, line1, line2,
       city, state_province, postal_code, country,
       phone, is_default, handling_class, delivery_instructions,
       created_at, updated_at
FROM identity.address
WHERE owner_user_id = $1 AND owner_type = $2 AND deleted_at IS NULL
ORDER BY is_default DESC, created_at ASC;

-- name: GetAddressByCode :one
SELECT id, code, owner_user_id, owner_type, label,
       recipient_name, company_name, line1, line2,
       city, state_province, postal_code, country,
       phone, is_default, handling_class, delivery_instructions,
       created_at, updated_at, deleted_at
FROM identity.address
WHERE code = $1 AND deleted_at IS NULL;

-- name: GetAddressByID :one
SELECT id, code, owner_user_id, owner_type, label,
       recipient_name, company_name, line1, line2,
       city, state_province, postal_code, country,
       phone, is_default, handling_class, delivery_instructions,
       created_at, updated_at, deleted_at
FROM identity.address
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateAddress :one
UPDATE identity.address
SET label = COALESCE($2, label),
    recipient_name = COALESCE($3, recipient_name),
    company_name = COALESCE($4, company_name),
    line1 = COALESCE($5, line1),
    line2 = COALESCE($6, line2),
    city = COALESCE($7, city),
    state_province = COALESCE($8, state_province),
    postal_code = COALESCE($9, postal_code),
    country = COALESCE($10, country),
    phone = COALESCE($11, phone),
    is_default = COALESCE($12, is_default),
    handling_class = COALESCE($13, handling_class),
    delivery_instructions = COALESCE($14, delivery_instructions),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, code, owner_user_id, owner_type, label, created_at, updated_at;

-- name: SetDefaultAddress :exec
UPDATE identity.address
SET is_default = CASE WHEN id = $1 THEN TRUE ELSE FALSE END,
    updated_at = NOW()
WHERE owner_user_id = $2 AND owner_type = $3 AND deleted_at IS NULL;

-- name: SoftDeleteAddress :exec
UPDATE identity.address
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- Verification Applications

-- name: CreateVerificationApplication :one
INSERT INTO identity.verification_application (
    code, buyer_profile_id, licence_number, licence_expiry,
    trading_name, business_address_line1, business_address_line2,
    business_city, business_state_province, business_postal_code, business_country
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING id, code, buyer_profile_id, status, created_at, updated_at;

-- name: GetVerificationApplicationByID :one
SELECT id, code, buyer_profile_id, status, submitted_at, decided_at,
       decided_by, decision_reason, rejection_reason,
       licence_number, licence_expiry, trading_name,
       business_address_line1, business_address_line2,
       business_city, business_state_province, business_postal_code, business_country,
       created_at, updated_at, deleted_at
FROM identity.verification_application
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetVerificationApplicationByBuyer :one
SELECT id, code, buyer_profile_id, status, submitted_at, decided_at,
       decided_by, decision_reason, rejection_reason,
       created_at, updated_at, deleted_at
FROM identity.verification_application
WHERE buyer_profile_id = $1 AND deleted_at IS NULL
ORDER BY submitted_at DESC
LIMIT 1;

-- name: UpdateVerificationApplicationStatus :one
UPDATE identity.verification_application
SET status = $2,
    decided_at = CASE WHEN $2 IN ('approved', 'rejected') THEN NOW() ELSE decided_at END,
    decided_by = CASE WHEN $2 IN ('approved', 'rejected') THEN $3 ELSE decided_by END,
    decision_reason = CASE WHEN $2 = 'approved' THEN $4 ELSE decision_reason END,
    rejection_reason = CASE WHEN $2 = 'rejected' THEN $4 ELSE rejection_reason END,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, code, buyer_profile_id, status, decided_at, decided_by;

-- name: ListVerificationApplications :many
SELECT id, code, buyer_profile_id, status, submitted_at, decided_at,
       decided_by, decision_reason, rejection_reason,
       created_at, updated_at
FROM identity.verification_application
WHERE deleted_at IS NULL
  AND ($1::identity.verification_status IS NULL OR status = $1)
ORDER BY submitted_at DESC
LIMIT $2 OFFSET $3;

-- name: CountVerificationApplications :one
SELECT COUNT(*) FROM identity.verification_application
WHERE deleted_at IS NULL
  AND ($1::identity.verification_status IS NULL OR status = $1);

-- Verification Documents

-- name: CreateVerificationDocument :one
INSERT INTO identity.verification_document (
    code, verification_application_id, document_type,
    file_name, file_path, file_size, mime_type, uploaded_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING id, code, verification_application_id, created_at;

-- name: GetVerificationDocuments :many
SELECT id, code, verification_application_id, document_type,
       file_name, file_path, file_size, mime_type,
       uploaded_at, uploaded_by, created_at
FROM identity.verification_document
WHERE verification_application_id = $1 AND deleted_at IS NULL
ORDER BY uploaded_at DESC;

-- Purchase Scopes

-- name: CreatePurchaseScope :one
INSERT INTO identity.purchase_scope (
    code, buyer_profile_id, handling_class, granted_by, expires_at
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING id, code, buyer_profile_id, handling_class, granted_at, expires_at;

-- name: GetPurchaseScopesByBuyer :many
SELECT id, code, buyer_profile_id, handling_class,
       granted_at, granted_by, expires_at, created_at
FROM identity.purchase_scope
WHERE buyer_profile_id = $1 AND deleted_at IS NULL
ORDER BY granted_at DESC;

-- name: DeletePurchaseScope :exec
UPDATE identity.purchase_scope
SET deleted_at = NOW()
WHERE id = $1;

-- Price List Assignments

-- name: CreatePriceListAssignment :one
INSERT INTO identity.price_list_assignment (
    code, buyer_profile_id, price_list_id, assigned_by,
    effective_from, effective_to
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING id, code, buyer_profile_id, price_list_id, effective_from, effective_to;

-- name: GetPriceListAssignmentsByBuyer :many
SELECT id, code, buyer_profile_id, price_list_id,
       effective_from, effective_to, assigned_at
FROM identity.price_list_assignment
WHERE buyer_profile_id = $1 AND deleted_at IS NULL
ORDER BY effective_from DESC;

-- Credit Accounts

-- name: CreateCreditAccount :one
INSERT INTO identity.credit_account (
    code, buyer_profile_id, limit_minor, currency, terms_days
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING id, code, buyer_profile_id, limit_minor, currency, terms_days, created_at;

-- name: GetCreditAccountByBuyer :one
SELECT id, code, buyer_profile_id, limit_minor, currency, terms_days,
       current_exposure_minor, available_minor, is_on_hold, hold_reason, hold_since,
       created_at, updated_at, deleted_at
FROM identity.credit_account
WHERE buyer_profile_id = $1 AND deleted_at IS NULL;

-- name: UpdateCreditAccountExposure :exec
UPDATE identity.credit_account
SET current_exposure_minor = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateCreditAccountHold :exec
UPDATE identity.credit_account
SET is_on_hold = $2,
    hold_reason = $3,
    hold_since = CASE WHEN $2 THEN NOW() ELSE NULL END,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- Roles and Permissions

-- name: CreateRole :one
INSERT INTO identity.role (code, name, description, is_system)
VALUES ($1, $2, $3, $4)
RETURNING id, code, name, description, is_system, created_at;

-- name: GetRoleByCode :one
SELECT id, code, name, description, is_system, created_at, updated_at
FROM identity.role
WHERE code = $1 AND deleted_at IS NULL;

-- name: ListRoles :many
SELECT id, code, name, description, is_system, created_at
FROM identity.role
WHERE deleted_at IS NULL
ORDER BY code;

-- name: CreatePermission :one
INSERT INTO identity.permission (code, name, description, resource, action)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, code, name, description, resource, action, created_at;

-- name: GetPermissionByCode :one
SELECT id, code, name, description, resource, action, created_at
FROM identity.permission
WHERE code = $1 AND deleted_at IS NULL;

-- name: ListPermissions :many
SELECT id, code, name, description, resource, action, created_at
FROM identity.permission
WHERE deleted_at IS NULL
ORDER BY resource, action;

-- name: AssignRoleToUser :exec
INSERT INTO identity.user_role (user_id, role_id, assigned_by)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, role_id) DO NOTHING;

-- name: RemoveRoleFromUser :exec
DELETE FROM identity.user_role
WHERE user_id = $1 AND role_id = $2;

-- name: GetUserRoles :many
SELECT r.id, r.code, r.name, r.description, r.is_system
FROM identity.role r
JOIN identity.user_role ur ON r.id = ur.role_id
WHERE ur.user_id = $1 AND r.deleted_at IS NULL;

-- name: GetUserPermissions :many
SELECT p.id, p.code, p.name, p.resource, p.action
FROM identity.permission p
JOIN identity.role_permission rp ON p.id = rp.permission_id
JOIN identity.user_role ur ON rp.role_id = ur.role_id
WHERE ur.user_id = $1 AND p.deleted_at IS NULL;

-- name: AssignPermissionToRole :exec
INSERT INTO identity.role_permission (role_id, permission_id)
VALUES ($1, $2)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Refresh Tokens

-- name: CreateRefreshToken :one
INSERT INTO identity.refresh_token (
    code, user_id, token_hash, family_id, expires_at
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING id, code, user_id, family_id, expires_at, created_at;

-- name: GetRefreshTokenByHash :one
SELECT id, code, user_id, token_hash, family_id, expires_at,
       revoked_at, revoked_reason, created_at, replaced_by_id
FROM identity.refresh_token
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW();

-- name: RevokeRefreshToken :exec
UPDATE identity.refresh_token
SET revoked_at = NOW(), revoked_reason = $2
WHERE id = $1;

-- name: RevokeRefreshTokenFamily :exec
UPDATE identity.refresh_token
SET revoked_at = NOW(), revoked_reason = $2
WHERE family_id = $1 AND revoked_at IS NULL;

-- name: ReplaceRefreshToken :one
WITH old AS (
    UPDATE identity.refresh_token
    SET revoked_at = NOW(), revoked_reason = 'rotated', replaced_by_id = $2
    WHERE id = $1 AND revoked_at IS NULL
    RETURNING family_id
)
INSERT INTO identity.refresh_token (code, user_id, token_hash, family_id, expires_at)
SELECT $3, rt.user_id, $4, rt.family_id, $5
FROM identity.refresh_token rt
JOIN old ON rt.family_id = old.family_id
WHERE rt.id = $1
RETURNING id, code, user_id, family_id, expires_at, created_at;

-- name: CleanupExpiredRefreshTokens :exec
DELETE FROM identity.refresh_token
WHERE expires_at < NOW() - INTERVAL '30 days';

-- Password Reset Tokens

-- name: CreatePasswordResetToken :one
INSERT INTO identity.password_reset_token (code, user_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, code, user_id, expires_at, created_at;

-- name: GetPasswordResetToken :one
SELECT id, code, user_id, token_hash, expires_at, used_at, created_at
FROM identity.password_reset_token
WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW();

-- name: UsePasswordResetToken :exec
UPDATE identity.password_reset_token
SET used_at = NOW()
WHERE id = $1;

-- name: CleanupExpiredPasswordResetTokens :exec
DELETE FROM identity.password_reset_token
WHERE expires_at < NOW() OR used_at IS NOT NULL;

-- OTP Codes

-- name: CreateOTPCode :one
INSERT INTO identity.otp_code (code, user_id, otp_hash, purpose, expires_at, max_attempts)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, code, user_id, purpose, expires_at, created_at;

-- name: GetOTPCode :one
SELECT id, code, user_id, otp_hash, purpose, expires_at, used_at, attempts, max_attempts, created_at
FROM identity.otp_code
WHERE user_id = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1;

-- name: IncrementOTPAttempts :exec
UPDATE identity.otp_code
SET attempts = attempts + 1
WHERE id = $1;

-- name: UseOTPCode :exec
UPDATE identity.otp_code
SET used_at = NOW()
WHERE id = $1;

-- name: CleanupExpiredOTPCodes :exec
DELETE FROM identity.otp_code
WHERE expires_at < NOW() OR used_at IS NOT NULL;

-- Sessions

-- name: CreateSession :one
INSERT INTO identity.session (code, user_id, refresh_token_id, ip_address, user_agent, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, code, user_id, expires_at, created_at;

-- name: GetSessionByCode :one
SELECT id, code, user_id, refresh_token_id, ip_address, user_agent,
       expires_at, revoked_at, created_at
FROM identity.session
WHERE code = $1 AND revoked_at IS NULL AND expires_at > NOW();

-- name: RevokeSession :exec
UPDATE identity.session
SET revoked_at = NOW()
WHERE id = $1;

-- name: RevokeUserSessions :exec
UPDATE identity.session
SET revoked_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: CleanupExpiredSessions :exec
DELETE FROM identity.session
WHERE expires_at < NOW() - INTERVAL '7 days';