package ai

import (
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/modules/ai/router"
	"github.com/atlas-platform/backend/internal/modules/ai/service"
)

func NewService(db *database.DB) *service.AIService {
	return service.NewAIService(db)
}

func New(svc *service.AIService) *router.Router {
	return router.New(svc)
}
