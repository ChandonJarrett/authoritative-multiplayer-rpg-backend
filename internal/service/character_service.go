package service

import (
	"context"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/validate"
	"github.com/google/uuid"
)

// Default constants for character creation.
const (
	DefaultMapID = "starter_zone"
)

// DefaultSpawn is the default spawn position for new characters.
var DefaultSpawn = domain.Vec3{X: 0, Y: 0, Z: 0}

// CharacterService provides character-related operations.
type CharacterService struct {
	characters store.CharacterStore
}

// NewCharacterService creates a new CharacterService with the given character store.
func NewCharacterService(characters store.CharacterStore) *CharacterService {
	return &CharacterService{characters: characters}
}

// CreateCharacter creates a new character for the given user ID and character name.
func (s *CharacterService) CreateCharacter(ctx context.Context, userID, name string) (string, error) {
	name, err := validate.CharacterName(name)
	if err != nil {
		return "", err
	}

	character := domain.Character{
		ID:       uuid.NewString(),
		UserID:   userID,
		Name:     name,
		MapID:    DefaultMapID,
		Position: DefaultSpawn,
	}

	if err := s.characters.CreateCharacter(ctx, character); err != nil {
		return "", err
	}

	return character.ID, nil
}

// ListCharacters lists all characters for the given user ID.
func (s *CharacterService) ListCharacters(ctx context.Context, userID string) ([]domain.Character, error) {
	return s.characters.ListCharactersByUserID(ctx, userID)
}
