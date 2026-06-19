// Package service provides application services that orchestrate domain operations and enforce business rules.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/auth"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/validate"
)

const fakePasswordHash = "$argon2id$v=19$m=19456,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

// AuthService owns registration, login, and session revocation.
type AuthService struct {
	users    store.UserStore
	sessions store.SessionStore
}

// AuthResult is returned after successful authentication.
type AuthResult struct {
	UserID       string
	SessionToken string
}

// NewAuthService creates an AuthService.
func NewAuthService(users store.UserStore, sessions store.SessionStore) (*AuthService, error) {
	if users == nil {
		return nil, fmt.Errorf("user store is required: %w", domain.ErrInvalidArgument)
	}
	if sessions == nil {
		return nil, fmt.Errorf("session store is required: %w", domain.ErrInvalidArgument)
	}
	return &AuthService{
		users:    users,
		sessions: sessions,
	}, nil
}

// Register creates a new user account and immediately creates a session.
func (s *AuthService) Register(ctx context.Context, email, password string) (AuthResult, error) {
	if s == nil {
		return AuthResult{}, domain.ErrInternal
	}

	email, err := validate.Email(email)
	if err != nil {
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

// Login authenticates a user and creates a new session.
func (s *AuthService) Login(ctx context.Context, email, password string) (AuthResult, error) {
	if s == nil {
		return AuthResult{}, domain.ErrInternal
	}

	email, err := validate.Email(email)
	if err != nil {
		return AuthResult{}, err
	}

	user, err := s.users.GetUserByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		_ = auth.VerifyPassword(fakePasswordHash, password)
		return AuthResult{}, domain.ErrUnauthenticated
	}
	if err != nil {
		return AuthResult{}, err
	}

	if err := auth.VerifyPassword(user.PasswordHash, password); err != nil {
		return AuthResult{}, domain.ErrUnauthenticated
	}

	return s.createSession(ctx, user.ID)
}

// RevokeSession deletes a single session token.
func (s *AuthService) RevokeSession(ctx context.Context, sessionToken string) error {
	if s == nil {
		return domain.ErrInternal
	}

	sessionToken, err := validate.SessionToken(sessionToken)
	if err != nil {
		return err
	}

	return s.sessions.DeleteSession(ctx, sessionToken)
}

// RevokeUserSessions deletes all known sessions for a user.
func (s *AuthService) RevokeUserSessions(ctx context.Context, userID string) error {
	if s == nil {
		return domain.ErrInternal
	}

	userID, err := validate.UserID(userID)
	if err != nil {
		return err
	}

	return s.sessions.DeleteUserSessions(ctx, userID)
}

func (s *AuthService) createSession(ctx context.Context, userID string) (AuthResult, error) {
	userID, err := validate.UserID(userID)
	if err != nil {
		return AuthResult{}, err
	}

	sessionToken, err := auth.NewSessionToken()
	if err != nil {
		return AuthResult{}, fmt.Errorf("create session token: %w", err)
	}

	if err := s.sessions.CreateSession(ctx, sessionToken, userID); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		UserID:       userID,
		SessionToken: sessionToken,
	}, nil
}
