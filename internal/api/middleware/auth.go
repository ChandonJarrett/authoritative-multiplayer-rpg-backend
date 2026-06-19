// Package middleware provides HTTP and ConnectRPC middleware for the API server.
package middleware

import (
	"context"
	"strings"

	"connectrpc.com/connect"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api/rpcerror"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	redisstore "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store/redis"
)

// PublicProcedures returns RPC procedures that do not require authentication.
func PublicProcedures() map[string]struct{} {
	return map[string]struct{}{
		"/rpg.v1.SystemService/Ping":   {},
		"/rpg.v1.AuthService/Register": {},
		"/rpg.v1.AuthService/Login":    {},
	}
}

// NewAuthInterceptor creates an authentication interceptor that checks for a valid session token.
func NewAuthInterceptor(sessions redisstore.SessionStore, publicMethods ...map[string]struct{}) connect.UnaryInterceptorFunc {
	public := PublicProcedures()
	if len(publicMethods) > 0 && publicMethods[0] != nil {
		public = publicMethods[0]
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if _, ok := public[req.Spec().Procedure]; ok {
				return next(ctx, req)
			}

			token, err := BearerToken(req.Header().Get("Authorization"))
			if err != nil {
				return nil, rpcerror.ToConnectError(err)
			}

			userID, err := sessions.GetSessionUserID(ctx, token)
			if err != nil {
				return nil, rpcerror.ToConnectError(err)
			}

			ctx = ContextWithAuthUser(ctx, AuthUser{UserID: userID})
			return next(ctx, req)
		}
	}
}

// BearerToken extracts a bearer token from an Authorization header.
func BearerToken(header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", domain.ErrUnauthenticated
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", domain.ErrUnauthenticated
	}

	return token, nil
}
