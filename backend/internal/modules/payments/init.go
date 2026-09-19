package payments

import (
	"github.com/atlas-platform/backend/internal/database"
	commerce_service "github.com/atlas-platform/backend/internal/modules/commerce/service"
	"github.com/atlas-platform/backend/internal/modules/payments/router"
	"github.com/atlas-platform/backend/internal/modules/payments/service"
)

func NewService(db *database.DB, commerceSvc *commerce_service.CommerceService, publisher service.EventPublisher) *service.PaymentService {
	return service.NewPaymentService(db, commerceSvc, publisher)
}

func New(svc *service.PaymentService) *router.Router {
	return router.New(svc)
}
