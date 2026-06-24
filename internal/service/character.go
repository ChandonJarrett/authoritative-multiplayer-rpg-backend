package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/validate"
)

// DefaultMapID is the ID of the default map new characters start on.
const DefaultMapID = "starter_zone"

// DefaultSpawn is the default spawn position for new characters.
var DefaultSpawn = domain.Vec2{X: 0, Y: 0}

// CharacterService provides character operations.
type CharacterService struct {
	characters store.CharacterStore
}

// NewCharacterService creates a CharacterService.
func NewCharacterService(characters store.CharacterStore) (*CharacterService, error) {
	if characters == nil {
		return nil, fmt.Errorf("character store is required: %w", domain.ErrInvalidArgument)
	}
	return &CharacterService{characters: characters}, nil
}

// CreateCharacter creates a new character for a user.
func (s *CharacterService) CreateCharacter(ctx context.Context, userID, name string) (string, error) {
	if s == nil {
		return "", domain.ErrInternal
	}

	userID, err := validate.UserID(userID)
	if err != nil {
		return "", err
	}

	name, err = validate.CharacterName(name)
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

// ListCharacters returns all characters owned by a user.
func (s *CharacterService) ListCharacters(ctx context.Context, userID string) ([]domain.Character, error) {
	if s == nil {
		return nil, domain.ErrInternal
	}

	userID, err := validate.UserID(userID)
	if err != nil {
		return nil, err
	}

	return s.characters.ListCharactersByUserID(ctx, userID)
}
