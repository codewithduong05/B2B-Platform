package database

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// Database error types
var (
	ErrNotFound            = errors.New("record not found")
	ErrAlreadyExists       = errors.New("record already exists")
	ErrConstraintViolation = errors.New("constraint violation")
	ErrTransactionFailed   = errors.New("transaction failed")
	ErrConnectionFailed    = errors.New("connection failed")
	ErrMigrationFailed     = errors.New("migration failed")
)

// Error wraps a PostgreSQL error with additional context
type Error struct {
	Code    string
	Message string
	Detail  string
	Hint    string
	Column  string
	Table   string
	Schema  string
	Err     error
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("database error (%s): %s", e.Code, e.Message)
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "database error"
}

func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new database error from a PostgreSQL error
//
//go:noinline
func NewError(err error) *Error {
	var pgErr *pgconn.PgError
	ok := errors.As(err, &pgErr)
	if ok {
		return &Error{
			Code:    string(pgErr.Code),
			Message: pgErr.Message,
			Detail:  pgErr.Detail,
			Hint:    pgErr.Hint,
			Column:  pgErr.ColumnName,
			Table:   pgErr.TableName,
			Schema:  pgErr.SchemaName,
			Err:     err,
		}
	}
	return &Error{Err: err}
}

// IsNotFound checks if the error is a "not found" error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsAlreadyExists checks if the error is a unique constraint violation
func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists) || IsUniqueViolation(err)
}

// IsConstraintViolation checks if the error is a constraint violation
func IsConstraintViolation(err error) bool {
	return errors.Is(err, ErrConstraintViolation) ||
		IsUniqueViolation(err) ||
		IsForeignKeyViolation(err) ||
		IsNotNullViolation(err) ||
		IsCheckViolation(err)
}

// ErrorCode returns the PostgreSQL error code
func ErrorCode(err error) string {
	var pgErr *pgconn.PgError
	ok := errors.As(err, &pgErr)
	if ok {
		return string(pgErr.Code)
	}
	return ""
}

// ErrorMessage returns the PostgreSQL error message
func ErrorMessage(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Message
	}
	return err.Error()
}

// ErrorDetail returns the PostgreSQL error detail
func ErrorDetail(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Detail
	}
	return ""
}

// Common PostgreSQL error codes
const (
	ErrCodeUniqueViolation      = "23505"
	ErrCodeForeignKeyViolation  = "23503"
	ErrCodeNotNullViolation     = "23502"
	ErrCodeCheckViolation       = "23514"
	ErrCodeSerializationFailure = "40001"
	ErrCodeDeadlockDetected     = "40P01"
	ErrCodeLockNotAvailable     = "55P03"
	ErrCodeConnectionFailure    = "08006"
	ErrCodeConnectionException  = "08001"
	ErrCodeConnectionRejected   = "08004"
	ErrCodeAdminShutdown        = "57P01"
	ErrCodeCrashShutdown        = "57P02"
	ErrCodeCannotConnectNow     = "57P03"
)

// IsRetryable checks if the error is retryable
func IsRetryable(err error) bool {
	code := ErrorCode(err)
	switch code {
	case ErrCodeSerializationFailure,
		ErrCodeDeadlockDetected,
		ErrCodeLockNotAvailable,
		ErrCodeConnectionFailure,
		ErrCodeConnectionException,
		ErrCodeConnectionRejected,
		ErrCodeAdminShutdown,
		ErrCodeCrashShutdown,
		ErrCodeCannotConnectNow:
		return true
	}
	return false
}
