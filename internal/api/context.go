package api

import (
	"context"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

type (
	authContextKey      struct{}
	requestIDContextKey struct{}
)

// AuthUser represents the authenticated user attached to a request context.
type AuthUser struct {
	UserID       string
	SessionToken string
}

// ContextWithAuthUser returns a context containing authenticated user data.
func ContextWithAuthUser(ctx context.Context, user AuthUser) context.Context {
	return context.WithValue(ctx, authContextKey{}, user)
}

// AuthUserFromContext returns authenticated user data from context.
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

// RequireAuthSession returns both user and session token.
func RequireAuthSession(ctx context.Context) (AuthUser, error) {
	user, err := RequireAuthUser(ctx)
	if err != nil {
		return AuthUser{}, err
	}
	if user.SessionToken == "" {
		return AuthUser{}, domain.ErrUnauthenticated
	}
	return user, nil
}

// ContextWithRequestID returns a context containing the request ID.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestIDFromContext returns the request ID stored in context, if present.
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}
