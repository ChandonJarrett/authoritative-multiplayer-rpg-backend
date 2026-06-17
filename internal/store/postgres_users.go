package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresUserStore implements the UserStore interface using PostgreSQL as the backend.
type PostgresUserStore struct {
	pool *pgxpool.Pool
}

// NewPostgresUserStore creates a new instance of PostgresUserStore with the given connection pool.
func NewPostgresUserStore(pool *pgxpool.Pool) *PostgresUserStore {
	return &PostgresUserStore{pool: pool}
}

// CreateUser inserts a new user into the database.
func (s *PostgresUserStore) CreateUser(ctx context.Context, user domain.User) error {
	const query = `
INSERT INTO users (id, email, password_hash)
VALUES ($1, $2, $3)
`

	_, err := s.pool.Exec(ctx, query, user.ID, user.Email, user.PasswordHash)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("user email already exists: %w", domain.ErrAlreadyExists)
	}

	return fmt.Errorf("create user: %w", err)
}

// GetUserByEmail retrieves a user by their email address.
func (s *PostgresUserStore) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	const query = `
SELECT id, email, password_hash, created_at, updated_at
FROM users
WHERE email = $1
`

	return scanUser(s.pool.QueryRow(ctx, query, email))
}

// GetUserByID retrieves a user by their unique ID.
func (s *PostgresUserStore) GetUserByID(ctx context.Context, userID string) (domain.User, error) {
	const query = `
SELECT id, email, password_hash, created_at, updated_at
FROM users
WHERE id = $1
`

	return scanUser(s.pool.QueryRow(ctx, query, userID))
}

func scanUser(row pgx.Row) (domain.User, error) {
	var user domain.User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == nil {
		return user, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	return domain.User{}, fmt.Errorf("scan user: %w", err)
}
