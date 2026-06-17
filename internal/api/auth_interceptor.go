package api

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store"
)

// NewAuthInterceptor creates a new authentication interceptor that checks for a valid session token in the Authorization header of incoming requests.
func NewAuthInterceptor(sessions store.SessionStore) connect.UnaryInterceptorFunc {
	publicMethods := map[string]struct{}{
		"/rpg.v1.SystemService/Ping":   {},
		"/rpg.v1.AuthService/Register": {},
		"/rpg.v1.AuthService/Login":    {},
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if _, ok := publicMethods[req.Spec().Procedure]; ok {
				return next(ctx, req)
			}

			token, err := bearerToken(req.Header().Get("Authorization"))
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

func bearerToken(header string) (string, error) {
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
