package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	goredis "github.com/redis/go-redis/v9"
)

// RedisJoinTokenStore implements JoinTokenStore using Redis for storage.
type RedisJoinTokenStore struct {
	client cache.Client
	keys   cache.KeyBuilder
	now    func() time.Time
}

// NewRedisJoinTokenStore creates a new RedisJoinTokenStore with the given Redis client and key builder.
func NewRedisJoinTokenStore(client cache.Client, keys cache.KeyBuilder) *RedisJoinTokenStore {
	return &RedisJoinTokenStore{
		client: client,
		keys:   keys,
		now:    time.Now,
	}
}

// CreateJoinToken stores a join token in Redis with an expiration.
func (s *RedisJoinTokenStore) CreateJoinToken(ctx context.Context, token domain.JoinToken) error {
	key, err := s.keys.JoinToken(token.Token)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal join token: %w", err)
	}

	if err := s.client.Set(ctx, key, payload, cache.DefaultJoinTokenTTL).Err(); err != nil {
		return fmt.Errorf("create join token: %w", err)
	}

	return nil
}

// ConsumeJoinToken retrieves and deletes a join token atomically.
func (s *RedisJoinTokenStore) ConsumeJoinToken(ctx context.Context, token string) (domain.JoinToken, error) {
	key, err := s.keys.JoinToken(token)
	if err != nil {
		return domain.JoinToken{}, err
	}

	raw, err := s.client.GetDel(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return domain.JoinToken{}, domain.ErrUnauthenticated
		}
		return domain.JoinToken{}, fmt.Errorf("getdel join token: %w", err)
	}

	var parsed domain.JoinToken
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return domain.JoinToken{}, fmt.Errorf("unmarshal join token: %w", err)
	}

	return parsed, nil
}
