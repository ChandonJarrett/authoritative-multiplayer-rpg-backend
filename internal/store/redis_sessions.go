package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	goredis "github.com/redis/go-redis/v9"
)

// RedisSessionStore implements SessionStore using Redis for storage.
type RedisSessionStore struct {
	client RedisCommander
	keys   cache.KeyBuilder
}

// NewRedisSessionStore creates a new RedisSessionStore with the given Redis client and key builder.
func NewRedisSessionStore(client RedisCommander, keys cache.KeyBuilder) *RedisSessionStore {
	return &RedisSessionStore{
		client: client,
		keys:   keys,
	}
}

// CreateSession creates a new session for the given session ID and user ID. It stores the session in Redis with an expiration time.
func (s *RedisSessionStore) CreateSession(ctx context.Context, sessionID, userID string) error {
	sessionKey, err := s.keys.Session(sessionID)
	if err != nil {
		return err
	}

	userSessionsKey, err := s.keys.UserSessions(userID)
	if err != nil {
		return err
	}

	if err := s.client.Set(ctx, sessionKey, userID, cache.DefaultSessionTTL).Err(); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	if err := s.client.SAdd(ctx, userSessionsKey, sessionID).Err(); err != nil {
		return fmt.Errorf("add user session: %w", err)
	}

	if err := s.client.Expire(ctx, userSessionsKey, cache.DefaultSessionTTL).Err(); err != nil {
		return fmt.Errorf("expire user sessions: %w", err)
	}

	return nil
}

// GetSessionUserID retrieves the user ID associated with the given session ID.
func (s *RedisSessionStore) GetSessionUserID(ctx context.Context, sessionID string) (string, error) {
	sessionKey, err := s.keys.Session(sessionID)
	if err != nil {
		return "", err
	}

	userID, err := s.client.Get(ctx, sessionKey).Result()
	if err == nil {
		return userID, nil
	}
	if errors.Is(err, goredis.Nil) {
		return "", domain.ErrUnauthenticated
	}

	return "", fmt.Errorf("get session: %w", err)
}

// DeleteSession deletes the session with the given session ID from Redis.
func (s *RedisSessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	sessionKey, err := s.keys.Session(sessionID)
	if err != nil {
		return err
	}

	if err := s.client.Del(ctx, sessionKey).Err(); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}
