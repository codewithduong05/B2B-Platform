-- Health check queries for database connectivity verification

-- name: HealthCheck :one
SELECT 1 AS ok;

-- name: HealthCheckWithTimestamp :one
SELECT 
    1 AS ok,
    NOW() AS server_time,
    version() AS postgres_version;

-- name: GetHealthCheckRecords :many
SELECT id, checked_at, status
FROM platform.health_check
ORDER BY checked_at DESC
LIMIT $1;

-- name: InsertHealthCheck :one
INSERT INTO platform.health_check (status)
VALUES ($1)
RETURNING id, checked_at, status;