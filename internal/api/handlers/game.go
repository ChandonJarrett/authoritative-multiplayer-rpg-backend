package handlers

import (
	"context"

	"connectrpc.com/connect"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api/middleware"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api/rpcerror"
	rpgv1 "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1"
	rpgv1connect "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1/rpgv1connect"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/service"
)

var _ rpgv1connect.GameServiceHandler = (*GameHandler)(nil)

// GameHandler implements the GameService, handling game-related API requests.
type GameHandler struct {
	handoff *service.GameHandoffService
}

// NewGameHandler creates a new GameHandler.
func NewGameHandler(handoff *service.GameHandoffService) *GameHandler {
	return &GameHandler{handoff: handoff}
}

// IssueJoinToken handles the API request to issue a join token for a game server.
func (h *GameHandler) IssueJoinToken(
	ctx context.Context,
	req *connect.Request[rpgv1.IssueJoinTokenRequest],
) (*connect.Response[rpgv1.IssueJoinTokenResponse], error) {
	user, err := middleware.RequireAuthUser(ctx)
	if err != nil {
		return nil, rpcerror.ToConnectError(err)
	}

	result, err := h.handoff.IssueJoinToken(ctx, user.UserID, req.Msg.CharacterId)
	if err != nil {
		return nil, rpcerror.ToConnectError(err)
	}

	return connect.NewResponse(&rpgv1.IssueJoinTokenResponse{
		JoinToken:        result.JoinToken,
		GameServerId:     result.GameServerID,
		GameServerAddr:   result.GameServerAddr,
		ExpiresInSeconds: result.ExpiresInSeconds,
	}), nil
}
