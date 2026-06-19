package postgres

import (
	"errors"
	"fmt"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const uniqueViolationCode = "23505"

func mapPostgresError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolationCode:
			return domain.ErrAlreadyExists
		default:
			return fmt.Errorf("postgres error: %w", err)
		}
	}

	return fmt.Errorf("postgres query: %w", err)
}

type scanner interface {
	Scan(dest ...any) error
}
