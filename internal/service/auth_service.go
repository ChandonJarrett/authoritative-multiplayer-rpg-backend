// Package service contains the business logic services.
package service

import (
	"context"
	"fmt"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/auth"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/validate"
	"github.com/google/uuid"
)

// AuthService provides methods for user registration, login, and session management.
type AuthService struct {
	users    store.UserStore
	sessions store.SessionStore
}

// NewAuthService creates a new AuthService with the given user store and session store.
func NewAuthService(users store.UserStore, sessions store.SessionStore) *AuthService {
	return &AuthService{
		users:    users,
		sessions: sessions,
	}
}

// AuthResult represents the result of an authentication operation.
type AuthResult struct {
	UserID       string
	SessionToken string
}

// Register creates a new user account and logs them in.
func (s *AuthService) Register(ctx context.Context, email, password string) (AuthResult, error) {
	email, err := validate.Email(email)
	if err != nil {
		return AuthResult{}, err
	}

	if err := auth.ValidatePassword(password); err != nil {
		return AuthResult{}, err
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	user := domain.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: passwordHash,
	}

	if err := s.users.CreateUser(ctx, user); err != nil {
		return AuthResult{}, err
	}

	return s.createSession(ctx, user.ID)
}

// Login authenticates a user and creates a new session for them.
func (s *AuthService) Login(ctx context.Context, email, password string) (AuthResult, error) {
	email, err := validate.Email(email)
	if err != nil {
		return AuthResult{}, err
	}

	user, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		return AuthResult{}, domain.ErrUnauthenticated
	}

	if err := auth.VerifyPassword(user.PasswordHash, password); err != nil {
		return AuthResult{}, domain.ErrUnauthenticated
	}

	return s.createSession(ctx, user.ID)
}

func (s *AuthService) createSession(ctx context.Context, userID string) (AuthResult, error) {
	sessionToken, err := auth.NewSessionToken()
	if err != nil {
		return AuthResult{}, err
	}

	if err := s.sessions.CreateSession(ctx, sessionToken, userID); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		UserID:       userID,
		SessionToken: sessionToken,
	}, nil
}
