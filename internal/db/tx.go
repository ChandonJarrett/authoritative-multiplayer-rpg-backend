package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TxFunc defines the signature of a function that can be executed within a transaction.
type TxFunc func(ctx context.Context, tx pgx.Tx) error

// InTx executes the provided function within a transaction using the default isolation level.
func InTx(ctx context.Context, pool *pgxpool.Pool, fn TxFunc) error {
	return inTxWithOptions(ctx, pool, pgx.TxOptions{}, fn)
}

// InSerializableTx executes the provided function within a transaction using the Serializable isolation level.
func InSerializableTx(ctx context.Context, pool *pgxpool.Pool, fn TxFunc) error {
	return inTxWithOptions(ctx, pool, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	}, fn)
}

// InReadCommittedTx executes the provided function within a transaction using the Read Committed isolation level.
func InReadCommittedTx(ctx context.Context, pool *pgxpool.Pool, fn TxFunc) error {
	return inTxWithOptions(ctx, pool, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	}, fn)
}

func inTxWithOptions(ctx context.Context, pool *pgxpool.Pool, opts pgx.TxOptions, fn TxFunc) error {
	if pool == nil {
		return ErrNilPool
	}
	if fn == nil {
		return errors.New("transaction function is nil")
	}

	tx, err := pool.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	committed := false

	defer func() {
		if !committed {
			rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
