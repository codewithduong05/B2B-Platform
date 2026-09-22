package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/atlas-platform/backend/internal/auth"
	"github.com/atlas-platform/backend/internal/config"
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/database/queries/identity"
	"github.com/atlas-platform/backend/internal/modules/identity/repository"
	"github.com/atlas-platform/backend/internal/modules/identity/schema"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUserNotFound         = errors.New("user not found")
	ErrUserInactive         = errors.New("user is inactive")
	ErrAccountLocked        = errors.New("account is temporarily locked")
	ErrInvalidOTP           = errors.New("invalid or expired OTP")
	ErrOTPMaxAttempts       = errors.New("maximum OTP attempts exceeded")
	ErrTokenExpired         = errors.New("token has expired")
	ErrTokenRevoked         = errors.New("token has been revoked")
	ErrInvalidToken         = errors.New("invalid token")
	ErrVerificationNotFound = errors.New("verification application not found")
	ErrVerificationPending  = errors.New("verification already pending")
	ErrAddressNotFound      = errors.New("address not found")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrBuyerProfileNotFound = errors.New("buyer profile not found")
	ErrRoleNotFound         = errors.New("role not found")
	ErrPermissionNotFound   = errors.New("permission not found")
	ErrOTPNotFound          = errors.New("OTP not found")
	ErrOTPExpired           = errors.New("OTP has expired")
	ErrInvalidInput         = errors.New("invalid input")
	ErrInvalidResetToken    = errors.New("invalid or expired reset token")
	ErrResetTokenUsed       = errors.New("reset token already used")
)

type AuthService struct {
	db                *database.DB
	userRepo          *repository.UserRepository
	refreshTokenRepo  *repository.RefreshTokenRepository
	otpRepo           *repository.OTPRepository
	passwordResetRepo *repository.PasswordResetRepository
	sessionRepo       *repository.SessionRepository
	buyerRepo         *repository.BuyerProfileRepository
	jwtManager        *auth.JWTManager
	tokenRevoker      auth.TokenRevoker
}

func NewAuthService() *AuthService { return &AuthService{} }

func (s *AuthService) SetDependencies(
	db *database.DB,
	userRepo *repository.UserRepository,
	refreshTokenRepo *repository.RefreshTokenRepository,
	otpRepo *repository.OTPRepository,
	passwordResetRepo *repository.PasswordResetRepository,
	sessionRepo *repository.SessionRepository,
	buyerRepo *repository.BuyerProfileRepository,
	cfg *config.JWTConfig,
) {
	s.db = db
	s.userRepo = userRepo
	s.refreshTokenRepo = refreshTokenRepo
	s.otpRepo = otpRepo
	s.passwordResetRepo = passwordResetRepo
	s.sessionRepo = sessionRepo
	s.buyerRepo = buyerRepo
	s.jwtManager = auth.NewJWTManager(cfg)
	s.tokenRevoker = &tokenRevokerImpl{refreshTokenRepo: refreshTokenRepo, sessionRepo: sessionRepo}
}

type tokenRevokerImpl struct {
	refreshTokenRepo *repository.RefreshTokenRepository
	sessionRepo      *repository.SessionRepository
}

