package reports

import (
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/reports/router"
	"github.com/atlas-platform/backend/internal/modules/reports/service"
)

func NewService(db *database.DB) *service.ReportsService {
	return service.NewReportsService(db)
}

func New(svc *service.ReportsService) *router.Router {
	return router.New(svc)
}
