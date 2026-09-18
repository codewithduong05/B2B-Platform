package identity

import (
	"github.com/atlas-platform/backend/internal/modules/identity/router"
	"github.com/atlas-platform/backend/internal/modules/identity/service"
)

var (
	// Router is the public HTTP router for the identity module
	Router = router.New()

	// AuthService provides authentication operations
	AuthService = service.NewAuthService()

	// BuyerService provides buyer profile operations
	BuyerService = service.NewBuyerService()
)
