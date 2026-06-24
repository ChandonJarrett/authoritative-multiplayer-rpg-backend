package handlers

import (
	"context"

	"connectrpc.com/connect"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api"
	rpgv1 "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1"
	rpgv1connect "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1/rpgv1connect"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/service"
)

var _ rpgv1connect.AuthServiceHandler = (*AuthHandler)(nil)

// AuthHandler implements authentication RPCs.
type AuthHandler struct {
	auth *service.AuthService
}

// NewAuthHandler creates an AuthHandler.
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
		return nil, api.ToConnectError(err)
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
		return nil, api.ToConnectError(err)
	}

	return connect.NewResponse(&rpgv1.LoginResponse{
		UserId:       result.UserID,
		SessionToken: result.SessionToken,
	}), nil
}

// Logout revokes the current authenticated bearer session.
func (h *AuthHandler) Logout(
	ctx context.Context,
	_ *connect.Request[rpgv1.LogoutRequest],
) (*connect.Response[rpgv1.LogoutResponse], error) {
	authUser, err := api.RequireAuthUser(ctx)
	if err != nil {
		return nil, api.ToConnectError(err)
	}

	if err := h.auth.RevokeSession(ctx, authUser.SessionToken); err != nil {
		return nil, api.ToConnectError(err)
	}

	return connect.NewResponse(&rpgv1.LogoutResponse{}), nil
}
