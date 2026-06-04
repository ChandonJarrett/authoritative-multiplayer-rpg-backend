package cache

import (
	"errors"
	"fmt"
	"strings"
	"time"
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

// KeyBuilder constructs Redis keys with a consistent format and validation.
type KeyBuilder struct {
	prefix string
}

// NewKeyBuilder creates a new KeyBuilder with the given application and environment namespaces.
func NewKeyBuilder(app, env string) (KeyBuilder, error) {
	if err := validateSegment(app); err != nil {
		return KeyBuilder{}, fmt.Errorf("app namespace: %w", err)
	}
	if err := validateSegment(env); err != nil {
		return KeyBuilder{}, fmt.Errorf("env namespace: %w", err)
	}

	return KeyBuilder{prefix: app + keySeparator + env}, nil
}

// Prefix returns the prefix used for all keys built by this KeyBuilder.
func (k KeyBuilder) Prefix() string {
	return k.prefix
}

// JoinToken constructs a Redis key for a join token with the given token ID.
func (k KeyBuilder) JoinToken(tokenID string) (string, error) {
	return k.build("join_token", tokenID)
}

// Session constructs a Redis key for a session with the given session ID.
func (k KeyBuilder) Session(sessionID string) (string, error) {
	return k.build("session", sessionID)
}

// UserSessions constructs a Redis key for all sessions associated with a user ID.
func (k KeyBuilder) UserSessions(userID string) (string, error) {
	return k.build("user_sessions", userID)
}

// Server constructs a Redis key for a server with the given server ID.
func (k KeyBuilder) Server(serverID string) (string, error) {
	return k.build("server", serverID)
}

// ServerSessions constructs a Redis key for all sessions associated with a server ID.
func (k KeyBuilder) ServerSessions(serverID string) (string, error) {
	return k.build("server_sessions", serverID)
}

// ServersIndex constructs a Redis key for the index of all servers.
func (k KeyBuilder) ServersIndex() (string, error) {
	return k.build("servers")
}

// CharacterLock constructs a Redis key for a character lock with the given character ID.
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
