package schema

import (
	"time"
)

// Auth schemas

type RegisterRequest struct {
	Email              string `json:"email" validate:"required,email"`
	Password           string `json:"password" validate:"required,min=8"`
	ConfirmPassword    string `json:"confirm_password" validate:"required,eqfield=Password"`
	BusinessName       string `json:"business_name" validate:"required"`
	TradingName        string `json:"trading_name"`
	Phone              string `json:"phone" validate:"required"`
	Website            string `json:"website"`
	Industry           string `json:"industry"`
	EmployeeCount      int    `json:"employee_count"`
	AnnualRevenue      int64  `json:"annual_revenue"`
	Currency           string `json:"currency" validate:"len=3"`
	TaxID              string `json:"tax_id"`
	RegistrationNumber string `json:"registration_number"`
}

type RegisterResponse struct {
	UserCode    string `json:"user_code"`
	Message     string `json:"message"`
	OTPRequired bool   `json:"otp_required"`
}

type RegisterVerifyRequest struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required,len=6"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"` // Only in direct API response, not in BFF cookie
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token           string `json:"token" validate:"required"`
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
}

// Buyer Profile schemas

type BuyerProfileResponse struct {
	Code               string    `json:"code"`
	BusinessName       string    `json:"business_name"`
	TradingName        string    `json:"trading_name"`
	TaxID              string    `json:"tax_id"`
	RegistrationNumber string    `json:"registration_number"`
	Phone              string    `json:"phone"`
	Website            string    `json:"website"`
	Industry           string    `json:"industry"`
	EmployeeCount      int       `json:"employee_count"`
	AnnualRevenueMinor int64     `json:"annual_revenue_minor"`
	Currency           string    `json:"currency"`
	CreditLimitMinor   int64     `json:"credit_limit_minor"`
	CreditTermsDays    int       `json:"credit_terms_days"`
	IsOnCreditHold     bool      `json:"is_on_credit_hold"`
	CreditHoldReason   string    `json:"credit_hold_reason"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type UpdateBuyerProfileRequest struct {
	BusinessName       *string `json:"business_name"`
	TradingName        *string `json:"trading_name"`
	TaxID              *string `json:"tax_id"`
	RegistrationNumber *string `json:"registration_number"`
	Phone              *string `json:"phone"`
	Website            *string `json:"website"`
	Industry           *string `json:"industry"`
	EmployeeCount      *int    `json:"employee_count"`
	AnnualRevenueMinor *int64  `json:"annual_revenue_minor"`
	Currency           *string `json:"currency"`
}

// Address schemas

