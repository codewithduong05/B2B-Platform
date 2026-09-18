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
	catalog_service "github.com/atlas-platform/backend/internal/modules/catalog/service"
	"github.com/atlas-platform/backend/internal/modules/identity"
	identity_service "github.com/atlas-platform/backend/internal/modules/identity/service"
	"github.com/atlas-platform/backend/internal/modules/pricing"
	pricing_service "github.com/atlas-platform/backend/internal/modules/pricing/service"
	"github.com/atlas-platform/backend/internal/server"
)

var version = "dev"

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