func (t *tokenRevokerImpl) IsRevoked(ctx context.Context, tokenID string) (bool, error) {
	token, err := t.refreshTokenRepo.GetRefreshTokenByHash(ctx, tokenID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	return token.RevokedAt.Valid, nil
}

func (t *tokenRevokerImpl) Revoke(ctx context.Context, tokenID string, reason string) error {
	// Parse tokenID to int64
	var tokenIDInt int64
	fmt.Sscanf(tokenID, "%d", &tokenIDInt)
	return t.refreshTokenRepo.RevokeRefreshToken(ctx, tokenIDInt, reason)
}

func (t *tokenRevokerImpl) RevokeFamily(ctx context.Context, familyID string, reason string) error {
	return t.refreshTokenRepo.RevokeRefreshTokenFamily(ctx, familyID, reason)
}

func (s *AuthService) Register(ctx context.Context, req schema.RegisterRequest) (*schema.RegisterResponse, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetUserByEmail(ctx, strings.ToLower(req.Email))
	if err == nil && existingUser.ID != 0 {
		return nil, ErrUserAlreadyExists
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Generate user code
	userCode := generateCode()

	// Create user
	userParams := identity.CreateUserParams{
		Code:         userCode,
		Email:        strings.ToLower(req.Email),
		PasswordHash: string(passwordHash),
		UserType:     identity.IdentityUserTypeBuyer,
	}

	userRow, err := s.userRepo.CreateUser(ctx, userParams)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Create buyer profile
	buyerCode := generateCode()
	buyerParams := identity.CreateBuyerProfileParams{
		Code:               buyerCode,
		UserID:             userRow.ID,
		BusinessName:       req.BusinessName,
		TradingName:        pgtype.Text{String: req.TradingName, Valid: req.TradingName != ""},
		Phone:              pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
		Website:            pgtype.Text{String: req.Website, Valid: req.Website != ""},
		Industry:           pgtype.Text{String: req.Industry, Valid: req.Industry != ""},
		EmployeeCount:      pgtype.Int4{Int32: int32(req.EmployeeCount), Valid: true},
		AnnualRevenueMinor: pgtype.Int8{Int64: req.AnnualRevenue, Valid: true},
		Currency:           req.Currency,
		TaxID:              pgtype.Text{String: req.TaxID, Valid: req.TaxID != ""},
		RegistrationNumber: pgtype.Text{String: req.RegistrationNumber, Valid: req.RegistrationNumber != ""},
	}

	_, err = s.buyerRepo.CreateBuyerProfile(ctx, buyerParams)
	if err != nil {
		return nil, fmt.Errorf("create buyer profile: %w", err)
	}

	// Generate and send OTP for email verification
	otp := generateOTP()
	otpHash := hashOTP(otp)
	otpParams := identity.CreateOTPCodeParams{
		Code:        generateCode(),
		UserID:      userRow.ID,
		OtpHash:     otpHash,
		Purpose:     "registration",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		MaxAttempts: 3,
	}

	_, err = s.otpRepo.CreateOTPCode(ctx, otpParams)
	if err != nil {
		return nil, fmt.Errorf("create OTP: %w", err)
	}

	// TODO: Send OTP via email

	return &schema.RegisterResponse{
		UserCode:    userCode,
		Message:     "Registration successful. Please check your email for verification code.",
		OTPRequired: true,
	}, nil
}

func (s *AuthService) VerifyRegistration(ctx context.Context, req schema.RegisterVerifyRequest) (*schema.TokenResponse, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, strings.ToLower(req.Email))
	if err != nil {
		return nil, ErrUserNotFound
	}

	otp, err := s.otpRepo.GetOTPCode(ctx, user.ID, "registration")
	if err != nil {
		return nil, ErrInvalidOTP
	}

	if otp.Attempts >= otp.MaxAttempts {
		return nil, ErrOTPMaxAttempts
	}

	if !verifyOTP(req.OTP, otp.OtpHash) {
		s.otpRepo.IncrementOTPAttempts(ctx, otp.ID)
		return nil, ErrInvalidOTP
	}

	s.otpRepo.UseOTPCode(ctx, otp.ID)

	// Mark user as verified
	_, err = s.userRepo.UpdateUser(ctx, user.ID, user.Email, user.UserType, user.IsActive, true)
	if err != nil {
		return nil, fmt.Errorf("update user verified: %w", err)
	}

	return s.generateTokens(ctx, user.ID, user.Code, user.Email, user.UserType)
}

func (s *AuthService) Login(ctx context.Context, req schema.LoginRequest) (*schema.TokenResponse, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, strings.ToLower(req.Email))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if user.LockedUntil.Valid && user.LockedUntil.Time.After(time.Now()) {
		return nil, ErrAccountLocked
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		attempts := user.FailedLoginAttempts + 1
		var lockedUntil *time.Time
		if attempts >= 5 {
			lockTime := time.Now().Add(15 * time.Minute)
			lockedUntil = &lockTime
		}
		s.userRepo.UpdateUserLoginAttempts(ctx, user.ID, attempts, lockedUntil)
		return nil, ErrInvalidCredentials
	}

	s.userRepo.UpdateUserLastLogin(ctx, user.ID)

	return s.generateTokens(ctx, user.ID, user.Code, user.Email, user.UserType)
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*schema.TokenResponse, error) {
	tokenHash := hashToken(refreshToken)

	storedToken, err := s.refreshTokenRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if time.Now().After(storedToken.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	if storedToken.RevokedAt.Valid {
		return nil, ErrTokenRevoked
	}

	user, err := s.userRepo.GetUserByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// Rotate refresh token
	newRefreshToken := generateRefreshToken()
	newTokenHash := hashToken(newRefreshToken)
	newExpiresAt := time.Now().Add(30 * 24 * time.Hour)

	_, err = s.refreshTokenRepo.ReplaceRefreshToken(ctx, storedToken.ID, generateCode(), newTokenHash, newExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("replace refresh token: %w", err)
	}

	accessToken, _, err := s.jwtManager.GenerateAccessToken(user.ID, user.Code, user.Email, string(user.UserType))
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	return &schema.TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	tokenHash := hashToken(refreshToken)
	storedToken, err := s.refreshTokenRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil
	}

	return s.refreshTokenRepo.RevokeRefreshTokenFamily(ctx, storedToken.FamilyID, "logout")
}

