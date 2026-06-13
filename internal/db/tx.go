package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrNilTxFunc is returned when a nil function is passed to InTx.
var ErrNilTxFunc = errors.New("transaction function is nil")

// RollbackTimeout is the maximum time allowed for an automatic rollback when
// a transaction function fails or a commit cannot be attempted.
var RollbackTimeout = 5 * time.Second

// TxBeginner is the minimal interface required to open a transaction; *pgxpool.Pool satisfies this interface.
type TxBeginner interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

// TxFunc is work executed inside a managed transaction.
type TxFunc func(ctx context.Context, tx pgx.Tx) error

// TxCommitter is the minimal transaction lifecycle interface used by RunAndCommit.
// pgx.Tx satisfies this interface. Exposed so unit tests can supply a lightweight
// stub to exercise commit and rollback paths without a real database.
type TxCommitter interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// InTx runs fn inside a transaction with the default isolation level.
// The transaction commits on success and rolls back on any error.
func InTx(ctx context.Context, beginner TxBeginner, fn TxFunc) error {
	return inTxWithOptions(ctx, beginner, pgx.TxOptions{}, fn)
}

// InSerializableTx runs fn inside a Serializable transaction.
func InSerializableTx(ctx context.Context, beginner TxBeginner, fn TxFunc) error {
	return inTxWithOptions(ctx, beginner, pgx.TxOptions{IsoLevel: pgx.Serializable}, fn)
}

// InReadCommittedTx runs fn inside a Read Committed transaction.
func InReadCommittedTx(ctx context.Context, beginner TxBeginner, fn TxFunc) error {
	return inTxWithOptions(ctx, beginner, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, fn)
}

func inTxWithOptions(ctx context.Context, beginner TxBeginner, opts pgx.TxOptions, fn TxFunc) error {
	if isNilBeginner(beginner) {
		return ErrNilPool
	}
	if fn == nil {
		return ErrNilTxFunc
	}

	tx, err := beginner.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	return RunAndCommit(ctx, tx, func() error { return fn(ctx, tx) })
}

// RunAndCommit executes work within a transaction and commits on success.
// On any failure the transaction is rolled back within RollbackTimeout using a
// detached context so the rollback is not cancelled by the caller's context.
//
// Exposed to allow unit tests to verify commit and rollback behaviour using a
// TxCommitter stub rather than a live database connection.
func RunAndCommit(ctx context.Context, tx TxCommitter, work func() error) error {
	committed := false

	defer func() {
		if !committed {
			rctx, cancel := context.WithTimeout(context.Background(), RollbackTimeout)
			defer cancel()
			_ = tx.Rollback(rctx)
		}
	}()

	if err := work(); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	committed = true

	return nil
}

// isNilBeginner returns true if beginner is a nil interface or a typed nil pointer.
func isNilBeginner(beginner TxBeginner) bool {
	if beginner == nil {
		return true
	}
	v := reflect.ValueOf(beginner)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
