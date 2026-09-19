package commerce

import (
	"github.com/atlas-platform/backend/internal/modules/commerce/router"
	"github.com/atlas-platform/backend/internal/modules/commerce/service"
)

func New(svc *service.CommerceService) *router.Router {
	return router.New(svc)
}
