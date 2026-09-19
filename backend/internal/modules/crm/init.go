package crm

import (
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/crm/router"
	"github.com/atlas-platform/backend/internal/modules/crm/service"
)

func NewService(db *database.DB) *service.CRMService {
	return service.NewCRMService(db)
}

func New(svc *service.CRMService) *router.Router {
	return router.New(svc)
}
