package handlers

import (
	"context"

	"connectrpc.com/connect"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	rpgv1 "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1"
	rpgv1connect "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1/rpgv1connect"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/service"
)

var _ rpgv1connect.CharacterServiceHandler = (*CharacterHandler)(nil)

// CharacterHandler implements character RPCs.
type CharacterHandler struct {
	characters *service.CharacterService
}

// NewCharacterHandler creates a CharacterHandler.
func NewCharacterHandler(characters *service.CharacterService) *CharacterHandler {
	return &CharacterHandler{characters: characters}
}

// CreateCharacter creates a character for the authenticated user.
func (h *CharacterHandler) CreateCharacter(
	ctx context.Context,
	req *connect.Request[rpgv1.CreateCharacterRequest],
) (*connect.Response[rpgv1.CreateCharacterResponse], error) {
	user, err := api.RequireAuthUser(ctx)
	if err != nil {
		return nil, api.ToConnectError(err)
	}

	characterID, err := h.characters.CreateCharacter(ctx, user.UserID, req.Msg.Name)
	if err != nil {
		return nil, api.ToConnectError(err)
	}

	return connect.NewResponse(&rpgv1.CreateCharacterResponse{
		CharacterId: characterID,
	}), nil
}

// ListCharacters lists all characters for the authenticated user.
func (h *CharacterHandler) ListCharacters(
	ctx context.Context,
	_ *connect.Request[rpgv1.ListCharactersRequest],
) (*connect.Response[rpgv1.ListCharactersResponse], error) {
	user, err := api.RequireAuthUser(ctx)
	if err != nil {
		return nil, api.ToConnectError(err)
	}

	characters, err := h.characters.ListCharacters(ctx, user.UserID)
	if err != nil {
		return nil, api.ToConnectError(err)
	}

	out := make([]*rpgv1.CharacterSummary, 0, len(characters))
	for _, character := range characters {
		out = append(out, characterToProto(character))
	}

	return connect.NewResponse(&rpgv1.ListCharactersResponse{
		Characters: out,
	}), nil
}

func characterToProto(character domain.Character) *rpgv1.CharacterSummary {
	return &rpgv1.CharacterSummary{
		CharacterId: character.ID,
		Name:        character.Name,
		MapId:       character.MapID,
		Position: &rpgv1.Vec3{
			X: character.Position.X,
			Y: character.Position.Y,
			Z: character.Position.Z,
		},
	}
}
