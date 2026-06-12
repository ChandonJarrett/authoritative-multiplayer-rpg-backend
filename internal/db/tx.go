package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrNilTxFunc indicates the transaction function is nil.
var ErrNilTxFunc = errors.New("transaction function is nil")

var rollbackTimeout = 5 * time.Second

// TxBeginner is the minimal dependency required to start a transaction.
type TxBeginner interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

// TxFunc defines work executed inside a transaction.
type TxFunc func(ctx context.Context, tx pgx.Tx) error

// InTx executes a function inside a transaction using the default isolation level.
func InTx(ctx context.Context, beginner TxBeginner, fn TxFunc) error {
	return inTxWithOptions(ctx, beginner, pgx.TxOptions{}, fn)
}

// InSerializableTx executes a function inside a Serializable transaction.
func InSerializableTx(ctx context.Context, beginner TxBeginner, fn TxFunc) error {
	return inTxWithOptions(ctx, beginner, pgx.TxOptions{IsoLevel: pgx.Serializable}, fn)
}

// InReadCommittedTx executes a function inside a Read Committed transaction.
func InReadCommittedTx(ctx context.Context, beginner TxBeginner, fn TxFunc) error {
	return inTxWithOptions(ctx, beginner, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, fn)
}

func inTxWithOptions(ctx context.Context, beginner TxBeginner, opts pgx.TxOptions, fn TxFunc) error {
	if isNil(beginner) {
		return ErrNilPool
	}
	if fn == nil {
		return ErrNilTxFunc
	}

	tx, err := beginner.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			rollbackCtx, cancel := context.WithTimeout(context.Background(), rollbackTimeout)
			defer cancel()
			_ = tx.Rollback(rollbackCtx)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		return fmt.Errorf("transaction function: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	committed = true
	return nil
}

func isNil(value any) bool {
	if value == nil {
		return true
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
