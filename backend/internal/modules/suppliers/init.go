package suppliers

import (
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/suppliers/router"
	"github.com/atlas-platform/backend/internal/modules/suppliers/service"
)

func NewService(db *database.DB) *service.SupplierService {
	return service.NewSupplierService(db)
}

func New(svc *service.SupplierService) *router.Router {
	return router.New(svc)
}
