package repository

import (
	"context"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/database/queries/identity"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserRepository struct {
	db *database.DB
	q  *identity.Queries
}

func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{
		db: db,
		q:  identity.New(db.Pool),
	}
}

func (r *UserRepository) CreateUser(ctx context.Context, params identity.CreateUserParams) (identity.CreateUserRow, error) {
	return r.q.CreateUser(ctx, params)
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int64) (identity.IdentityUser, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *UserRepository) GetUserByCode(ctx context.Context, code string) (identity.IdentityUser, error) {
	return r.q.GetUserByCode(ctx, code)
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (identity.IdentityUser, error) {
	return r.q.GetUserByEmail(ctx, email)
}

func (r *UserRepository) UpdateUser(ctx context.Context, id int64, email string, userType identity.IdentityUserType, isActive, isVerified bool) (identity.UpdateUserRow, error) {
	return r.q.UpdateUser(ctx, identity.UpdateUserParams{
		ID:         id,
		Email:      email,
		UserType:   userType,
		IsActive:   isActive,
		IsVerified: isVerified,
	})
}

func (r *UserRepository) UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error {
	return r.q.UpdateUserPassword(ctx, identity.UpdateUserPasswordParams{
		ID:           id,
		PasswordHash: passwordHash,
	})
}

func (r *UserRepository) UpdateUserLoginAttempts(ctx context.Context, id int64, attempts int32, lockedUntil *time.Time) error {
	var lockedUntilVal pgtype.Timestamptz
	if lockedUntil != nil {
		lockedUntilVal = pgtype.Timestamptz{Time: *lockedUntil, Valid: true}
	} else {
		lockedUntilVal = pgtype.Timestamptz{Valid: false}
	}
	return r.q.UpdateUserLoginAttempts(ctx, identity.UpdateUserLoginAttemptsParams{
		ID:                  id,
		FailedLoginAttempts: attempts,
		LockedUntil:         lockedUntilVal,
	})
}

func (r *UserRepository) UpdateUserLastLogin(ctx context.Context, id int64) error {
	return r.q.UpdateUserLastLogin(ctx, id)
}

func (r *UserRepository) SoftDeleteUser(ctx context.Context, id int64) error {
	return r.q.SoftDeleteUser(ctx, id)
}

func (r *UserRepository) ListUsers(ctx context.Context, limit, offset int32) ([]identity.ListUsersRow, error) {
	return r.q.ListUsers(ctx, identity.ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (r *UserRepository) CountUsers(ctx context.Context) (int64, error) {
	return r.q.CountUsers(ctx)
}

func (r *UserRepository) WithTx(ctx context.Context, fn func(*UserRepository) error) error {
	return r.db.WithTx(ctx, func(tx *database.Tx) error {
		txRepo := &UserRepository{
			db: r.db,
			q:  identity.New(tx),
		}
		return fn(txRepo)
	})
}

type BuyerProfileRepository struct {
	db *database.DB
	q  *identity.Queries
}

func NewBuyerProfileRepository(db *database.DB) *BuyerProfileRepository {
	return &BuyerProfileRepository{
		db: db,
		q:  identity.New(db.Pool),
	}
}

func (r *BuyerProfileRepository) CreateBuyerProfile(ctx context.Context, params identity.CreateBuyerProfileParams) (identity.CreateBuyerProfileRow, error) {
	return r.q.CreateBuyerProfile(ctx, params)
}

func (r *BuyerProfileRepository) GetBuyerProfileByUserID(ctx context.Context, userID int64) (identity.GetBuyerProfileByUserIDRow, error) {
	return r.q.GetBuyerProfileByUserID(ctx, userID)
}

func (r *BuyerProfileRepository) GetBuyerProfileByCode(ctx context.Context, code string) (identity.GetBuyerProfileByCodeRow, error) {
	return r.q.GetBuyerProfileByCode(ctx, code)
}

func (r *BuyerProfileRepository) UpdateBuyerProfile(ctx context.Context, userID int64, req map[string]interface{}) (identity.UpdateBuyerProfileRow, error) {
	params := identity.UpdateBuyerProfileParams{ID: userID}
	if v, ok := req["business_name"].(string); ok {
		params.BusinessName = v
	}
	if v, ok := req["trading_name"].(string); ok {
		params.TradingName = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["tax_id"].(string); ok {
		params.TaxID = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["registration_number"].(string); ok {
		params.RegistrationNumber = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["phone"].(string); ok {
		params.Phone = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["website"].(string); ok {
		params.Website = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["industry"].(string); ok {
		params.Industry = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["employee_count"].(int); ok {
		params.EmployeeCount = pgtype.Int4{Int32: int32(v), Valid: true}
	}
	if v, ok := req["annual_revenue_minor"].(int64); ok {
		params.AnnualRevenueMinor = pgtype.Int8{Int64: v, Valid: true}
	}
	if v, ok := req["currency"].(string); ok {
		params.Currency = v
	}
	if v, ok := req["credit_limit_minor"].(int64); ok {
		params.CreditLimitMinor = pgtype.Int8{Int64: v, Valid: true}
	}
	if v, ok := req["credit_terms_days"].(int); ok {
		params.CreditTermsDays = pgtype.Int4{Int32: int32(v), Valid: true}
	}
	if v, ok := req["is_on_credit_hold"].(bool); ok {
		params.IsOnCreditHold = v
	}
	if v, ok := req["credit_hold_reason"].(string); ok {
		params.CreditHoldReason = pgtype.Text{String: v, Valid: true}
	}
	return r.q.UpdateBuyerProfile(ctx, params)
}

func (r *BuyerProfileRepository) SoftDeleteBuyerProfile(ctx context.Context, id int64) error {
	return r.q.SoftDeleteBuyerProfile(ctx, id)
}

type AddressRepository struct {
	db *database.DB
	q  *identity.Queries
}

func NewAddressRepository(db *database.DB) *AddressRepository {
	return &AddressRepository{
		db: db,
		q:  identity.New(db.Pool),
	}
}

func (r *AddressRepository) CreateAddress(ctx context.Context, params identity.CreateAddressParams) (identity.CreateAddressRow, error) {
	return r.q.CreateAddress(ctx, params)
}

func (r *AddressRepository) GetAddressesByOwner(ctx context.Context, ownerUserID int64, ownerType identity.IdentityUserType) ([]identity.GetAddressesByOwnerRow, error) {
	return r.q.GetAddressesByOwner(ctx, identity.GetAddressesByOwnerParams{
		OwnerUserID: ownerUserID,
		OwnerType:   ownerType,
	})
}

func (r *AddressRepository) GetAddressByCode(ctx context.Context, code string) (identity.IdentityAddress, error) {
	return r.q.GetAddressByCode(ctx, code)
}
func (r *AddressRepository) UpdateAddress(ctx context.Context, code string, req map[string]interface{}) (identity.UpdateAddressRow, error) {
	// Get current address by code first
	current, err := r.q.GetAddressByCode(ctx, code)
	if err != nil {
		return identity.UpdateAddressRow{}, err
	}

	params := identity.UpdateAddressParams{ID: current.ID}
	if v, ok := req["label"].(string); ok {
		params.Label = v
	}
	if v, ok := req["recipient_name"].(string); ok {
		params.RecipientName = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["company_name"].(string); ok {
		params.CompanyName = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["line1"].(string); ok {
		params.Line1 = v
	}
	if v, ok := req["line2"].(string); ok {
		params.Line2 = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["city"].(string); ok {
		params.City = v
	}
	if v, ok := req["state_province"].(string); ok {
		params.StateProvince = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["postal_code"].(string); ok {
		params.PostalCode = v
	}
	if v, ok := req["country"].(string); ok {
		params.Country = v
	}
	if v, ok := req["phone"].(string); ok {
		params.Phone = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["is_default"].(bool); ok {
		params.IsDefault = v
	}
	if v, ok := req["handling_class"].(string); ok {
		params.HandlingClass = pgtype.Text{String: v, Valid: true}
	}
	if v, ok := req["delivery_instructions"].(string); ok {
		params.DeliveryInstructions = pgtype.Text{String: v, Valid: true}
	}

	return r.q.UpdateAddress(ctx, params)
}

func (r *AddressRepository) SetDefaultAddress(ctx context.Context, addressID int64, ownerUserID int64, ownerType identity.IdentityUserType) error {
	return r.q.SetDefaultAddress(ctx, identity.SetDefaultAddressParams{
		ID:          addressID,
		OwnerUserID: ownerUserID,
		OwnerType:   ownerType,
	})
}

func (r *AddressRepository) SoftDeleteAddress(ctx context.Context, id int64) error {
	return r.q.SoftDeleteAddress(ctx, id)
}

type VerificationRepository struct {
	db *database.DB
	q  *identity.Queries
}

func NewVerificationRepository(db *database.DB) *VerificationRepository {
	return &VerificationRepository{
		db: db,
		q:  identity.New(db.Pool),
	}
}

func (r *VerificationRepository) CreateVerificationApplication(ctx context.Context, params identity.CreateVerificationApplicationParams) (identity.CreateVerificationApplicationRow, error) {
	return r.q.CreateVerificationApplication(ctx, params)
}

func (r *VerificationRepository) GetVerificationApplicationByID(ctx context.Context, id int64) (identity.IdentityVerificationApplication, error) {
	return r.q.GetVerificationApplicationByID(ctx, id)
}

func (r *VerificationRepository) GetVerificationApplicationByBuyer(ctx context.Context, buyerProfileID int64) (identity.GetVerificationApplicationByBuyerRow, error) {
	return r.q.GetVerificationApplicationByBuyer(ctx, buyerProfileID)
}

func (r *VerificationRepository) UpdateVerificationApplicationStatus(ctx context.Context, id int64, status string, decidedBy int64, reason string) (identity.UpdateVerificationApplicationStatusRow, error) {
	var decidedByVal pgtype.Int8
	if decidedBy > 0 {
		decidedByVal = pgtype.Int8{Int64: decidedBy, Valid: true}
	}
	return r.q.UpdateVerificationApplicationStatus(ctx, identity.UpdateVerificationApplicationStatusParams{
		ID:             id,
		Status:         identity.IdentityVerificationStatus(status),
		DecidedBy:      decidedByVal,
		DecisionReason: pgtype.Text{String: reason, Valid: true},
	})
}

func (r *VerificationRepository) ListVerificationApplications(ctx context.Context, status *string, limit, offset int32) ([]identity.ListVerificationApplicationsRow, error) {
	var statusVal identity.IdentityVerificationStatus
	if status != nil {
		statusVal = identity.IdentityVerificationStatus(*status)
	}
	return r.q.ListVerificationApplications(ctx, identity.ListVerificationApplicationsParams{
		Column1: statusVal,
		Limit:   limit,
		Offset:  offset,
	})
}

func (r *VerificationRepository) CountVerificationApplications(ctx context.Context, status *string) (int64, error) {
	var statusVal identity.IdentityVerificationStatus
	if status != nil {
		statusVal = identity.IdentityVerificationStatus(*status)
	}
	return r.q.CountVerificationApplications(ctx, statusVal)
}

func (r *VerificationRepository) CreateVerificationDocument(ctx context.Context, params identity.CreateVerificationDocumentParams) (identity.CreateVerificationDocumentRow, error) {
	return r.q.CreateVerificationDocument(ctx, params)
}

func (r *VerificationRepository) GetVerificationDocuments(ctx context.Context, applicationID int64) ([]identity.GetVerificationDocumentsRow, error) {
	return r.q.GetVerificationDocuments(ctx, applicationID)
}

type RefreshTokenRepository struct {
	db *database.DB
	q  *identity.Queries
}

func NewRefreshTokenRepository(db *database.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		db: db,
		q:  identity.New(db.Pool),
	}
}

func (r *RefreshTokenRepository) CreateRefreshToken(ctx context.Context, params identity.CreateRefreshTokenParams) (identity.CreateRefreshTokenRow, error) {
	return r.q.CreateRefreshToken(ctx, params)
}

func (r *RefreshTokenRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (identity.IdentityRefreshToken, error) {
	return r.q.GetRefreshTokenByHash(ctx, tokenHash)
}

func (r *RefreshTokenRepository) RevokeRefreshToken(ctx context.Context, id int64, reason string) error {
	return r.q.RevokeRefreshToken(ctx, identity.RevokeRefreshTokenParams{
		ID:            id,
		RevokedReason: pgtype.Text{String: reason, Valid: true},
	})
}

func (r *RefreshTokenRepository) RevokeRefreshTokenFamily(ctx context.Context, familyID, reason string) error {
	return r.q.RevokeRefreshTokenFamily(ctx, identity.RevokeRefreshTokenFamilyParams{
		FamilyID:      familyID,
		RevokedReason: pgtype.Text{String: reason, Valid: true},
	})
}

func (r *RefreshTokenRepository) ReplaceRefreshToken(ctx context.Context, oldID int64, newCode, newTokenHash string, expiresAt time.Time) (identity.ReplaceRefreshTokenRow, error) {
	return r.q.ReplaceRefreshToken(ctx, identity.ReplaceRefreshTokenParams{
		ID:        oldID,
		Code:      newCode,
		TokenHash: newTokenHash,
		ExpiresAt: expiresAt,
	})
}

func (r *RefreshTokenRepository) CleanupExpiredRefreshTokens(ctx context.Context) error {
	return r.q.CleanupExpiredRefreshTokens(ctx)
}

type PasswordResetRepository struct {
	db *database.DB
	q  *identity.Queries
}

func NewPasswordResetRepository(db *database.DB) *PasswordResetRepository {
	return &PasswordResetRepository{
		db: db,
		q:  identity.New(db.Pool),
	}
}

func (r *PasswordResetRepository) CreatePasswordResetToken(ctx context.Context, params identity.CreatePasswordResetTokenParams) (identity.CreatePasswordResetTokenRow, error) {
	return r.q.CreatePasswordResetToken(ctx, params)
}

func (r *PasswordResetRepository) GetPasswordResetToken(ctx context.Context, tokenHash string) (identity.IdentityPasswordResetToken, error) {
	return r.q.GetPasswordResetToken(ctx, tokenHash)
}

func (r *PasswordResetRepository) UsePasswordResetToken(ctx context.Context, id int64) error {
	return r.q.UsePasswordResetToken(ctx, id)
}

func (r *PasswordResetRepository) CleanupExpiredPasswordResetTokens(ctx context.Context) error {
	return r.q.CleanupExpiredPasswordResetTokens(ctx)
}

type OTPRepository struct {
	db *database.DB
	q  *identity.Queries
}

func NewOTPRepository(db *database.DB) *OTPRepository {
	return &OTPRepository{
		db: db,
		q:  identity.New(db.Pool),
	}
}

func (r *OTPRepository) CreateOTPCode(ctx context.Context, params identity.CreateOTPCodeParams) (identity.CreateOTPCodeRow, error) {
	return r.q.CreateOTPCode(ctx, params)
}

func (r *OTPRepository) GetOTPCode(ctx context.Context, userID int64, purpose string) (identity.IdentityOtpCode, error) {
	return r.q.GetOTPCode(ctx, identity.GetOTPCodeParams{
		UserID:  userID,
		Purpose: purpose,
	})
}

func (r *OTPRepository) IncrementOTPAttempts(ctx context.Context, id int64) error {
	return r.q.IncrementOTPAttempts(ctx, id)
}

func (r *OTPRepository) UseOTPCode(ctx context.Context, id int64) error {
	return r.q.UseOTPCode(ctx, id)
}

func (r *OTPRepository) CleanupExpiredOTPCodes(ctx context.Context) error {
	return r.q.CleanupExpiredOTPCodes(ctx)
}

type SessionRepository struct {
	db *database.DB
	q  *identity.Queries
}

func NewSessionRepository(db *database.DB) *SessionRepository {
	return &SessionRepository{
		db: db,
		q:  identity.New(db.Pool),
	}
}

func (r *SessionRepository) CreateSession(ctx context.Context, params identity.CreateSessionParams) (identity.CreateSessionRow, error) {
	return r.q.CreateSession(ctx, params)
}

func (r *SessionRepository) GetSessionByCode(ctx context.Context, code string) (identity.IdentitySession, error) {
	return r.q.GetSessionByCode(ctx, code)
}

func (r *SessionRepository) RevokeSession(ctx context.Context, id int64) error {
	return r.q.RevokeSession(ctx, id)
}

func (r *SessionRepository) RevokeUserSessions(ctx context.Context, userID int64) error {
	return r.q.RevokeUserSessions(ctx, userID)
}

func (r *SessionRepository) CleanupExpiredSessions(ctx context.Context) error {
	return r.q.CleanupExpiredSessions(ctx)
}

type RoleRepository struct {
	db *database.DB
	q  *identity.Queries
}

func NewRoleRepository(db *database.DB) *RoleRepository {
	return &RoleRepository{
		db: db,
		q:  identity.New(db.Pool),
	}
}

func (r *RoleRepository) CreateRole(ctx context.Context, params identity.CreateRoleParams) (identity.CreateRoleRow, error) {
	return r.q.CreateRole(ctx, params)
}

func (r *RoleRepository) GetRoleByCode(ctx context.Context, code string) (identity.GetRoleByCodeRow, error) {
	return r.q.GetRoleByCode(ctx, code)
}

func (r *RoleRepository) ListRoles(ctx context.Context) ([]identity.ListRolesRow, error) {
	return r.q.ListRoles(ctx)
}

func (r *RoleRepository) CreatePermission(ctx context.Context, params identity.CreatePermissionParams) (identity.CreatePermissionRow, error) {
	return r.q.CreatePermission(ctx, params)
}

func (r *RoleRepository) GetPermissionByCode(ctx context.Context, code string) (identity.GetPermissionByCodeRow, error) {
	return r.q.GetPermissionByCode(ctx, code)
}

func (r *RoleRepository) ListPermissions(ctx context.Context) ([]identity.ListPermissionsRow, error) {
	return r.q.ListPermissions(ctx)
}

func (r *RoleRepository) AssignRoleToUser(ctx context.Context, userID, roleID int64, assignedBy int64) error {
	return r.q.AssignRoleToUser(ctx, identity.AssignRoleToUserParams{
		UserID:     userID,
		RoleID:     roleID,
		AssignedBy: pgtype.Int8{Int64: assignedBy, Valid: true},
	})
}

func (r *RoleRepository) RemoveRoleFromUser(ctx context.Context, userID, roleID int64) error {
	return r.q.RemoveRoleFromUser(ctx, identity.RemoveRoleFromUserParams{
		UserID: userID,
		RoleID: roleID,
	})
}

func (r *RoleRepository) GetUserRoles(ctx context.Context, userID int64) ([]identity.GetUserRolesRow, error) {
	return r.q.GetUserRoles(ctx, userID)
}

func (r *RoleRepository) GetUserPermissions(ctx context.Context, userID int64) ([]identity.GetUserPermissionsRow, error) {
	return r.q.GetUserPermissions(ctx, userID)
}

func (r *RoleRepository) AssignPermissionToRole(ctx context.Context, roleID, permissionID int64) error {
	return r.q.AssignPermissionToRole(ctx, identity.AssignPermissionToRoleParams{
		RoleID:       roleID,
		PermissionID: permissionID,
	})
}
