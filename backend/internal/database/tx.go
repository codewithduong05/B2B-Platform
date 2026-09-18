package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tx is a wrapper around pgx.Tx for transaction management
type Tx struct {
	pgx.Tx
}

// BeginTx starts a new transaction with the given options
func (db *DB) BeginTx(ctx context.Context, opts pgx.TxOptions) (*Tx, error) {
	tx, err := db.Pool.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	return &Tx{tx}, nil
}

// Begin starts a new transaction with default options
func (db *DB) Begin(ctx context.Context) (*Tx, error) {
	return db.BeginTx(ctx, pgx.TxOptions{})
}

// WithTx executes a function within a transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func (db *DB) WithTx(ctx context.Context, fn func(*Tx) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("transaction failed: %w, rollback failed: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}

// WithTxOptions executes a function within a transaction with custom options
func (db *DB) WithTxOptions(ctx context.Context, opts pgx.TxOptions, fn func(*Tx) error) error {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("transaction failed: %w, rollback failed: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}

// IsRetryableError checks if a PostgreSQL error is retryable
func IsRetryableError(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	// Retryable error codes:
	// 40001 - serialization_failure
	// 40P01 - deadlock_detected
	// 55P03 - lock_not_available
	// 08006 - connection_failure
	// 08001 - sqlclient_unable_to_establish_sqlconnection
	// 08004 - sqlserver_rejected_establishment_of_sqlconnection
	// 57P01 - admin_shutdown
	// 57P02 - crash_shutdown
	// 57P03 - cannot_connect_now
	switch pgErr.Code {
	case "40001", "40P01", "55P03", "08006", "08001", "08004", "57P01", "57P02", "57P03":
		return true
	}
	return false
}

// IsUniqueViolation checks if a PostgreSQL error is a unique constraint violation
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505"
}

// IsForeignKeyViolation checks if a PostgreSQL error is a foreign key violation
func IsForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23503"
}

// IsNotNullViolation checks if a PostgreSQL error is a not-null violation
func IsNotNullViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23502"
}

// IsCheckViolation checks if a PostgreSQL error is a check constraint violation
func IsCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23514"
}

// IsSerializationFailure checks if a PostgreSQL error is a serialization failure
func IsSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "40001"
}

// IsDeadlockDetected checks if a PostgreSQL error is a deadlock
func IsDeadlockDetected(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "40P01"
}

// ErrCode extracts the PostgreSQL error code from an error
func ErrCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return string(pgErr.Code)
	}
	return ""
}

// ErrMessage extracts the PostgreSQL error message from an error
func ErrMessage(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Message
	}
	return err.Error()
}

// ErrDetail extracts the PostgreSQL error detail from an error
func ErrDetail(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Detail
	}
	return ""
}

// ErrHint extracts the PostgreSQL error hint from an error
func ErrHint(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Hint
	}
	return ""
}

// AcquireTx acquires a connection and begins a transaction
func (db *DB) AcquireTx(ctx context.Context) (*Tx, *pgxpool.Conn, error) {
	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("acquire connection: %w", err)
	}

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		conn.Release()
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}

	return &Tx{tx}, conn, nil
}
