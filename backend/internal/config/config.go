package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App       AppConfig
	HTTP      HTTPConfig
	Postgres  PostgresConfig
	RabbitMQ  RabbitMQConfig
	Redis     RedisConfig
	Log       LogConfig
	Migration MigrationConfig
	JWT       JWTConfig
}

type AppConfig struct {
	Environment string
	ServiceName string
	Version     string
}

type HTTPConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type PostgresConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type RabbitMQConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	VHost    string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	Database int
}

type LogConfig struct {
	Level  string
	Format string
}

type MigrationConfig struct {
	Path string
}

type JWTConfig struct {
	SecretKey       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Issuer          string
	Audience        string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Environment: getEnv("APP_ENV", "local"),
			ServiceName: getEnv("APP_SERVICE_NAME", "atlas-api"),
			Version:     getEnv("APP_VERSION", "dev"),
		},
		HTTP: HTTPConfig{
			Host:         getEnv("HTTP_HOST", "0.0.0.0"),
			Port:         getEnvInt("HTTP_PORT", 8000),
			ReadTimeout:  getEnvDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getEnvDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getEnvDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		},
		Postgres: PostgresConfig{
			Host:            getEnv("POSTGRES_HOST", "localhost"),
			Port:            getEnvInt("POSTGRES_PORT", 5432),
			User:            getEnv("POSTGRES_USER", "postgres"),
			Password:        getEnv("POSTGRES_PASSWORD", "postgres"),
			Database:        getEnv("POSTGRES_DATABASE", "atlas"),
			SSLMode:         getEnv("POSTGRES_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvInt("POSTGRES_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("POSTGRES_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("POSTGRES_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getEnvDuration("POSTGRES_CONN_MAX_IDLE_TIME", 1*time.Minute),
		},
		RabbitMQ: RabbitMQConfig{
			Host:     getEnv("RABBITMQ_HOST", "localhost"),
			Port:     getEnvInt("RABBITMQ_PORT", 5672),
			User:     getEnv("RABBITMQ_USER", "guest"),
			Password: getEnv("RABBITMQ_PASSWORD", "guest"),
			VHost:    getEnv("RABBITMQ_VHOST", "/"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			Database: getEnvInt("REDIS_DATABASE", 0),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Migration: MigrationConfig{
			Path: getEnv("MIGRATION_PATH", "file://migrations"),
		},
		JWT: JWTConfig{
			SecretKey:       getEnv("JWT_SECRET_KEY", "dev-secret-change-in-production"),
			AccessTokenTTL:  getEnvDuration("JWT_ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL: getEnvDuration("JWT_REFRESH_TOKEN_TTL", 7*24*time.Hour),
			Issuer:          getEnv("JWT_ISSUER", "atlas-api"),
			Audience:        getEnv("JWT_AUDIENCE", "atlas-client"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Postgres.Host == "" {
		return errors.New("POSTGRES_HOST is required")
	}
	if c.Postgres.User == "" {
		return errors.New("POSTGRES_USER is required")
	}
	if c.Postgres.Database == "" {
		return errors.New("POSTGRES_DATABASE is required")
	}
	return nil
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Postgres.User,
		c.Postgres.Password,
		c.Postgres.Host,
		c.Postgres.Port,
		c.Postgres.Database,
		c.Postgres.SSLMode,
	)
}

func (c *Config) RabbitMQURL() string {
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%d%s",
		c.RabbitMQ.User,
		c.RabbitMQ.Password,
		c.RabbitMQ.Host,
		c.RabbitMQ.Port,
		c.RabbitMQ.VHost,
	)
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

func (c *Config) HTTPAddr() string {
	return fmt.Sprintf("%s:%d", c.HTTP.Host, c.HTTP.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
