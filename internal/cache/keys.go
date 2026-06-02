package cache

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const keySeparator = ":"

var (
	ErrEmptyPrefix    = errors.New("key prefix is empty")
	ErrInvalidSegment = errors.New("key segment is invalid")
)

const (
	DefaultJoinTokenTTL     = 60 * time.Second
	DefaultServerTTL        = 10 * time.Second
	DefaultCharacterLockTTL = 20 * time.Second
	DefaultSessionTTL       = 2 * time.Hour
)

type KeyBuilder struct {
	prefix string
}

func NewKeyBuilder(app, env string) (KeyBuilder, error) {
	if err := validateSegment(app); err != nil {
		return KeyBuilder{}, fmt.Errorf("app namespace: %w", err)
	}
	if err := validateSegment(env); err != nil {
		return KeyBuilder{}, fmt.Errorf("env namespace: %w", err)
	}

	return KeyBuilder{prefix: app + keySeparator + env}, nil
}

func (k KeyBuilder) Prefix() string {
	return k.prefix
}

func (k KeyBuilder) JoinToken(tokenID string) (string, error) {
	return k.build("join_token", tokenID)
}

func (k KeyBuilder) Session(sessionID string) (string, error) {
	return k.build("session", sessionID)
}

func (k KeyBuilder) UserSessions(userID string) (string, error) {
	return k.build("user_sessions", userID)
}

func (k KeyBuilder) Server(serverID string) (string, error) {
	return k.build("server", serverID)
}

func (k KeyBuilder) ServerSessions(serverID string) (string, error) {
	return k.build("server_sessions", serverID)
}

func (k KeyBuilder) ServersIndex() (string, error) {
	return k.build("servers")
}

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