type AddressResponse struct {
	Code                 string    `json:"code"`
	Label                string    `json:"label"`
	RecipientName        string    `json:"recipient_name"`
	CompanyName          string    `json:"company_name"`
	Line1                string    `json:"line1"`
	Line2                string    `json:"line2"`
	City                 string    `json:"city"`
	StateProvince        string    `json:"state_province"`
	PostalCode           string    `json:"postal_code"`
	Country              string    `json:"country"`
	Phone                string    `json:"phone"`
	IsDefault            bool      `json:"is_default"`
	HandlingClass        string    `json:"handling_class"`
	DeliveryInstructions string    `json:"delivery_instructions"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type CreateAddressRequest struct {
	Label                string `json:"label" validate:"required"`
	RecipientName        string `json:"recipient_name"`
	CompanyName          string `json:"company_name"`
	Line1                string `json:"line1" validate:"required"`
	Line2                string `json:"line2"`
	City                 string `json:"city" validate:"required"`
	StateProvince        string `json:"state_province"`
	PostalCode           string `json:"postal_code" validate:"required"`
	Country              string `json:"country" validate:"required,len=2"`
	Phone                string `json:"phone"`
	IsDefault            bool   `json:"is_default"`
	HandlingClass        string `json:"handling_class"`
	DeliveryInstructions string `json:"delivery_instructions"`
}

type UpdateAddressRequest struct {
	Label                *string `json:"label"`
	RecipientName        *string `json:"recipient_name"`
	CompanyName          *string `json:"company_name"`
	Line1                *string `json:"line1"`
	Line2                *string `json:"line2"`
	City                 *string `json:"city"`
	StateProvince        *string `json:"state_province"`
	PostalCode           *string `json:"postal_code"`
	Country              *string `json:"country"`
	Phone                *string `json:"phone"`
	IsDefault            *bool   `json:"is_default"`
	HandlingClass        *string `json:"handling_class"`
	DeliveryInstructions *string `json:"delivery_instructions"`
}

// Verification schemas

type VerificationApplicationResponse struct {
	Code                  string     `json:"code"`
	Status                string     `json:"status"`
	SubmittedAt           time.Time  `json:"submitted_at"`
	DecidedAt             *time.Time `json:"decided_at"`
	DecisionReason        string     `json:"decision_reason"`
	RejectionReason       string     `json:"rejection_reason"`
	LicenceNumber         string     `json:"licence_number"`
	LicenceExpiry         *time.Time `json:"licence_expiry"`
	TradingName           string     `json:"trading_name"`
	BusinessAddressLine1  string     `json:"business_address_line1"`
	BusinessAddressLine2  string     `json:"business_address_line2"`
	BusinessCity          string     `json:"business_city"`
	BusinessStateProvince string     `json:"business_state_province"`
	BusinessPostalCode    string     `json:"business_postal_code"`
	BusinessCountry       string     `json:"business_country"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type SubmitVerificationRequest struct {
	LicenceNumber         string `json:"licence_number" validate:"required"`
	LicenceExpiry         string `json:"licence_expiry" validate:"required"`
	TradingName           string `json:"trading_name" validate:"required"`
	BusinessAddressLine1  string `json:"business_address_line1" validate:"required"`
	BusinessAddressLine2  string `json:"business_address_line2"`
	BusinessCity          string `json:"business_city" validate:"required"`
	BusinessStateProvince string `json:"business_state_province" validate:"required"`
	BusinessPostalCode    string `json:"business_postal_code" validate:"required"`
	BusinessCountry       string `json:"business_country" validate:"required,len=2"`
}

type VerificationDocumentResponse struct {
	Code         string    `json:"code"`
	DocumentType string    `json:"document_type"`
	FileName     string    `json:"file_name"`
	FileSize     int64     `json:"file_size"`
	MimeType     string    `json:"mime_type"`
	UploadedAt   time.Time `json:"uploaded_at"`
}

type UploadVerificationDocumentRequest struct {
	DocumentType string `json:"document_type" validate:"required"`
	FileName     string `json:"file_name" validate:"required"`
	FileSize     int64  `json:"file_size" validate:"required"`
	MimeType     string `json:"mime_type" validate:"required"`
}

// Admin schemas

type AdminUserResponse struct {
	ID              int64                    `json:"id"`
	Code            string                   `json:"code"`
	Email           string                   `json:"email"`
	UserType        string                   `json:"user_type"`
	IsActive        bool                     `json:"is_active"`
	IsVerified      bool                     `json:"is_verified"`
	LastLoginAt     *time.Time               `json:"last_login_at"`
	CreatedAt       time.Time                `json:"created_at"`
	BuyerProfile    *BuyerProfileResponse    `json:"buyer_profile,omitempty"`
	SupplierProfile *SupplierProfileResponse `json:"supplier_profile,omitempty"`
}

type AdminUserUpdateRequest struct {
	IsActive *bool   `json:"is_active"`
	UserType *string `json:"user_type"`
}

type AdminSuspendRequest struct {
	Reason string `json:"reason" validate:"required"`
}

type AdminVerificationDecisionRequest struct {
	Decision string `json:"decision" validate:"required,oneof=approved rejected"`
	Reason   string `json:"reason" validate:"required"`
}

type AdminRoleResponse struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
}

type AdminPermissionResponse struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	CreatedAt   time.Time `json:"created_at"`
}

type AdminSetRolePermissionsRequest struct {
	PermissionCodes []string `json:"permission_codes" validate:"required"`
}

// Supplier profile (for admin)
type SupplierProfileResponse struct {
	Code               string     `json:"code"`
	CompanyName        string     `json:"company_name"`
	TradingName        string     `json:"trading_name"`
	TaxID              string     `json:"tax_id"`
	RegistrationNumber string     `json:"registration_number"`
	Phone              string     `json:"phone"`
	Email              string     `json:"email"`
	Website            string     `json:"website"`
	IsApproved         bool       `json:"is_approved"`
	ApprovedAt         *time.Time `json:"approved_at"`
	CreatedAt          time.Time  `json:"created_at"`
}

// Pagination
type PaginationParams struct {
	Page     int `form:"page,default=1" validate:"min=1"`
	PageSize int `form:"page_size,default=20" validate:"min=1,max=200"`
}

type PaginatedResponse[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
	HasNext  bool  `json:"has_next"`
}
