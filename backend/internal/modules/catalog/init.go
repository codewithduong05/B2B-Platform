package catalog

import (
	"github.com/atlas-platform/backend/internal/modules/catalog/router"
	"github.com/atlas-platform/backend/internal/modules/catalog/service"
)

func New(svc *service.Services) *router.Router {
	return router.New(svc)
}
