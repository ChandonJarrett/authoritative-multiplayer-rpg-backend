package service

import (
	"context"
	"fmt"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/validate"
	"github.com/google/uuid"
)

// CharacterStore is the durable character storage required by character and handoff services.
type CharacterStore interface {
	CreateCharacter(ctx context.Context, character domain.Character) error
	ListCharactersByUserID(ctx context.Context, userID string) ([]domain.Character, error)
	GetCharacterByID(ctx context.Context, characterID string) (domain.Character, error)
}

// DefaultMapID is the ID of the default map new characters start on.
const DefaultMapID = "starter_zone"

// DefaultSpawn is the default spawn position for new characters.
var DefaultSpawn = domain.Vec3{X: 0, Y: 0, Z: 0}

// CharacterService provides character operations.
type CharacterService struct {
	characters CharacterStore
}

// NewCharacterService creates a CharacterService.
func NewCharacterService(characters CharacterStore) (*CharacterService, error) {
	if characters == nil {
		return nil, fmt.Errorf("character service character store: %w", domain.ErrInternal)
	}

	return &CharacterService{characters: characters}, nil
}

// CreateCharacter creates a new character for a user.
func (s *CharacterService) CreateCharacter(ctx context.Context, userID, name string) (string, error) {
	if s == nil {
		return "", domain.ErrInternal
	}

	userID, err := validate.RequiredID("user ID", userID)
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

	userID, err := validate.RequiredID("user ID", userID)
	if err != nil {
		return nil, err
	}

	return s.characters.ListCharactersByUserID(ctx, userID)
}
