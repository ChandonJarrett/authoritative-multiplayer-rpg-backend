package api

import (
	"context"
	"strings"

	"connectrpc.com/connect"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store"
)

// PublicProcedures returns RPC procedures that do not require authentication.
func PublicProcedures() map[string]struct{} {
	return map[string]struct{}{
		"/rpg.v1.SystemService/Ping":   {},
		"/rpg.v1.AuthService/Register": {},
		"/rpg.v1.AuthService/Login":    {},
	}
}

// NewAuthInterceptor validates bearer sessions and attaches the authenticated user to context.
func NewAuthInterceptor(sessions store.SessionStore, publicMethods ...map[string]struct{}) connect.UnaryInterceptorFunc {
	public := PublicProcedures()
	if len(publicMethods) > 0 && publicMethods[0] != nil {
		public = publicMethods[0]
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if _, ok := public[req.Spec().Procedure]; ok {
				return next(ctx, req)
			}

			if sessions == nil {
				return nil, ToConnectError(domain.ErrUnavailable)
			}

			token, err := BearerToken(req.Header().Get("Authorization"))
			if err != nil {
				return nil, ToConnectError(err)
			}

			userID, err := sessions.GetSessionUserID(ctx, token)
			if err != nil {
				return nil, ToConnectError(err)
			}

			ctx = ContextWithAuthUser(ctx, AuthUser{UserID: userID})
			return next(ctx, req)
		}
	}
}

// BearerToken extracts a bearer token from an Authorization header.
func BearerToken(header string) (string, error) {
	scheme, value, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok {
		return "", domain.ErrUnauthenticated
	}

	if !strings.EqualFold(scheme, "Bearer") {
		return "", domain.ErrUnauthenticated
	}

	token := strings.TrimSpace(value)
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", domain.ErrUnauthenticated
	}

	return token, nil
}
