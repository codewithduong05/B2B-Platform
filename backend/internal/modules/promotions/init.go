package promotions

import (
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/promotions/router"
	"github.com/atlas-platform/backend/internal/modules/promotions/service"
)

func NewService(db *database.DB, publisher service.EventPublisher) *service.PromotionService {
	return service.NewPromotionService(db, publisher)
}

func New(svc *service.PromotionService) *router.Router {
	return router.New(svc)
}
