package middleware

import (
	"context"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

type authContextKey struct{}

// AuthUser represents the authenticated user information stored in the context.
type AuthUser struct {
	UserID string
}

// ContextWithAuthUser returns a new context with the given AuthUser stored in it.
func ContextWithAuthUser(ctx context.Context, user AuthUser) context.Context {
	return context.WithValue(ctx, authContextKey{}, user)
}

// AuthUserFromContext retrieves the AuthUser from the context.
func AuthUserFromContext(ctx context.Context) (AuthUser, bool) {
	user, ok := ctx.Value(authContextKey{}).(AuthUser)
	return user, ok
}

// RequireAuthUser returns the authenticated user or ErrUnauthenticated.
func RequireAuthUser(ctx context.Context) (AuthUser, error) {
	user, ok := AuthUserFromContext(ctx)
	if !ok {
		return AuthUser{}, domain.ErrUnauthenticated
	}
	return user, nil
}
