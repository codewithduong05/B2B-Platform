package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/atlas-platform/backend/internal/database"
	"github.com/atlas-platform/backend/internal/messaging"
)

type HealthStatus string

const (
	StatusOK       HealthStatus = "ok"
	StatusDegraded HealthStatus = "degraded"
	StatusDown     HealthStatus = "down"
)

type ComponentHealth struct {
	Name   string         `json:"name"`
	Status HealthStatus   `json:"status"`
	Detail map[string]any `json:"detail,omitempty"`
	Error  string         `json:"error,omitempty"`
}

type HealthResponse struct {
	Status     HealthStatus      `json:"status"`
	Timestamp  string            `json:"timestamp"`
	Version    string            `json:"version"`
	Components []ComponentHealth `json:"components,omitempty"`
}

type Checker interface {
	Name() string
	Check(ctx context.Context) ComponentHealth
}

type Health struct {
	db       *database.DB
	rabbitmq *messaging.RabbitMQ
	version  string
	Checkers []Checker
}

func New(db *database.DB, rabbitmq *messaging.RabbitMQ, version string) *Health {
	return &Health{
		db:       db,
		rabbitmq: rabbitmq,
		version:  version,
		Checkers: []Checker{
			&dbChecker{db: db},
			&rabbitmqChecker{rabbitmq: rabbitmq},
		},
	}
}

func (h *Health) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:    StatusOK,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   h.version,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Health) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := HealthResponse{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   h.version,
	}

	allOK := true
	for _, checker := range h.Checkers {
		health := checker.Check(ctx)
		resp.Components = append(resp.Components, health)
		if health.Status != StatusOK {
			allOK = false
		}
	}

	if allOK {
		resp.Status = StatusOK
		w.WriteHeader(http.StatusOK)
	} else {
		resp.Status = StatusDegraded
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

type dbChecker struct {
	db *database.DB
}

func (c *dbChecker) Name() string {
	return "postgres"
}

func (c *dbChecker) Check(ctx context.Context) ComponentHealth {
	health := ComponentHealth{Name: c.Name()}

	if c.db == nil {
		health.Status = StatusDown
		health.Error = "database not configured"
		return health
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := c.db.Ping(ctx); err != nil {
		health.Status = StatusDown
		health.Error = err.Error()
		slog.ErrorContext(ctx, "health check failed", slog.String("component", c.Name()), slog.String("error", err.Error()))
		return health
	}

	stats := c.db.Stats()
	health.Status = StatusOK
	health.Detail = map[string]any{
		"open_connections":  stats.AcquiredConns(),
		"idle_connections":  stats.IdleConns(),
		"total_connections": stats.TotalConns(),
	}
	return health
}

type rabbitmqChecker struct {
	rabbitmq *messaging.RabbitMQ
}

func (c *rabbitmqChecker) Name() string {
	return "rabbitmq"
}

func (c *rabbitmqChecker) Check(ctx context.Context) ComponentHealth {
	health := ComponentHealth{Name: c.Name()}

	if c.rabbitmq == nil {
		health.Status = StatusDown
		health.Error = "not configured"
		return health
	}

	if !c.rabbitmq.IsConnected() {
		health.Status = StatusDown
		health.Error = "connection closed"
		return health
	}

	health.Status = StatusOK
	return health
}

type SQLChecker struct {
	DB          *sql.DB
	CheckerName string
	Query       string
}

func (c *SQLChecker) Name() string {
	return c.CheckerName
}

func (c *SQLChecker) Check(ctx context.Context) ComponentHealth {
	health := ComponentHealth{Name: c.Name()}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := c.DB.PingContext(ctx); err != nil {
		health.Status = StatusDown
		health.Error = err.Error()
		return health
	}

	if c.Query != "" {
		if err := c.DB.QueryRowContext(ctx, c.Query).Scan(); err != nil {
			health.Status = StatusDown
			health.Error = err.Error()
			return health
		}
	}

	health.Status = StatusOK
	return health
}
