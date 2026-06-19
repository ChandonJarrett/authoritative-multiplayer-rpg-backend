// Package redis implements Redis stores.
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/validate"
	goredis "github.com/redis/go-redis/v9"
)

type sessionPayload struct {
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionStore stores authenticated sessions in Redis.
type SessionStore struct {
	client cache.Client
	keys   cache.KeyBuilder
}

// NewSessionStore creates a Redis session store.
func NewSessionStore(client cache.Client, keys cache.KeyBuilder) SessionStore {
	return SessionStore{
		client: client,
		keys:   keys,
	}
}

// CreateSession stores a session token with a TTL and indexes it by user.
func (s SessionStore) CreateSession(ctx context.Context, sessionID, userID string) error {
	if s.client == nil {
		return cache.ErrNilClient
	}

	sessionID, err := validate.RequiredID("session ID", sessionID)
	if err != nil {
		return err
	}

	userID, err = validate.RequiredID("user ID", userID)
	if err != nil {
		return err
	}

	sessionKey, err := s.keys.Session(sessionID)
	if err != nil {
		return fmt.Errorf("build session key: %w", err)
	}

	userSessionsKey, err := s.keys.UserSessions(userID)
	if err != nil {
		return fmt.Errorf("build user sessions key: %w", err)
	}

	payload := sessionPayload{
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(cache.DefaultSessionTTL),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	if err := s.client.Set(ctx, sessionKey, data, cache.DefaultSessionTTL).Err(); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	if err := s.client.SAdd(ctx, userSessionsKey, sessionID).Err(); err != nil {
		return fmt.Errorf("index session by user: %w", err)
	}

	if err := s.client.Expire(ctx, userSessionsKey, cache.DefaultSessionTTL).Err(); err != nil {
		return fmt.Errorf("expire user sessions index: %w", err)
	}

	return nil
}

// GetSessionUserID returns the user ID for a session token.
func (s SessionStore) GetSessionUserID(ctx context.Context, sessionID string) (string, error) {
	if s.client == nil {
		return "", cache.ErrNilClient
	}

	sessionID, err := validate.RequiredID("session ID", sessionID)
	if err != nil {
		return "", err
	}

	sessionKey, err := s.keys.Session(sessionID)
	if err != nil {
		return "", fmt.Errorf("build session key: %w", err)
	}

	raw, err := s.client.Get(ctx, sessionKey).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return "", domain.ErrUnauthenticated
		}

		return "", fmt.Errorf("get session: %w", err)
	}

	var payload sessionPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", fmt.Errorf("decode session: %w", err)
	}

	userID, err := validate.RequiredID("user ID", payload.UserID)
	if err != nil {
		return "", domain.ErrUnauthenticated
	}

	return userID, nil
}

// DeleteSession deletes a single session token.
//
// Note: without storing a reverse session-to-user cleanup transaction, this may leave
// old session IDs inside the user_sessions set. DeleteUserSessions handles bulk cleanup.
func (s SessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	if s.client == nil {
		return cache.ErrNilClient
	}

	sessionID, err := validate.RequiredID("session ID", sessionID)
	if err != nil {
		return err
	}

	sessionKey, err := s.keys.Session(sessionID)
	if err != nil {
		return fmt.Errorf("build session key: %w", err)
	}

	if err := s.client.Del(ctx, sessionKey).Err(); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

// DeleteUserSessions deletes all known session tokens for a user and removes the user session index.
func (s SessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	if s.client == nil {
		return cache.ErrNilClient
	}

	userID, err := validate.RequiredID("user ID", userID)
	if err != nil {
		return err
	}

	userSessionsKey, err := s.keys.UserSessions(userID)
	if err != nil {
		return fmt.Errorf("build user sessions key: %w", err)
	}

	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return fmt.Errorf("list user sessions: %w", err)
	}

	keysToDelete := make([]string, 0, len(sessionIDs)+1)
	for _, sessionID := range sessionIDs {
		if sessionID == "" {
			continue
		}

		sessionKey, err := s.keys.Session(sessionID)
		if err != nil {
			continue
		}

		keysToDelete = append(keysToDelete, sessionKey)
	}

	keysToDelete = append(keysToDelete, userSessionsKey)

	if len(keysToDelete) == 0 {
		return nil
	}

	if err := s.client.Del(ctx, keysToDelete...).Err(); err != nil {
		return fmt.Errorf("delete user sessions: %w", err)
	}

	return nil
}
