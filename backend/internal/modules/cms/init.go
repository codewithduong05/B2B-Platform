package cms

import (
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/cms/router"
	"github.com/atlas-platform/backend/internal/modules/cms/service"
)

func NewService(db *database.DB) *service.CMSService {
	return service.NewCMSService(db)
}

func New(svc *service.CMSService) *router.Router {
	return router.New(svc)
}
