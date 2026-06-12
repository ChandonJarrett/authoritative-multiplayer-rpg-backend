// Package cache provides Redis connectivity, key construction, and cache-related utilities.
package cache

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/config"
)

const keySeparator = ":"

// ErrEmptyPrefix indicates the key prefix is empty.
var ErrEmptyPrefix = errors.New("key prefix is empty")

// ErrInvalidSegment indicates a key segment is invalid.
var ErrInvalidSegment = errors.New("key segment is invalid")

// Default TTL values for various Redis keys.
const (
	DefaultJoinTokenTTL     = 60 * time.Second
	DefaultServerTTL        = 10 * time.Second
	DefaultCharacterLockTTL = 20 * time.Second
	DefaultSessionTTL       = 2 * time.Hour
)

// KeyBuilder constructs Redis keys with a consistent namespace and validation.
type KeyBuilder struct {
	prefix string
}

// NewKeyBuilder creates a KeyBuilder from application configuration.
func NewKeyBuilder(cfg config.Config) (KeyBuilder, error) {
	if err := validateSegment(cfg.AppName); err != nil {
		return KeyBuilder{}, fmt.Errorf("app namespace: %w", err)
	}
	if err := validateSegment(cfg.Env); err != nil {
		return KeyBuilder{}, fmt.Errorf("env namespace: %w", err)
	}

	return KeyBuilder{prefix: cfg.AppName + keySeparator + cfg.Env}, nil
}

// Prefix returns the namespace prefix used for every generated key.
func (k KeyBuilder) Prefix() string {
	return k.prefix
}

// JoinToken constructs a Redis key for a join token.
func (k KeyBuilder) JoinToken(tokenID string) (string, error) {
	return k.build("join_token", tokenID)
}

// Session constructs a Redis key for an authenticated session.
func (k KeyBuilder) Session(sessionID string) (string, error) {
	return k.build("session", sessionID)
}

// UserSessions constructs a Redis key for sessions owned by a user.
func (k KeyBuilder) UserSessions(userID string) (string, error) {
	return k.build("user_sessions", userID)
}

// Server constructs a Redis key for a game server registry entry.
func (k KeyBuilder) Server(serverID string) (string, error) {
	return k.build("server", serverID)
}

// ServerSessions constructs a Redis key for sessions active on a server.
func (k KeyBuilder) ServerSessions(serverID string) (string, error) {
	return k.build("server_sessions", serverID)
}

// ServersIndex constructs a Redis key for the game server index.
func (k KeyBuilder) ServersIndex() (string, error) {
	return k.build("servers")
}

// CharacterLock constructs a Redis key for a character lock.
func (k KeyBuilder) CharacterLock(characterID string) (string, error) {
	return k.build("character_lock", characterID)
}

func (k KeyBuilder) build(parts ...string) (string, error) {
	if k.prefix == "" {
		return "", ErrEmptyPrefix
	}
	if len(parts) == 0 {
		return "", ErrInvalidSegment
	}

	for _, part := range parts {
		if err := validateSegment(part); err != nil {
			return "", fmt.Errorf("key segment %q: %w", part, err)
		}
	}

	return k.prefix + keySeparator + strings.Join(parts, keySeparator), nil
}

func validateSegment(segment string) error {
	if strings.TrimSpace(segment) == "" {
		return ErrInvalidSegment
	}
	if strings.Contains(segment, keySeparator) {
		return ErrInvalidSegment
	}
	if strings.ContainsAny(segment, " \t\r\n") {
		return ErrInvalidSegment
	}

	return nil
}
