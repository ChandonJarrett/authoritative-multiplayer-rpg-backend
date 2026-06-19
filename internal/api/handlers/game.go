package handlers

import (
	"context"

	"connectrpc.com/connect"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api"
	rpgv1 "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1"
	rpgv1connect "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1/rpgv1connect"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/service"
)

var _ rpgv1connect.GameServiceHandler = (*GameHandler)(nil)

// GameHandler implements game handoff RPCs.
type GameHandler struct {
	games *service.GameService
}

// NewGameHandler creates a GameHandler.
func NewGameHandler(games *service.GameService) *GameHandler {
	return &GameHandler{games: games}
}

// ListGameServers lists available game servers.
func (h *GameHandler) ListGameServers(
	ctx context.Context,
	_ *connect.Request[rpgv1.ListGameServersRequest],
) (*connect.Response[rpgv1.ListGameServersResponse], error) {
	if _, err := api.RequireAuthUser(ctx); err != nil {
		return nil, api.ToConnectError(err)
	}

	servers, err := h.games.ListGameServers(ctx)
	if err != nil {
		return nil, api.ToConnectError(err)
	}

	out := make([]*rpgv1.GameServerSummary, 0, len(servers))
	for _, server := range servers {
		out = append(out, &rpgv1.GameServerSummary{
			GameServerId:   server.ID,
			GameServerAddr: server.Addr,
		})
	}

	return connect.NewResponse(&rpgv1.ListGameServersResponse{
		Servers: out,
	}), nil
}

// IssueJoinToken issues a short-lived token for game server handoff.
func (h *GameHandler) IssueJoinToken(
	ctx context.Context,
	req *connect.Request[rpgv1.IssueJoinTokenRequest],
) (*connect.Response[rpgv1.IssueJoinTokenResponse], error) {
	user, err := api.RequireAuthUser(ctx)
	if err != nil {
		return nil, api.ToConnectError(err)
	}

	result, err := h.games.IssueJoinToken(
		ctx,
		user.UserID,
		req.Msg.CharacterId,
		req.Msg.GameServerId,
	)
	if err != nil {
		return nil, api.ToConnectError(err)
	}

	return connect.NewResponse(&rpgv1.IssueJoinTokenResponse{
		JoinToken:        result.JoinToken,
		GameServerId:     result.GameServerID,
		GameServerAddr:   result.GameServerAddr,
		ExpiresInSeconds: result.ExpiresInSeconds,
	}), nil
}
