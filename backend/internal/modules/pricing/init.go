package pricing

import (
	"github.com/atlas-platform/backend/internal/modules/pricing/router"
	"github.com/atlas-platform/backend/internal/modules/pricing/service"
)

func New(svc *service.Services) *router.Router {
	return router.New(svc)
}
