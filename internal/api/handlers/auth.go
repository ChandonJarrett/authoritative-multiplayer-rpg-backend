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

var _ rpgv1connect.AuthServiceHandler = (*AuthHandler)(nil)

// AuthHandler implements the gRPC handlers for authentication-related operations.
type AuthHandler struct {
	auth *service.AuthService
}

// NewAuthHandler creates a new AuthHandler with the given AuthService.
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: authService}
}

// Register handles user registration requests.
func (h *AuthHandler) Register(
	ctx context.Context,
	req *connect.Request[rpgv1.RegisterRequest],
) (*connect.Response[rpgv1.RegisterResponse], error) {
	result, err := h.auth.Register(ctx, req.Msg.Email, req.Msg.Password)
	if err != nil {
		return nil, rpcerror.ToConnectError(err)
	}

	return connect.NewResponse(&rpgv1.RegisterResponse{
		UserId:       result.UserID,
		SessionToken: result.SessionToken,
	}), nil
}

// Login handles user login requests.
func (h *AuthHandler) Login(
	ctx context.Context,
	req *connect.Request[rpgv1.LoginRequest],
) (*connect.Response[rpgv1.LoginResponse], error) {
	result, err := h.auth.Login(ctx, req.Msg.Email, req.Msg.Password)
	if err != nil {
		return nil, rpcerror.ToConnectError(err)
	}

	return connect.NewResponse(&rpgv1.LoginResponse{
		UserId:       result.UserID,
		SessionToken: result.SessionToken,
	}), nil
}

// Logout revokes the current authenticated bearer session.
func (h *AuthHandler) Logout(
	ctx context.Context,
	req *connect.Request[rpgv1.LogoutRequest],
) (*connect.Response[rpgv1.LogoutResponse], error) {
	token, err := middleware.BearerToken(req.Header().Get("Authorization"))
	if err != nil {
		return nil, rpcerror.ToConnectError(err)
	}

	if err := h.auth.RevokeSession(ctx, token); err != nil {
		return nil, rpcerror.ToConnectError(err)
	}

	return connect.NewResponse(&rpgv1.LogoutResponse{}), nil
}
