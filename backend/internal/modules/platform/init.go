package platform

import (
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/platform/router"
	"github.com/atlas-platform/backend/internal/modules/platform/service"
)

func NewService(db *database.DB) *service.PlatformService {
	return service.NewPlatformService(db)
}

func New(svc *service.PlatformService) *router.Router {
	return router.New(svc)
}
