package erp

import (
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/erp/router"
	"github.com/atlas-platform/backend/internal/modules/erp/service"
)

// NewService creates a new ERP service.
func NewService(db *database.DB) *service.ERPService {
	return service.NewERPService(db)
}

// New creates a new ERP router.
func New(svc *service.ERPService) *router.Router {
	return router.New(svc)
}
