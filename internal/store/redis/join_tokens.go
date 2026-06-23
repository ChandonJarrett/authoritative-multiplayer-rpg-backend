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

type joinTokenPayload struct {
	UserID      string    `json:"user_id"`
	CharacterID string    `json:"character_id"`
	ServerID    string    `json:"server_id"`
	ServerAddr  string    `json:"server_addr"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// JoinTokenStore stores short-lived game handoff tokens in Redis.
type JoinTokenStore struct {
	client cache.Client
	keys   cache.KeyBuilder
}

// NewJoinTokenStore creates a Redis join-token store.
func NewJoinTokenStore(client cache.Client, keys cache.KeyBuilder) *JoinTokenStore {
	return &JoinTokenStore{
		client: client,
		keys:   keys,
	}
}

// CreateJoinToken stores a short-lived join token.
// Callers must ensure all Token, UserID, CharacterID, ServerID, and ServerAddr
// fields are already validated and non-empty.
func (s *JoinTokenStore) CreateJoinToken(ctx context.Context, token domain.JoinToken) error {
	if s == nil || s.client == nil {
		return domain.ErrUnavailable
	}

	key, err := s.keys.JoinToken(token.Token)
	if err != nil {
		return redisKeyError("build join-token key", err)
	}

	ttl := time.Until(token.ExpiresAt)
	if token.ExpiresAt.IsZero() {
		return fmt.Errorf("join token expiry is required: %w", domain.ErrInvalidArgument)
	}
	if ttl <= 0 {
		return fmt.Errorf("join token already expired: %w", domain.ErrInvalidArgument)
	}

	payload := joinTokenPayload{
		UserID:      token.UserID,
		CharacterID: token.CharacterID,
		ServerID:    token.ServerID,
		ServerAddr:  token.ServerAddr,
		ExpiresAt:   token.ExpiresAt.UTC(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return redisInternal("marshal join token", err)
	}

	if err := s.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return redisUnavailable("create join token", err)
	}

	return nil
}

// ConsumeJoinToken atomically returns and deletes a join token.
func (s *JoinTokenStore) ConsumeJoinToken(ctx context.Context, token string) (domain.JoinToken, error) {
	if s == nil || s.client == nil {
		return domain.JoinToken{}, domain.ErrUnavailable
	}

	tokenID, err := validate.RequiredID("join token", token)
	if err != nil {
		return domain.JoinToken{}, err
	}

	key, err := s.keys.JoinToken(tokenID)
	if err != nil {
		return domain.JoinToken{}, redisKeyError("build join-token key", err)
	}

	raw, err := s.client.GetDel(ctx, key).Result()
	if errors.Is(err, goredis.Nil) {
		return domain.JoinToken{}, domain.ErrUnauthenticated
	}
	if err != nil {
		return domain.JoinToken{}, redisUnavailable("consume join token", err)
	}

	var payload joinTokenPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return domain.JoinToken{}, domain.ErrUnauthenticated
	}

	if _, err := validate.RequiredID("user ID", payload.UserID); err != nil {
		return domain.JoinToken{}, domain.ErrUnauthenticated
	}
	if _, err := validate.RequiredID("character ID", payload.CharacterID); err != nil {
		return domain.JoinToken{}, domain.ErrUnauthenticated
	}
	if _, err := validate.RequiredID("server ID", payload.ServerID); err != nil {
		return domain.JoinToken{}, domain.ErrUnauthenticated
	}
	if _, err := validate.RequiredID("server address", payload.ServerAddr); err != nil {
		return domain.JoinToken{}, domain.ErrUnauthenticated
	}

	return domain.JoinToken{
		Token:       tokenID,
		UserID:      payload.UserID,
		CharacterID: payload.CharacterID,
		ServerID:    payload.ServerID,
		ServerAddr:  payload.ServerAddr,
		ExpiresAt:   payload.ExpiresAt,
	}, nil
}
