package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/atlas-platform/backend/internal/health"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
)

type Server struct {
	httpServer *http.Server
	router     *chi.Mux
	config     *config.HTTPConfig
}

func New(cfg *config.Config, healthHandler *health.Health) *Server {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	logger := httplog.NewLogger("atlas-api", httplog.Options{
		JSON:            cfg.Log.Format == "json",
		Concise:         true,
		RequestHeaders:  false,
		ResponseHeaders: false,
	})
	router.Use(httplog.RequestLogger(logger))

	router.Get("/health", healthHandler.LivenessHandler)
	router.Get("/readyz", healthHandler.ReadinessHandler)

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
		})
	})

	httpServer := &http.Server{
		Addr:         cfg.HTTPAddr(),
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	return &Server{
		httpServer: httpServer,
		router:     router,
		config:     &cfg.HTTP,
	}
}

func (s *Server) Router() *chi.Mux {
	return s.router
}

func (s *Server) Start(ctx context.Context) error {
	slog.InfoContext(ctx, "starting HTTP server", slog.String("addr", s.httpServer.Addr))

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(ctx, "HTTP server error", slog.String("error", err.Error()))
		}
	}()

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.InfoContext(ctx, "shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) WaitForShutdown(ctx context.Context) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.InfoContext(ctx, "received signal", slog.String("signal", sig.String()))
	case <-ctx.Done():
		slog.InfoContext(ctx, "context cancelled")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		slog.ErrorContext(ctx, "server shutdown error", slog.String("error", err.Error()))
	} else {
		slog.InfoContext(ctx, "server shutdown complete")
	}
}
