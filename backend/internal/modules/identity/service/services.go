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
	buyerRepo   *repository.BuyerProfileRepository
	addressRepo *repository.AddressRepository
	userRepo    *repository.UserRepository
}

func NewBuyerService() *BuyerService { return &BuyerService{} }

func (s *BuyerService) SetDependencies(buyerRepo *repository.BuyerProfileRepository, addressRepo *repository.AddressRepository, userRepo *repository.UserRepository) {
	s.buyerRepo = buyerRepo
	s.addressRepo = addressRepo
	s.userRepo = userRepo
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

	profile, err := s.buyerRepo.UpdateBuyerProfile(ctx, userID, updates)
	if err != nil {
		return nil, fmt.Errorf("update buyer profile: %w", err)
	}

	return s.toBuyerProfileResponse(profile), nil
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

	updatedAddr, err := s.addressRepo.UpdateAddress(ctx, address.ID, updates)
	if err != nil {
		return nil, fmt.Errorf("update address: %w", err)
	}

	return s.toAddressResponseFromIdentity(updatedAddr), nil
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
