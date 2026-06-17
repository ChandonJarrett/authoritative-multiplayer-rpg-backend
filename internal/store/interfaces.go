// Package store defines interfaces for database operations.
package store

import (
	"context"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

// UserStore defines the interface for user-related database operations.
type UserStore interface {
	CreateUser(ctx context.Context, user domain.User) error
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUserByID(ctx context.Context, userID string) (domain.User, error)
}

// CharacterStore defines the interface for character-related database operations.
type CharacterStore interface {
	CreateCharacter(ctx context.Context, character domain.Character) error
	ListCharactersByUserID(ctx context.Context, userID string) ([]domain.Character, error)
	GetCharacterByID(ctx context.Context, characterID string) (domain.Character, error)
}

// SessionStore defines the interface for session-related database operations.
type SessionStore interface {
	CreateSession(ctx context.Context, sessionID, userID string) error
	GetSessionUserID(ctx context.Context, sessionID string) (string, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

// JoinTokenStore defines the interface for join token-related database operations.
type JoinTokenStore interface {
	CreateJoinToken(ctx context.Context, token domain.JoinToken) error
	ConsumeJoinToken(ctx context.Context, token string) (domain.JoinToken, error)
}

// GameServerStore defines the interface for game server-related database operations.
type GameServerStore interface {
	ListGameServers(ctx context.Context) ([]domain.GameServer, error)
}
