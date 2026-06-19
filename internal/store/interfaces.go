// Package store defines persistence contracts used by application services.
package store

import (
	"context"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

// UserStore is the durable user storage contract required by auth workflows.
type UserStore interface {
	CreateUser(ctx context.Context, user domain.User) error
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUserByID(ctx context.Context, userID string) (domain.User, error)
}

// SessionStore is the ephemeral session storage contract required by auth workflows.
type SessionStore interface {
	CreateSession(ctx context.Context, sessionID, userID string) error
	GetSessionUserID(ctx context.Context, sessionID string) (string, error)
	DeleteSession(ctx context.Context, sessionID string) error
	DeleteUserSessions(ctx context.Context, userID string) error
}

// CharacterStore is the durable character storage contract required by character and handoff workflows.
type CharacterStore interface {
	CreateCharacter(ctx context.Context, character domain.Character) error
	ListCharactersByUserID(ctx context.Context, userID string) ([]domain.Character, error)
	GetCharacterByID(ctx context.Context, characterID string) (domain.Character, error)
}

// JoinTokenStore is the ephemeral join-token storage contract required by game handoff workflows.
type JoinTokenStore interface {
	CreateJoinToken(ctx context.Context, token domain.JoinToken) error
	ConsumeJoinToken(ctx context.Context, token string) (domain.JoinToken, error)
}

// GameServerStore is the ephemeral game-server registry contract required by game handoff workflows.
type GameServerStore interface {
	RegisterGameServer(ctx context.Context, server domain.GameServer) error
	DeregisterGameServer(ctx context.Context, serverID string) error
	ListGameServers(ctx context.Context) ([]domain.GameServer, error)
	GetGameServerByID(ctx context.Context, serverID string) (domain.GameServer, error)
}
