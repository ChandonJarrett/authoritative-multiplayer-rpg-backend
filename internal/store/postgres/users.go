// Package postgres implements PostgreSQL stores.
package postgres

import (
	"context"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/db"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserStore stores users in PostgreSQL.
type UserStore struct {
	pool *pgxpool.Pool
}

// NewUserStore creates a PostgreSQL user store.
func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

// CreateUser inserts a new user.
func (s *UserStore) CreateUser(ctx context.Context, user domain.User) error {
	if s == nil || s.pool == nil {
		return db.ErrNilPool
	}

	const query = `
        INSERT INTO users (
            id,
            email,
            password_hash
        )
        VALUES (
            $1,
            $2,
            $3
        )
    `
	_, err := s.pool.Exec(ctx, query, user.ID, user.Email, user.PasswordHash)
	if err != nil {
		return mapPostgresError(err)
	}

	return nil
}

// GetUserByEmail returns a user by normalized email address.
func (s *UserStore) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	if s == nil || s.pool == nil {
		return domain.User{}, db.ErrNilPool
	}

	const query = `
        SELECT
            id,
            email,
            password_hash,
            created_at,
            updated_at
        FROM users
        WHERE email = $1
    `
	user, err := scanUser(s.pool.QueryRow(ctx, query, email))
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

// GetUserByID returns a user by ID.
func (s *UserStore) GetUserByID(ctx context.Context, userID string) (domain.User, error) {
	if s == nil || s.pool == nil {
		return domain.User{}, db.ErrNilPool
	}

	const query = `
        SELECT
            id,
            email,
            password_hash,
            created_at,
            updated_at
        FROM users
        WHERE id = $1
    `
	user, err := scanUser(s.pool.QueryRow(ctx, query, userID))
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func scanUser(row scanner) (domain.User, error) {
	var user domain.User

	if err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return domain.User{}, mapPostgresError(err)
	}

	return user, nil
}
