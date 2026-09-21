package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/health"
	"github.com/atlas-platform/backend/internal/logger"
	"github.com/atlas-platform/backend/internal/messaging"
	"github.com/atlas-platform/backend/internal/modules/ai"
	"github.com/atlas-platform/backend/internal/modules/catalog"
	catalog_repo "github.com/atlas-platform/backend/internal/modules/catalog/repository"
	catalog_service "github.com/atlas-platform/backend/internal/modules/catalog/service"
	"github.com/atlas-platform/backend/internal/modules/cms"
	"github.com/atlas-platform/backend/internal/modules/commerce"
	commerce_repo "github.com/atlas-platform/backend/internal/modules/commerce/repository"
	commerce_service "github.com/atlas-platform/backend/internal/modules/commerce/service"
	"github.com/atlas-platform/backend/internal/modules/crm"
	"github.com/atlas-platform/backend/internal/modules/erp"
	"github.com/atlas-platform/backend/internal/modules/identity"
	identity_service "github.com/atlas-platform/backend/internal/modules/identity/service"
	"github.com/atlas-platform/backend/internal/modules/inventory"
	"github.com/atlas-platform/backend/internal/modules/payments"
	"github.com/atlas-platform/backend/internal/modules/platform"
	"github.com/atlas-platform/backend/internal/modules/pricing"
	pricing_service "github.com/atlas-platform/backend/internal/modules/pricing/service"
	"github.com/atlas-platform/backend/internal/modules/promotions"
	"github.com/atlas-platform/backend/internal/modules/reports"
	reports_repo "github.com/atlas-platform/backend/internal/modules/reports/repository"
	"github.com/atlas-platform/backend/internal/modules/suppliers"
	"github.com/atlas-platform/backend/internal/server"
	"github.com/atlas-platform/backend/internal/storage"
	"github.com/atlas-platform/backend/internal/worker"
	"github.com/go-chi/chi/v5"
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

func erpWebhookSecret() string {
	if v := os.Getenv("ERP_WEBHOOK_SECRET"); v != "" {
		return v
	}
	return "local-dev-erp-secret"
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

	// Collect module handlers for the API dispatcher.
	// chi v5 does not allow multiple Mount("/", ...) calls, so we use a
	// dispatcher that tries each module's chi.Router sequentially.
	var moduleHandlers []http.Handler

	// Identity
	identityRouter := identity.Router
	identityRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	moduleHandlers = append(moduleHandlers, identityRouter.ChiRouter())

	// Catalog
	catalogRouter := catalog.New(catalogServices)
	catalogRouter.RegisterRoutes()
	moduleHandlers = append(moduleHandlers, catalogRouter.ChiRouter())

	// Pricing
	pricingRouter := pricing.New(pricingServices)
	pricingRouter.RegisterRoutes()
	moduleHandlers = append(moduleHandlers, pricingRouter.ChiRouter())

	// Inventory
	inventoryRouter := inventory.New(inventoryService)
	inventoryRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	moduleHandlers = append(moduleHandlers, inventoryRouter.ChiRouter())

	// Commerce
	commerceRouter := commerce.New(commerceService)
	commerceRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	moduleHandlers = append(moduleHandlers, commerceRouter.ChiRouter())

	// Promotions
	promotionsRouter := promotions.New(promotionService)
	promotionsRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	moduleHandlers = append(moduleHandlers, promotionsRouter.ChiRouter())

	// Payments
	paymentsRouter := payments.New(paymentService)
	paymentsRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	moduleHandlers = append(moduleHandlers, paymentsRouter.ChiRouter())

	// CRM
	crmService := crm.NewService(db)
	crmRouter := crm.New(crmService)
	crmRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	moduleHandlers = append(moduleHandlers, crmRouter.ChiRouter())

	// Reports
	reportsService := reports.NewService(db)
	reportsRouter := reports.New(reportsService)
	reportsRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	moduleHandlers = append(moduleHandlers, reportsRouter.ChiRouter())

	// CMS
	cmsService := cms.NewService(db)
	cmsRouter := cms.New(cmsService)
	cmsRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	moduleHandlers = append(moduleHandlers, cmsRouter.ChiRouter())

	// Suppliers
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
	moduleHandlers = append(moduleHandlers, supplierRouter.ChiRouter())

	// ERP
	erpService := erp.NewService(db)
	erpService.SetWebhookSecret(erpWebhookSecret())
	erpRouter := erp.New(erpService)
	erpRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	moduleHandlers = append(moduleHandlers, erpRouter.ChiRouter())

	// Platform
	platformService := platform.NewService(db)
	platformRouter := platform.New(platformService)
	platformRouter.RegisterRoutes(
		authMiddleware(authService),
		adminMiddleware(nil),
	)
	moduleHandlers = append(moduleHandlers, platformRouter.ChiRouter())

	// AI — registers routes directly on a dedicated chi.Router instead of
	// calling Register(apiRouter) which requires the shared router.
	aiService := ai.NewService(db)
	aiTempRouter := chi.NewRouter()
	ai.New(aiService).Register(aiTempRouter)
	moduleHandlers = append(moduleHandlers, aiTempRouter)

	// Mount the API dispatcher at /api/v1. The dispatcher strips the
	// prefix before forwarding to module handlers.
	apiDispatcher := server.NewModuleDispatcher("/api/v1", moduleHandlers...)
	srv.Router().Handle("/api/v1/*", apiDispatcher)

	// /api/v1/ping — lightweight liveness probe under the API prefix.
	srv.Router().Get("/api/v1/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Provider webhooks live outside /api/v1 per contract.
	webhookDispatcher := server.NewModuleDispatcher("",
		paymentsRouter.WebhookRouter(),
		erpRouter.WebhookRouter(),
	)
	srv.Router().Handle("/webhooks/*", webhookDispatcher)

	// Start export worker
	storageRoot := os.Getenv("EXPORT_STORAGE_DIR")
	if storageRoot == "" {
		storageRoot = "/tmp/atlas-exports"
	}
	baseURL := os.Getenv("EXPORT_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080/api/v1"
	}
	store, err := storage.NewLocalStorage(storageRoot, baseURL)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create storage", slog.String("error", err.Error()))
		os.Exit(1)
	}
	reportsRepo := reports_repo.NewReportsRepository(db)
	exportWorker := worker.NewExportWorker(reportsRepo, store, slog.Default())

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := exportWorker.ProcessNext(ctx); err != nil {
					slog.ErrorContext(ctx, "export worker error", slog.String("error", err.Error()))
				}
			}
		}
	}()

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
