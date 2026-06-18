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

// RedisGameServerStore implements GameServerStore using Redis for storage.
type RedisGameServerStore struct {
	client cache.Client
	keys   cache.KeyBuilder
	now    func() time.Time
}

// NewRedisGameServerStore creates a new RedisGameServerStore with the given Redis client and key builder.
func NewRedisGameServerStore(client cache.Client, keys cache.KeyBuilder) *RedisGameServerStore {
	return &RedisGameServerStore{
		client: client,
		keys:   keys,
		now:    time.Now,
	}
}

// ListGameServers returns a list of all game servers.
func (s *RedisGameServerStore) ListGameServers(ctx context.Context) ([]domain.GameServer, error) {
	indexKey, err := s.keys.ServersIndex()
	if err != nil {
		return nil, err
	}

	serverIDs, err := s.client.SMembers(ctx, indexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("list game server ids: %w", err)
	}

	servers := make([]domain.GameServer, 0, len(serverIDs))
	for _, serverID := range serverIDs {
		serverKey, err := s.keys.Server(serverID)
		if err != nil {
			return nil, err
		}

		raw, err := s.client.Get(ctx, serverKey).Bytes()
		if err != nil {
			if errors.Is(err, goredis.Nil) {
				continue
			}
			return nil, fmt.Errorf("get game server: %w", err)
		}

		var server domain.GameServer
		if err := json.Unmarshal(raw, &server); err != nil {
			continue
		}

		servers = append(servers, server)
	}

	return servers, nil
}
