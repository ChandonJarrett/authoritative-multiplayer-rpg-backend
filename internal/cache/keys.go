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

// ErrEmptyPrefix is returned when a KeyBuilder with an empty prefix is used.
var ErrEmptyPrefix = errors.New("key prefix is empty")

// ErrInvalidSegment is returned when a key segment is blank, contains a colon, or contains whitespace.
var ErrInvalidSegment = errors.New("key segment is invalid")

// Default TTL values for Redis keys.
const (
	DefaultJoinTokenTTL     = 60 * time.Second
	DefaultServerTTL        = 10 * time.Second
	DefaultCharacterLockTTL = 20 * time.Second
	DefaultSessionTTL       = 2 * time.Hour
)

// KeyBuilder constructs namespaced Redis keys in the format {app}:{env}:{type}:{id}.
type KeyBuilder struct {
	prefix string
}

// NewKeyBuilder creates a KeyBuilder from app and env namespace segments.
func NewKeyBuilder(app, env string) (KeyBuilder, error) {
	if err := validateSegment(app); err != nil {
		return KeyBuilder{}, fmt.Errorf("app segment: %w", err)
	}
	if err := validateSegment(env); err != nil {
		return KeyBuilder{}, fmt.Errorf("env segment: %w", err)
	}
	return KeyBuilder{prefix: app + keySeparator + env}, nil
}

// NewKeyBuilderFromConfig creates a KeyBuilder from application configuration.
func NewKeyBuilderFromConfig(cfg config.Config) (KeyBuilder, error) {
	return NewKeyBuilder(cfg.AppName, cfg.Env)
}

// MustNewKeyBuilder creates a KeyBuilder and panics if the segments are invalid.
func MustNewKeyBuilder(app, env string) KeyBuilder {
	kb, err := NewKeyBuilder(app, env)
	if err != nil {
		panic(fmt.Sprintf("cache.MustNewKeyBuilder(%q, %q): %v", app, env, err))
	}
	return kb
}

// Prefix returns the {app}:{env} prefix shared by all keys from this builder.
func (k KeyBuilder) Prefix() string {
	return k.prefix
}

// JoinToken returns the Redis key for a join token.
func (k KeyBuilder) JoinToken(tokenID string) (string, error) {
	return k.build("join_token", tokenID)
}

// Session returns the Redis key for an authenticated session.
func (k KeyBuilder) Session(sessionID string) (string, error) {
	return k.build("session", sessionID)
}

// SessionPrefix returns the key prefix shared by all session keys, including the trailing separator.
// It is used for bulk operations that need to iterate over session keys.
func (k KeyBuilder) SessionPrefix() (string, error) {
	base, err := k.build("session")
	if err != nil {
		return "", err
	}
	return base + keySeparator, nil
}

// UserSessions returns the Redis key for the session set owned by a user.
func (k KeyBuilder) UserSessions(userID string) (string, error) {
	return k.build("user_sessions", userID)
}

// JoinTokenPrefix returns the key prefix shared by all join-token keys, including the trailing separator.
func (k KeyBuilder) JoinTokenPrefix() (string, error) {
	base, err := k.build("join_token")
	if err != nil {
		return "", err
	}
	return base + keySeparator, nil
}

// Server returns the Redis key for a game server registry entry.
func (k KeyBuilder) Server(serverID string) (string, error) {
	return k.build("server", serverID)
}

// ServerPrefix returns the key prefix shared by all game server keys, including the trailing separator.
func (k KeyBuilder) ServerPrefix() (string, error) {
	base, err := k.build("server")
	if err != nil {
		return "", err
	}
	return base + keySeparator, nil
}

// ServerSessions returns the Redis key for sessions active on a specific game server.
func (k KeyBuilder) ServerSessions(serverID string) (string, error) {
	return k.build("server_sessions", serverID)
}

// ServersIndex returns the Redis key for the global game server index.
func (k KeyBuilder) ServersIndex() (string, error) {
	return k.build("servers")
}

// CharacterLock returns the Redis key for a character lock.
func (k KeyBuilder) CharacterLock(characterID string) (string, error) {
	return k.build("character_lock", characterID)
}

// RateLimit returns the Redis key for rate-limit counters.
func (k KeyBuilder) RateLimit(parts ...string) (string, error) {
	all := append([]string{"rate_limit"}, parts...)
	return k.build(all...)
}

func (k KeyBuilder) build(parts ...string) (string, error) {
	if k.prefix == "" {
		return "", ErrEmptyPrefix
	}
	if len(parts) == 0 {
		return "", ErrInvalidSegment
	}
	for _, p := range parts {
		if err := validateSegment(p); err != nil {
			return "", fmt.Errorf("segment %q: %w", p, err)
		}
	}
	return k.prefix + keySeparator + strings.Join(parts, keySeparator), nil
}

// validateSegment checks that a key segment is non-empty and free of colons and whitespace.
func validateSegment(seg string) error {
	const invalidChars = " \t\r\n"
	if strings.TrimSpace(seg) == "" || strings.Contains(seg, keySeparator) || strings.ContainsAny(seg, invalidChars) {
		return ErrInvalidSegment
	}
	return nil
}
