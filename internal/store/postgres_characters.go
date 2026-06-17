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

// PostgresCharacterStore implements the CharacterStore interface using PostgreSQL as the backend.
type PostgresCharacterStore struct {
	pool *pgxpool.Pool
}

// NewPostgresCharacterStore creates a new instance of PostgresCharacterStore with the given connection pool.
func NewPostgresCharacterStore(pool *pgxpool.Pool) *PostgresCharacterStore {
	return &PostgresCharacterStore{pool: pool}
}

// CreateCharacter inserts a new character into the database.
func (s *PostgresCharacterStore) CreateCharacter(ctx context.Context, character domain.Character) error {
	const query = `
INSERT INTO characters (
    id,
    user_id,
    name,
    map_id,
    position_x,
    position_y,
    position_z
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`

	_, err := s.pool.Exec(
		ctx,
		query,
		character.ID,
		character.UserID,
		character.Name,
		character.MapID,
		character.Position.X,
		character.Position.Y,
		character.Position.Z,
	)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("character already exists: %w", domain.ErrAlreadyExists)
	}

	return fmt.Errorf("create character: %w", err)
}

// ListCharactersByUserID retrieves all characters associated with the given user ID.
func (s *PostgresCharacterStore) ListCharactersByUserID(ctx context.Context, userID string) ([]domain.Character, error) {
	const query = `
SELECT id, user_id, name, map_id, position_x, position_y, position_z, created_at, updated_at
FROM characters
WHERE user_id = $1
ORDER BY created_at ASC
`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list characters: %w", err)
	}
	defer rows.Close()

	characters := make([]domain.Character, 0)
	for rows.Next() {
		character, err := scanCharacter(rows)
		if err != nil {
			return nil, err
		}
		characters = append(characters, character)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list characters rows: %w", err)
	}

	return characters, nil
}

// GetCharacterByID retrieves a character by their ID.
func (s *PostgresCharacterStore) GetCharacterByID(ctx context.Context, characterID string) (domain.Character, error) {
	const query = `
SELECT id, user_id, name, map_id, position_x, position_y, position_z, created_at, updated_at
FROM characters
WHERE id = $1
`

	character, err := scanCharacter(s.pool.QueryRow(ctx, query, characterID))
	if err == nil {
		return character, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Character{}, domain.ErrNotFound
	}
	return domain.Character{}, err
}

type characterRow interface {
	Scan(dest ...any) error
}

func scanCharacter(row characterRow) (domain.Character, error) {
	var character domain.Character
	err := row.Scan(
		&character.ID,
		&character.UserID,
		&character.Name,
		&character.MapID,
		&character.Position.X,
		&character.Position.Y,
		&character.Position.Z,
		&character.CreatedAt,
		&character.UpdatedAt,
	)
	if err != nil {
		return domain.Character{}, fmt.Errorf("scan character: %w", err)
	}
	return character, nil
}
