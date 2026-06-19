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
func (s *JoinTokenStore) CreateJoinToken(ctx context.Context, token domain.JoinToken) error {
	if s == nil || s.client == nil {
		return cache.ErrNilClient
	}

	tokenID, err := validate.RequiredID("join token", token.Token)
	if err != nil {
		return err
	}

	userID, err := validate.RequiredID("user ID", token.UserID)
	if err != nil {
		return err
	}

	characterID, err := validate.RequiredID("character ID", token.CharacterID)
	if err != nil {
		return err
	}

	serverID, err := validate.RequiredID("server ID", token.ServerID)
	if err != nil {
		return err
	}

	serverAddr, err := validate.RequiredID("server address", token.ServerAddr)
	if err != nil {
		return err
	}

	key, err := s.keys.JoinToken(tokenID)
	if err != nil {
		return fmt.Errorf("build join-token key: %w", err)
	}

	payload := joinTokenPayload{
		UserID:      userID,
		CharacterID: characterID,
		ServerID:    serverID,
		ServerAddr:  serverAddr,
		ExpiresAt:   time.Now().UTC().Add(cache.DefaultJoinTokenTTL),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal join token: %w", err)
	}

	if err := s.client.Set(ctx, key, data, cache.DefaultJoinTokenTTL).Err(); err != nil {
		return fmt.Errorf("create join token: %w", err)
	}

	return nil
}

// ConsumeJoinToken atomically reads and deletes a join token.
func (s *JoinTokenStore) ConsumeJoinToken(ctx context.Context, token string) (domain.JoinToken, error) {
	if s == nil || s.client == nil {
		return domain.JoinToken{}, cache.ErrNilClient
	}

	tokenID, err := validate.RequiredID("join token", token)
	if err != nil {
		return domain.JoinToken{}, err
	}

	key, err := s.keys.JoinToken(tokenID)
	if err != nil {
		return domain.JoinToken{}, fmt.Errorf("build join-token key: %w", err)
	}

	raw, err := s.client.GetDel(ctx, key).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return domain.JoinToken{}, domain.ErrUnauthenticated
		}

		return domain.JoinToken{}, fmt.Errorf("consume join token: %w", err)
	}

	var payload joinTokenPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return domain.JoinToken{}, fmt.Errorf("decode join token: %w", err)
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
