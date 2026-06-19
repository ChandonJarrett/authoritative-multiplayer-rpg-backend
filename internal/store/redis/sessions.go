// Package redis implements Redis stores.
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/validate"
	goredis "github.com/redis/go-redis/v9"
)

const createSessionScript = `
redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2])
redis.call("SADD", KEYS[2], ARGV[3])
redis.call("PEXPIRE", KEYS[2], ARGV[2])
return 1
`

const deleteSessionScript = `
local user_id = redis.call("GET", KEYS[1])
if not user_id then
    return 0
end

redis.call("DEL", KEYS[1])
redis.call("SREM", KEYS[2], ARGV[1])
return 1
`

const deleteUserSessionsScript = `
local session_ids = redis.call("SMEMBERS", KEYS[1])
for _, session_id in ipairs(session_ids) do
    redis.call("DEL", ARGV[1] .. session_id)
end

redis.call("DEL", KEYS[1])
return #session_ids
`

// SessionStore stores authenticated sessions in Redis.
//
// Session keys store the owning user ID directly.
// User-session set keys track all session IDs for bulk revocation.
type SessionStore struct {
	client cache.Client
	keys   cache.KeyBuilder
	ttl    time.Duration
}

// NewSessionStore creates a Redis session store.
func NewSessionStore(client cache.Client, keys cache.KeyBuilder) *SessionStore {
	return &SessionStore{
		client: client,
		keys:   keys,
		ttl:    cache.DefaultSessionTTL,
	}
}

// CreateSession creates a session and indexes it by user in one Redis script.
func (s *SessionStore) CreateSession(ctx context.Context, sessionID, userID string) error {
	if s == nil || s.client == nil {
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

	ttl := s.ttl
	if ttl <= 0 {
		ttl = cache.DefaultSessionTTL
	}

	if err := s.client.Eval(
		ctx,
		createSessionScript,
		[]string{sessionKey, userSessionsKey},
		userID,
		ttl.Milliseconds(),
		sessionID,
	).Err(); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

// GetSessionUserID returns the user ID that owns a session.
func (s *SessionStore) GetSessionUserID(ctx context.Context, sessionID string) (string, error) {
	if s == nil || s.client == nil {
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

	userID, err := s.client.Get(ctx, sessionKey).Result()
	if errors.Is(err, goredis.Nil) {
		return "", domain.ErrUnauthenticated
	}
	if err != nil {
		return "", fmt.Errorf("get session: %w", err)
	}

	userID, err = validate.RequiredID("user ID", userID)
	if err != nil {
		return "", domain.ErrUnauthenticated
	}

	return userID, nil
}

// DeleteSession deletes one session and removes it from the owning user's session index.
func (s *SessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	if s == nil || s.client == nil {
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

	userID, err := s.GetSessionUserID(ctx, sessionID)
	if errors.Is(err, domain.ErrUnauthenticated) {
		return nil
	}
	if err != nil {
		return err
	}

	userSessionsKey, err := s.keys.UserSessions(userID)
	if err != nil {
		return fmt.Errorf("build user sessions key: %w", err)
	}

	if err := s.client.Eval(
		ctx,
		deleteSessionScript,
		[]string{sessionKey, userSessionsKey},
		sessionID,
	).Err(); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

// DeleteUserSessions deletes every known session for a user.
func (s *SessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	if s == nil || s.client == nil {
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

	sessionPrefix := s.keys.Prefix() + ":session:"

	if err := s.client.Eval(
		ctx,
		deleteUserSessionsScript,
		[]string{userSessionsKey},
		sessionPrefix,
	).Err(); err != nil {
		return fmt.Errorf("delete user sessions: %w", err)
	}

	return nil
}
