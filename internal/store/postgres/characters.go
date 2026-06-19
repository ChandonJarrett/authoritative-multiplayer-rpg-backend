package postgres

import (
	"context"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/db"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CharacterStore stores characters in PostgreSQL.
type CharacterStore struct {
	pool *pgxpool.Pool
}

// NewCharacterStore creates a PostgreSQL character store.
func NewCharacterStore(pool *pgxpool.Pool) *CharacterStore {
	return &CharacterStore{pool: pool}
}

// CreateCharacter inserts a new character.
func (s *CharacterStore) CreateCharacter(ctx context.Context, character domain.Character) error {
	if s == nil || s.pool == nil {
		return db.ErrNilPool
	}

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
        VALUES (
            $1,
            $2,
            $3,
            $4,
            $5,
            $6,
            $7
        )
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
	if err != nil {
		return mapPostgresError(err)
	}

	return nil
}

// ListCharactersByUserID returns all characters owned by a user.
func (s *CharacterStore) ListCharactersByUserID(ctx context.Context, userID string) ([]domain.Character, error) {
	if s == nil || s.pool == nil {
		return nil, db.ErrNilPool
	}

	const query = `
        SELECT
            id,
            user_id,
            name,
            map_id,
            position_x,
            position_y,
            position_z,
            created_at,
            updated_at
        FROM characters
        WHERE user_id = $1
        ORDER BY created_at ASC, id ASC
    `
	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, mapPostgresError(err)
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
		return nil, mapPostgresError(err)
	}

	return characters, nil
}

// GetCharacterByID returns one character by ID.
func (s *CharacterStore) GetCharacterByID(ctx context.Context, characterID string) (domain.Character, error) {
	if s == nil || s.pool == nil {
		return domain.Character{}, db.ErrNilPool
	}

	const query = `
        SELECT
            id,
            user_id,
            name,
            map_id,
            position_x,
            position_y,
            position_z,
            created_at,
            updated_at
        FROM characters
        WHERE id = $1
    `
	character, err := scanCharacter(s.pool.QueryRow(ctx, query, characterID))
	if err != nil {
		return domain.Character{}, err
	}

	return character, nil
}

// scanCharacter maps a DB row into a domain.Character.
func scanCharacter(row scanner) (domain.Character, error) {
	var character domain.Character

	if err := row.Scan(
		&character.ID,
		&character.UserID,
		&character.Name,
		&character.MapID,
		&character.Position.X,
		&character.Position.Y,
		&character.Position.Z,
		&character.CreatedAt,
		&character.UpdatedAt,
	); err != nil {
		return domain.Character{}, mapPostgresError(err)
	}

	return character, nil
}
