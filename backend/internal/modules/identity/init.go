package identity

import (
	"github.com/atlas-platform/backend/internal/config"
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/identity/repository"
	"github.com/atlas-platform/backend/internal/modules/identity/router"
	"github.com/atlas-platform/backend/internal/modules/identity/service"
)

var (
	// Router is the public HTTP router for the identity module
	Router = router.New(
		service.NewAuthService(),
		service.NewBuyerService(),
		service.NewAdminService(),
	)

	// AuthService provides authentication operations
	AuthService = service.NewAuthService()

	// BuyerService provides buyer profile operations
	BuyerService = service.NewBuyerService()

	// AdminService provides admin operations
	AdminService = service.NewAdminService()
)

// NewRouter creates a new identity router with the given services.
func NewRouter(authSvc *service.AuthService, buyerSvc *service.BuyerService, adminSvc *service.AdminService) *router.Router {
	return router.New(authSvc, buyerSvc, adminSvc)
}

// Init initializes the identity module with database dependencies
func Init(db *database.DB, jwtConfig *config.JWTConfig) {
	// Create repositories
	userRepo := repository.NewUserRepository(db)
	buyerRepo := repository.NewBuyerProfileRepository(db)
	verificationRepo := repository.NewVerificationRepository(db)
	addressRepo := repository.NewAddressRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	passwordResetRepo := repository.NewPasswordResetRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	roleRepo := repository.NewRoleRepository(db)

	// Initialize AuthService
	AuthService.SetDependencies(
		db,
		userRepo,
		refreshTokenRepo,
		otpRepo,
		passwordResetRepo,
		sessionRepo,
		buyerRepo,
		jwtConfig,
	)

	// Initialize BuyerService
	BuyerService.SetDependencies(buyerRepo, addressRepo, userRepo, verificationRepo)

	// Initialize AdminService
	AdminService.SetDependencies(
		userRepo,
		buyerRepo,
		verificationRepo,
		roleRepo,
		refreshTokenRepo,
		sessionRepo,
	)
}