func (s *AuthService) ForgotPassword(ctx context.Context, req schema.ForgotPasswordRequest) error {
	user, err := s.userRepo.GetUserByEmail(ctx, strings.ToLower(req.Email))
	if err != nil {
		return nil
	}

	resetToken := generateResetToken()
	tokenHash := hashToken(resetToken)

	tokenParams := identity.CreatePasswordResetTokenParams{
		Code:      generateCode(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	_, err = s.passwordResetRepo.CreatePasswordResetToken(ctx, tokenParams)
	if err != nil {
		return fmt.Errorf("create reset token: %w", err)
	}

	// TODO: Send reset token via email
	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, req schema.ResetPasswordRequest) error {
	tokenHash := hashToken(req.Token)

	token, err := s.passwordResetRepo.GetPasswordResetToken(ctx, tokenHash)
	if err != nil {
		return ErrInvalidResetToken
	}

	if time.Now().After(token.ExpiresAt) {
		return ErrInvalidResetToken
	}

	if token.UsedAt.Valid {
		return ErrResetTokenUsed
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	err = s.userRepo.UpdateUserPassword(ctx, token.UserID, string(passwordHash))
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	s.passwordResetRepo.UsePasswordResetToken(ctx, token.ID)

	s.sessionRepo.RevokeUserSessions(ctx, token.UserID)
	s.refreshTokenRepo.RevokeRefreshTokenFamily(ctx, "", "password_reset")

	return nil
}

func (s *AuthService) generateTokens(ctx context.Context, userID int64, userCode, email string, userType identity.IdentityUserType) (*schema.TokenResponse, error) {
	family := generateCode()
	accessToken, _, err := s.jwtManager.GenerateAccessToken(userID, userCode, email, string(userType))
	if err != nil {
		return nil, err
	}

	refreshToken := generateRefreshToken()
	tokenHash := hashToken(refreshToken)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	_, err = s.refreshTokenRepo.CreateRefreshToken(ctx, identity.CreateRefreshTokenParams{
		Code:      generateCode(),
		UserID:    userID,
		TokenHash: tokenHash,
		FamilyID:  family,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
	}

	sessionParams := identity.CreateSessionParams{
		Code:           generateCode(),
		UserID:         userID,
		RefreshTokenID: pgtype.Int8{Int64: 0, Valid: false},
		ExpiresAt:      expiresAt,
	}
	s.sessionRepo.CreateSession(ctx, sessionParams)

	return &schema.TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: refreshToken,
	}, nil
}

func generateCode() string {
	b := make([]byte, 16)
	rand.Read(b)
	return strings.ToLower(hex.EncodeToString(b))[:26]
}

func generateOTP() string {
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("%06d", int(b[0])%1000000)
}

func generateRefreshToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateResetToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func hashOTP(otp string) string {
	hash := sha256.Sum256([]byte(otp))
	return hex.EncodeToString(hash[:])
}

func verifyOTP(otp, hash string) bool {
	return hashOTP(otp) == hash
}

// ==================== BuyerService ====================

type BuyerService struct {
	buyerRepo        *repository.BuyerProfileRepository
	addressRepo      *repository.AddressRepository
	userRepo         *repository.UserRepository
	verificationRepo *repository.VerificationRepository
}

func NewBuyerService() *BuyerService { return &BuyerService{} }

func (s *BuyerService) SetDependencies(buyerRepo *repository.BuyerProfileRepository, addressRepo *repository.AddressRepository, userRepo *repository.UserRepository, verificationRepo *repository.VerificationRepository) {
	s.buyerRepo = buyerRepo
	s.addressRepo = addressRepo
	s.userRepo = userRepo
	s.verificationRepo = verificationRepo
}

func (s *BuyerService) GetProfile(ctx context.Context, userID int64) (*schema.BuyerProfileResponse, error) {
	profile, err := s.buyerRepo.GetBuyerProfileByUserID(ctx, userID)
	if err != nil {
		return nil, ErrBuyerProfileNotFound
	}
	return s.toBuyerProfileResponse(profile), nil
}

func (s *BuyerService) UpdateProfile(ctx context.Context, userID int64, req schema.UpdateBuyerProfileRequest) (*schema.BuyerProfileResponse, error) {
	updates := make(map[string]interface{})
	if req.BusinessName != nil {
		updates["business_name"] = *req.BusinessName
	}
	if req.TradingName != nil {
		updates["trading_name"] = *req.TradingName
	}
	if req.TaxID != nil {
		updates["tax_id"] = *req.TaxID
	}
	if req.RegistrationNumber != nil {
		updates["registration_number"] = *req.RegistrationNumber
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Website != nil {
		updates["website"] = *req.Website
	}
	if req.Industry != nil {
		updates["industry"] = *req.Industry
	}
	if req.EmployeeCount != nil {
		updates["employee_count"] = *req.EmployeeCount
	}
	if req.AnnualRevenueMinor != nil {
		updates["annual_revenue_minor"] = *req.AnnualRevenueMinor
	}
	if req.Currency != nil {
		updates["currency"] = *req.Currency
	}

	_, err := s.buyerRepo.UpdateBuyerProfile(ctx, userID, updates)
	if err != nil {
		return nil, fmt.Errorf("update buyer profile: %w", err)
	}

	// Fetch full profile after update
	fullProfile, err := s.buyerRepo.GetBuyerProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get updated buyer profile: %w", err)
	}

	return s.toBuyerProfileResponse(fullProfile), nil
}

func (s *BuyerService) ListAddresses(ctx context.Context, userID int64) ([]schema.AddressResponse, error) {
	addresses, err := s.addressRepo.GetAddressesByOwner(ctx, userID, identity.IdentityUserTypeBuyer)
	if err != nil {
		return nil, fmt.Errorf("get addresses: %w", err)
	}

	var resp []schema.AddressResponse
	for _, addr := range addresses {
		resp = append(resp, s.toAddressResponse(addr))
	}
	return resp, nil
}

func (s *BuyerService) CreateAddress(ctx context.Context, userID int64, req schema.CreateAddressRequest) (*schema.AddressResponse, error) {
	params := identity.CreateAddressParams{
		Code:                 generateCode(),
		OwnerUserID:          userID,
		OwnerType:            identity.IdentityUserTypeBuyer,
		Label:                req.Label,
		RecipientName:        pgtype.Text{String: req.RecipientName, Valid: req.RecipientName != ""},
		CompanyName:          pgtype.Text{String: req.CompanyName, Valid: req.CompanyName != ""},
		Line1:                req.Line1,
		Line2:                pgtype.Text{String: req.Line2, Valid: req.Line2 != ""},
		City:                 req.City,
		StateProvince:        pgtype.Text{String: req.StateProvince, Valid: req.StateProvince != ""},
		PostalCode:           req.PostalCode,
		Country:              req.Country,
		Phone:                pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
		IsDefault:            req.IsDefault,
		HandlingClass:        pgtype.Text{String: req.HandlingClass, Valid: req.HandlingClass != ""},
		DeliveryInstructions: pgtype.Text{String: req.DeliveryInstructions, Valid: req.DeliveryInstructions != ""},
	}

	_, err := s.addressRepo.CreateAddress(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create address: %w", err)
	}

	return &schema.AddressResponse{
		Label:                req.Label,
		RecipientName:        req.RecipientName,
		CompanyName:          req.CompanyName,
		Line1:                req.Line1,
		Line2:                req.Line2,
		City:                 req.City,
		StateProvince:        req.StateProvince,
		PostalCode:           req.PostalCode,
		Country:              req.Country,
		Phone:                req.Phone,
		IsDefault:            req.IsDefault,
		HandlingClass:        req.HandlingClass,
		DeliveryInstructions: req.DeliveryInstructions,
	}, nil
}

func (s *BuyerService) UpdateAddress(ctx context.Context, userID int64, addressCode string, req schema.UpdateAddressRequest) (*schema.AddressResponse, error) {
	address, err := s.addressRepo.GetAddressByCode(ctx, addressCode)
	if err != nil {
		return nil, ErrAddressNotFound
	}

	if address.OwnerUserID != userID || address.OwnerType != identity.IdentityUserTypeBuyer {
		return nil, ErrAddressNotFound
	}

	updates := make(map[string]interface{})
	if req.Label != nil {
		updates["label"] = *req.Label
	}
	if req.RecipientName != nil {
		updates["recipient_name"] = *req.RecipientName
	}
	if req.CompanyName != nil {
		updates["company_name"] = *req.CompanyName
	}
	if req.Line1 != nil {
		updates["line1"] = *req.Line1
	}
	if req.Line2 != nil {
		updates["line2"] = *req.Line2
	}
	if req.City != nil {
		updates["city"] = *req.City
	}
	if req.StateProvince != nil {
		updates["state_province"] = *req.StateProvince
	}
	if req.PostalCode != nil {
		updates["postal_code"] = *req.PostalCode
	}
	if req.Country != nil {
		updates["country"] = *req.Country
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.IsDefault != nil {
		updates["is_default"] = *req.IsDefault
	}
	if req.HandlingClass != nil {
		updates["handling_class"] = *req.HandlingClass
	}
	if req.DeliveryInstructions != nil {
		updates["delivery_instructions"] = *req.DeliveryInstructions
	}

	_, err = s.addressRepo.UpdateAddress(ctx, addressCode, updates)
	if err != nil {
		return nil, fmt.Errorf("update address: %w", err)
	}

	// Fetch full address after update
	fullAddr, err := s.addressRepo.GetAddressByCode(ctx, addressCode)
	if err != nil {
		return nil, fmt.Errorf("get updated address: %w", err)
	}

	addrResp := s.toAddressResponseFromIdentity(fullAddr)
	return addrResp, nil
}

func (s *BuyerService) DeleteAddress(ctx context.Context, userID int64, addressCode string) error {
	address, err := s.addressRepo.GetAddressByCode(ctx, addressCode)
	if err != nil {
		return ErrAddressNotFound
	}

	if address.OwnerUserID != userID || address.OwnerType != identity.IdentityUserTypeBuyer {
		return ErrAddressNotFound
	}

	return s.addressRepo.SoftDeleteAddress(ctx, address.ID)
}

func (s *BuyerService) GetVerificationStatus(ctx context.Context, userID int64) (*schema.VerificationApplicationResponse, error) {
	profile, err := s.buyerRepo.GetBuyerProfileByUserID(ctx, userID)
	if err != nil {
		return nil, ErrBuyerProfileNotFound
	}

	app, err := s.verificationRepo.GetVerificationApplicationByBuyer(ctx, profile.ID)
	if err != nil {
		return nil, ErrVerificationNotFound
	}

	return s.toVerificationApplicationResponse(app), nil
}

func (s *BuyerService) SubmitVerification(ctx context.Context, userID int64, req schema.SubmitVerificationRequest) (*schema.VerificationApplicationResponse, error) {
	profile, err := s.buyerRepo.GetBuyerProfileByUserID(ctx, userID)
	if err != nil {
		return nil, ErrBuyerProfileNotFound
	}

	existing, err := s.verificationRepo.GetVerificationApplicationByBuyer(ctx, profile.ID)
	if err == nil && (existing.Status == "pending" || existing.Status == "approved") {
		return nil, ErrVerificationPending
	}

	expiry, err := time.Parse("2006-01-02", req.LicenceExpiry)
	if err != nil {
		return nil, fmt.Errorf("invalid licence_expiry format (use YYYY-MM-DD): %w", ErrInvalidInput)
	}

	params := identity.CreateVerificationApplicationParams{
		Code:                  generateCode(),
		BuyerProfileID:        profile.ID,
		LicenceNumber:         pgtype.Text{String: req.LicenceNumber, Valid: req.LicenceNumber != ""},
		LicenceExpiry:         pgtype.Date{Time: expiry, Valid: true},
		TradingName:           pgtype.Text{String: req.TradingName, Valid: req.TradingName != ""},
		BusinessAddressLine1:  pgtype.Text{String: req.BusinessAddressLine1, Valid: req.BusinessAddressLine1 != ""},
		BusinessAddressLine2:  pgtype.Text{String: req.BusinessAddressLine2, Valid: req.BusinessAddressLine2 != ""},
		BusinessCity:          pgtype.Text{String: req.BusinessCity, Valid: req.BusinessCity != ""},
		BusinessStateProvince: pgtype.Text{String: req.BusinessStateProvince, Valid: req.BusinessStateProvince != ""},
		BusinessPostalCode:    pgtype.Text{String: req.BusinessPostalCode, Valid: req.BusinessPostalCode != ""},
		BusinessCountry:       req.BusinessCountry,
	}

	row, err := s.verificationRepo.CreateVerificationApplication(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create verification application: %w", err)
	}

	return &schema.VerificationApplicationResponse{
		Code:                  row.Code,
		Status:                string(row.Status),
		SubmittedAt:           row.CreatedAt,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
		LicenceNumber:         req.LicenceNumber,
		LicenceExpiry:         &expiry,
		TradingName:           req.TradingName,
		BusinessAddressLine1:  req.BusinessAddressLine1,
		BusinessAddressLine2:  req.BusinessAddressLine2,
		BusinessCity:          req.BusinessCity,
		BusinessStateProvince: req.BusinessStateProvince,
		BusinessPostalCode:    req.BusinessPostalCode,
		BusinessCountry:       req.BusinessCountry,
	}, nil
}

func (s *BuyerService) toVerificationApplicationResponse(app identity.GetVerificationApplicationByBuyerRow) *schema.VerificationApplicationResponse {
	return &schema.VerificationApplicationResponse{
		Code:            app.Code,
		Status:          string(app.Status),
		SubmittedAt:     app.SubmittedAt,
		DecisionReason:  app.DecisionReason.String,
		RejectionReason: app.RejectionReason.String,
		CreatedAt:       app.CreatedAt,
		UpdatedAt:       app.UpdatedAt,
	}
}

func (s *BuyerService) toBuyerProfileResponse(profile identity.GetBuyerProfileByUserIDRow) *schema.BuyerProfileResponse {
	return &schema.BuyerProfileResponse{
		Code:               profile.Code,
		BusinessName:       profile.BusinessName,
		TradingName:        profile.TradingName.String,
		TaxID:              profile.TaxID.String,
		RegistrationNumber: profile.RegistrationNumber.String,
		Phone:              profile.Phone.String,
		Website:            profile.Website.String,
		Industry:           profile.Industry.String,
		EmployeeCount:      int(profile.EmployeeCount.Int32),
		AnnualRevenueMinor: profile.AnnualRevenueMinor.Int64,
		Currency:           profile.Currency,
		CreditLimitMinor:   profile.CreditLimitMinor.Int64,
		CreditTermsDays:    int(profile.CreditTermsDays.Int32),
		IsOnCreditHold:     profile.IsOnCreditHold,
		CreditHoldReason:   profile.CreditHoldReason.String,
		CreatedAt:          profile.CreatedAt,
		UpdatedAt:          profile.UpdatedAt,
	}
}

func (s *BuyerService) toAddressResponse(addr identity.GetAddressesByOwnerRow) schema.AddressResponse {
	return schema.AddressResponse{
		Code:                 addr.Code,
		Label:                addr.Label,
		RecipientName:        addr.RecipientName.String,
		CompanyName:          addr.CompanyName.String,
		Line1:                addr.Line1,
		Line2:                addr.Line2.String,
		City:                 addr.City,
		StateProvince:        addr.StateProvince.String,
		PostalCode:           addr.PostalCode,
		Country:              addr.Country,
		Phone:                addr.Phone.String,
		IsDefault:            addr.IsDefault,
		HandlingClass:        addr.HandlingClass.String,
		DeliveryInstructions: addr.DeliveryInstructions.String,
		CreatedAt:            addr.CreatedAt,
		UpdatedAt:            addr.UpdatedAt,
	}
}

func (s *BuyerService) toAddressResponseFromIdentity(addr identity.IdentityAddress) *schema.AddressResponse {
	return &schema.AddressResponse{
		Code:                 addr.Code,
		Label:                addr.Label,
		RecipientName:        addr.RecipientName.String,
		CompanyName:          addr.CompanyName.String,
		Line1:                addr.Line1,
		Line2:                addr.Line2.String,
		City:                 addr.City,
		StateProvince:        addr.StateProvince.String,
		PostalCode:           addr.PostalCode,
		Country:              addr.Country,
		Phone:                addr.Phone.String,
		IsDefault:            addr.IsDefault,
		HandlingClass:        addr.HandlingClass.String,
		DeliveryInstructions: addr.DeliveryInstructions.String,
		CreatedAt:            addr.CreatedAt,
		UpdatedAt:            addr.UpdatedAt,
	}
}

// ==================== AdminService ====================

type AdminService struct {
	userRepo         *repository.UserRepository
	buyerRepo        *repository.BuyerProfileRepository
	verificationRepo *repository.VerificationRepository
	roleRepo         *repository.RoleRepository
	refreshTokenRepo *repository.RefreshTokenRepository
	sessionRepo      *repository.SessionRepository
}

func NewAdminService() *AdminService { return &AdminService{} }

func (s *AdminService) SetDependencies(
	userRepo *repository.UserRepository,
	buyerRepo *repository.BuyerProfileRepository,
	verificationRepo *repository.VerificationRepository,
	roleRepo *repository.RoleRepository,
	refreshTokenRepo *repository.RefreshTokenRepository,
	sessionRepo *repository.SessionRepository,
) {
	s.userRepo = userRepo
	s.buyerRepo = buyerRepo
	s.verificationRepo = verificationRepo
	s.roleRepo = roleRepo
	s.refreshTokenRepo = refreshTokenRepo
	s.sessionRepo = sessionRepo
}

// ListUsers returns paginated list of users
func (s *AdminService) ListUsers(ctx context.Context, page, pageSize int32) ([]schema.AdminUserResponse, int64, error) {
	users, err := s.userRepo.ListUsers(ctx, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	total, err := s.userRepo.CountUsers(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	var resp []schema.AdminUserResponse
	for _, u := range users {
		resp = append(resp, s.toAdminUserResponse(u))
	}
	return resp, total, nil
}

// GetUserByID returns a user by ID with profiles
func (s *AdminService) GetUserByID(ctx context.Context, id int64) (*schema.AdminUserResponse, error) {
	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	resp := s.toAdminUserResponseFromUser(user)
	return resp, nil
}

// UpdateUser updates a user
func (s *AdminService) UpdateUser(ctx context.Context, id int64, req schema.AdminUserUpdateRequest) (*schema.AdminUserResponse, error) {
	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	isActive := user.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	userType := user.UserType
	if req.UserType != nil {
		userType = identity.IdentityUserType(*req.UserType)
	}

	updated, err := s.userRepo.UpdateUser(ctx, id, user.Email, userType, isActive, user.IsVerified)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return s.toAdminUserResponseFromIdentity(updated), nil
}

// SuspendUser suspends a user
func (s *AdminService) SuspendUser(ctx context.Context, id int64, req schema.AdminSuspendRequest) error {
	user, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("get user: %w", err)
	}

	if !user.IsActive {
		return nil // Already inactive
	}

	_, err = s.userRepo.UpdateUser(ctx, id, user.Email, user.UserType, false, user.IsVerified)
	if err != nil {
		return fmt.Errorf("suspend user: %w", err)
	}

	// Revoke all sessions and refresh tokens
	s.sessionRepo.RevokeUserSessions(ctx, id)
	s.refreshTokenRepo.RevokeRefreshTokenFamily(ctx, "", "suspended: "+req.Reason)

	return nil
}

// ListVerifications returns paginated verification applications
func (s *AdminService) ListVerifications(ctx context.Context, status *string, page, pageSize int32) ([]schema.VerificationApplicationResponse, int64, error) {
	apps, err := s.verificationRepo.ListVerificationApplications(ctx, status, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list verifications: %w", err)
	}
	total, err := s.verificationRepo.CountVerificationApplications(ctx, status)
	if err != nil {
		return nil, 0, fmt.Errorf("count verifications: %w", err)
	}

	var resp []schema.VerificationApplicationResponse
	for _, a := range apps {
		resp = append(resp, s.toVerificationResponse(a))
	}
	return resp, total, nil
}

// DecideVerification approves or rejects a verification
func (s *AdminService) DecideVerification(ctx context.Context, id int64, req schema.AdminVerificationDecisionRequest, decidedBy int64) (*schema.VerificationApplicationResponse, error) {
	if req.Decision != "approved" && req.Decision != "rejected" {
		return nil, ErrInvalidInput
	}

	_, err := s.verificationRepo.UpdateVerificationApplicationStatus(ctx, id, req.Decision, decidedBy, req.Reason)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVerificationNotFound
		}
		return nil, fmt.Errorf("decide verification: %w", err)
	}

	// Update buyer profile verification status
	app, err := s.verificationRepo.GetVerificationApplicationByID(ctx, id)
	if err == nil {
		verificationStatus := "pending"
		if req.Decision == "approved" {
			verificationStatus = "approved"
		} else if req.Decision == "rejected" {
			verificationStatus = "rejected"
		}
		updates := map[string]interface{}{"verification_status": verificationStatus}
		s.buyerRepo.UpdateBuyerProfile(ctx, app.BuyerProfileID, updates)
	}

	app, err = s.verificationRepo.GetVerificationApplicationByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get verification: %w", err)
	}

	return s.toVerificationResponseFromIdentity(app), nil
}

// ListRoles returns all roles
func (s *AdminService) ListRoles(ctx context.Context) ([]schema.AdminRoleResponse, error) {
	roles, err := s.roleRepo.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	var resp []schema.AdminRoleResponse
	for _, r := range roles {
		resp = append(resp, schema.AdminRoleResponse{
			ID:          r.ID,
			Code:        r.Code,
			Name:        r.Name,
			Description: r.Description.String,
			IsSystem:    r.IsSystem,
			CreatedAt:   r.CreatedAt,
		})
	}
	return resp, nil
}

// SetRolePermissions sets permissions for a role
func (s *AdminService) SetRolePermissions(ctx context.Context, roleCode string, req schema.AdminSetRolePermissionsRequest) error {
	role, err := s.roleRepo.GetRoleByCode(ctx, roleCode)
	if err != nil {
		return ErrRoleNotFound
	}

	// Remove all existing permissions
	// Note: This would require a DeleteRolePermissions query which we don't have
	// For now, we just add new ones (they're inserted with ON CONFLICT DO NOTHING)
	for _, permCode := range req.PermissionCodes {
		perm, err := s.roleRepo.GetPermissionByCode(ctx, permCode)
		if err != nil {
			return fmt.Errorf("permission %s not found: %w", permCode, ErrPermissionNotFound)
		}
		if err := s.roleRepo.AssignPermissionToRole(ctx, role.ID, perm.ID); err != nil {
			return fmt.Errorf("assign permission: %w", err)
		}
	}
	return nil
}

// GetUserPermissions returns permissions for a user
func (s *AdminService) GetUserPermissions(ctx context.Context, userID int64) ([]schema.AdminPermissionResponse, error) {
	perms, err := s.roleRepo.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user permissions: %w", err)
	}

	var resp []schema.AdminPermissionResponse
	for _, p := range perms {
		resp = append(resp, schema.AdminPermissionResponse{
			ID:       p.ID,
			Code:     p.Code,
			Name:     p.Name,
			Resource: p.Resource,
			Action:   p.Action,
		})
	}
	return resp, nil
}

func (s *AdminService) toAdminUserResponse(u identity.ListUsersRow) schema.AdminUserResponse {
	var lastLoginAt *time.Time
	if u.LastLoginAt.Valid {
		t := u.LastLoginAt.Time
		lastLoginAt = &t
	}
	return schema.AdminUserResponse{
		ID:          u.ID,
		Code:        u.Code,
		Email:       u.Email,
		UserType:    string(u.UserType),
		IsActive:    u.IsActive,
		IsVerified:  u.IsVerified,
		LastLoginAt: lastLoginAt,
		CreatedAt:   u.CreatedAt,
	}
}

func (s *AdminService) toAdminUserResponseFromIdentity(u identity.UpdateUserRow) *schema.AdminUserResponse {
	return &schema.AdminUserResponse{
		ID:         u.ID,
		Code:       u.Code,
		Email:      u.Email,
		UserType:   string(u.UserType),
		IsActive:   u.IsActive,
		IsVerified: u.IsVerified,
		CreatedAt:  u.CreatedAt,
	}
}

func (s *AdminService) toAdminUserResponseFromUser(u identity.IdentityUser) *schema.AdminUserResponse {
	var lastLoginAt *time.Time
	if u.LastLoginAt.Valid {
		t := u.LastLoginAt.Time
		lastLoginAt = &t
	}
	return &schema.AdminUserResponse{
		ID:          u.ID,
		Code:        u.Code,
		Email:       u.Email,
		UserType:    string(u.UserType),
		IsActive:    u.IsActive,
		IsVerified:  u.IsVerified,
		LastLoginAt: lastLoginAt,
		CreatedAt:   u.CreatedAt,
	}
}

func (s *AdminService) toVerificationResponse(v identity.ListVerificationApplicationsRow) schema.VerificationApplicationResponse {
	var decidedAt *time.Time
	if v.DecidedAt.Valid {
		t := v.DecidedAt.Time
		decidedAt = &t
	}
	return schema.VerificationApplicationResponse{
		Code:            v.Code,
		Status:          string(v.Status),
		SubmittedAt:     v.SubmittedAt,
		DecidedAt:       decidedAt,
		DecisionReason:  v.DecisionReason.String,
		RejectionReason: v.RejectionReason.String,
		CreatedAt:       v.CreatedAt,
		UpdatedAt:       v.UpdatedAt,
	}
}

func (s *AdminService) toVerificationResponseFromIdentity(v identity.IdentityVerificationApplication) *schema.VerificationApplicationResponse {
	var decidedAt *time.Time
	if v.DecidedAt.Valid {
		t := v.DecidedAt.Time
		decidedAt = &t
	}
	return &schema.VerificationApplicationResponse{
		Code:            v.Code,
		Status:          string(v.Status),
		SubmittedAt:     v.SubmittedAt,
		DecidedAt:       decidedAt,
		DecisionReason:  v.DecisionReason.String,
		RejectionReason: v.RejectionReason.String,
		CreatedAt:       v.CreatedAt,
		UpdatedAt:       v.UpdatedAt,
	}
}
