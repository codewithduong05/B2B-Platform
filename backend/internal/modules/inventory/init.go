package inventory

import (
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/inventory/router"
	"github.com/atlas-platform/backend/internal/modules/inventory/service"
)

func NewService(db *database.DB, publisher service.EventPublisher) *service.InventoryService {
	return service.NewInventoryService(db, publisher)
}

func New(svc *service.InventoryService) *router.Router {
	return router.New(svc)
}
