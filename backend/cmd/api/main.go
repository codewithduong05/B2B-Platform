package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/health"
	"github.com/atlas-platform/backend/internal/logger"
	"github.com/atlas-platform/backend/internal/messaging"
	"github.com/atlas-platform/backend/internal/modules/catalog"
	catalog_repo "github.com/atlas-platform/backend/internal/modules/catalog/repository"
	catalog_service "github.com/atlas-platform/backend/internal/modules/catalog/service"
	"github.com/atlas-platform/backend/internal/modules/commerce"
	commerce_repo "github.com/atlas-platform/backend/internal/modules/commerce/repository"
	commerce_service "github.com/atlas-platform/backend/internal/modules/commerce/service"
	"github.com/atlas-platform/backend/internal/modules/crm"
	"github.com/atlas-platform/backend/internal/modules/identity"
	identity_service "github.com/atlas-platform/backend/internal/modules/identity/service"
	"github.com/atlas-platform/backend/internal/modules/inventory"
	"github.com/atlas-platform/backend/internal/modules/payments"
	"github.com/atlas-platform/backend/internal/modules/pricing"
	pricing_service "github.com/atlas-platform/backend/internal/modules/pricing/service"
	"github.com/atlas-platform/backend/internal/modules/promotions"
	"github.com/atlas-platform/backend/internal/modules/suppliers"
	"github.com/atlas-platform/backend/internal/server"
)

var version = "dev"

// webhookSecret reads a per-provider webhook HMAC secret. Local default is
// explicit and logged by the caller environment, never a production value.
func webhookSecret(provider string) string {
	if v := os.Getenv("PAYMENTS_WEBHOOK_SECRET_" + provider); v != "" {
		return v
	}
	if v := os.Getenv("PAYMENTS_WEBHOOK_SECRET"); v != "" {
		return v
	}
	return "local-dev-secret"
}

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Init(cfg.Log.Level, cfg.Log.Format, cfg.App.ServiceName)

	// Run database migrations
	if err := database.RunMigrations(ctx, &cfg.Postgres, cfg.Migration.Path); err != nil {
		slog.ErrorContext(ctx, "failed to run migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	db, err := database.New(ctx, &cfg.Postgres)
	if err != nil {
		slog.ErrorContext(ctx, "failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	var rmq *messaging.RabbitMQ
	rmq, err = messaging.New(ctx, &cfg.RabbitMQ)
	if err != nil {
		slog.WarnContext(ctx, "failed to connect to rabbitmq, continuing without it", slog.String("error", err.Error()))
	} else {
		defer rmq.Close()
	}

	// Initialize identity module services
	authService := identity_service.NewAuthService()
	// TODO: Add other identity services

	// Initialize catalog module services
	catalogServices := catalog_service.NewServices(db)

	// Initialize pricing module services
	pricingServices := pricing_service.NewServices(db)

	// Initialize inventory module services
	inventoryService := inventory.NewService(db, nil)

	// Initialize promotions module services (nil publisher: best-effort events)
	promotionService := promotions.NewService(db, nil)

	// Initialize commerce module services
	commerceService := commerce_service.NewCommerceService(db, pricingServices.PriceList, inventoryService, nil, promotionService)

	// Initialize payments module services (nil publisher: best-effort events)
	paymentService := payments.NewService(db, commerceService, nil)
	paymentService.SetWebhookSecrets(map[string]string{
		"sim":  webhookSecret("sim"),
		"bank": webhookSecret("bank"),
	})
	// Credit gate for checkout (optional seam; nil disables).
	commerceService.SetCreditChecker(paymentService)

	healthHandler := health.New(db, nil, version)
	srv := server.New(cfg, healthHandler)

	// Register identity module routes
	identityRouter := identity.Router
	identityRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil), // TODO: Implement admin service
	)
	srv.Router().Mount("/api/v1", identityRouter.ChiRouter())

	// Register catalog module routes
	catalogRouter := catalog.New(catalogServices)
	catalogRouter.RegisterRoutes()
	srv.Router().Mount("/api/v1", catalogRouter.ChiRouter())

	// Register pricing module routes
	pricingRouter := pricing.New(pricingServices)
	pricingRouter.RegisterRoutes()
	srv.Router().Mount("/api/v1", pricingRouter.ChiRouter())

	// Register inventory module routes
	inventoryRouter := inventory.New(inventoryService)
	inventoryRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil), // TODO: Implement admin permission check (inventory.quarantine)
	)
	srv.Router().Mount("/api/v1", inventoryRouter.ChiRouter())

	// Register commerce module routes
	commerceRouter := commerce.New(commerceService)
	commerceRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	srv.Router().Mount("/api/v1", commerceRouter.ChiRouter())

	// Register promotions module routes
	promotionsRouter := promotions.New(promotionService)
	promotionsRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	srv.Router().Mount("/api/v1", promotionsRouter.ChiRouter())

	// Register payments module routes
	paymentsRouter := payments.New(paymentService)
	paymentsRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	srv.Router().Mount("/api/v1", paymentsRouter.ChiRouter())
	// Provider webhooks mount outside /api/v1 per contract.
	srv.Router().Mount("/", paymentsRouter.WebhookRouter())

	// Register CRM module routes
	crmService := crm.NewService(db)
	crmRouter := crm.New(crmService)
	crmRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	srv.Router().Mount("/api/v1", crmRouter.ChiRouter())

	// Register suppliers module routes
	supplierService := suppliers.NewService(db)
	supplierService.SetPortalDependencies(
		catalog_repo.NewSupplierRepository(db),
		catalog_repo.NewProductRepository(db),
		commerce_repo.NewCommerceRepository(db),
		commerceService,
		inventoryService,
	)
	supplierRouter := suppliers.New(supplierService)
	supplierRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	srv.Router().Mount("/api/v1", supplierRouter.ChiRouter())

	if err := srv.Start(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to start server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	srv.WaitForShutdown(ctx)
}

func authMiddleware(authService *identity_service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: Implement JWT validation
			next.ServeHTTP(w, r)
		})
	}
}

func adminMiddleware(adminService interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: Implement admin permission check
			next.ServeHTTP(w, r)
		})
	}
}
