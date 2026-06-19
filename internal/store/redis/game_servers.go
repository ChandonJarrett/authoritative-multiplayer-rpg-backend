package redis

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/validate"
	goredis "github.com/redis/go-redis/v9"
)

type gameServerPayload struct {
	ID        string    `json:"id"`
	Addr      string    `json:"addr"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GameServerStore stores ephemeral game server registry records in Redis.
type GameServerStore struct {
	client cache.Client
	keys   cache.KeyBuilder
}

// NewGameServerStore creates a Redis game-server store.
func NewGameServerStore(client cache.Client, keys cache.KeyBuilder) *GameServerStore {
	return &GameServerStore{
		client: client,
		keys:   keys,
	}
}

// RegisterGameServer registers or refreshes a game server heartbeat.
func (s *GameServerStore) RegisterGameServer(ctx context.Context, server domain.GameServer) error {
	if s == nil || s.client == nil {
		return domain.ErrUnavailable
	}

	serverID, err := validate.RequiredID("server ID", server.ID)
	if err != nil {
		return err
	}

	addr, err := validate.RequiredID("server address", server.Addr)
	if err != nil {
		return err
	}

	serverKey, err := s.keys.Server(serverID)
	if err != nil {
		return redisKeyError("build server key", err)
	}

	indexKey, err := s.keys.ServersIndex()
	if err != nil {
		return redisKeyError("build servers index key", err)
	}

	payload := gameServerPayload{
		ID:        serverID,
		Addr:      addr,
		ExpiresAt: time.Now().UTC().Add(cache.DefaultServerTTL),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return redisInternal("marshal game server", err)
	}

	if err := s.client.Set(ctx, serverKey, data, cache.DefaultServerTTL).Err(); err != nil {
		return redisUnavailable("register game server", err)
	}

	if err := s.client.SAdd(ctx, indexKey, serverID).Err(); err != nil {
		return redisUnavailable("index game server", err)
	}

	if err := s.client.Expire(ctx, indexKey, cache.DefaultServerTTL).Err(); err != nil {
		return redisUnavailable("expire game server index", err)
	}

	return nil
}

// DeregisterGameServer removes the server heartbeat key.
// The servers index has a TTL and will naturally expire if no servers refresh it.
func (s *GameServerStore) DeregisterGameServer(ctx context.Context, serverID string) error {
	if s == nil || s.client == nil {
		return domain.ErrUnavailable
	}

	serverID, err := validate.RequiredID("server ID", serverID)
	if err != nil {
		return err
	}

	serverKey, err := s.keys.Server(serverID)
	if err != nil {
		return redisKeyError("build server key", err)
	}

	if err := s.client.Del(ctx, serverKey).Err(); err != nil {
		return redisUnavailable("deregister game server", err)
	}

	return nil
}

// ListGameServers returns currently visible game servers.
func (s *GameServerStore) ListGameServers(ctx context.Context) ([]domain.GameServer, error) {
	if s == nil || s.client == nil {
		return nil, domain.ErrUnavailable
	}

	indexKey, err := s.keys.ServersIndex()
	if err != nil {
		return nil, redisKeyError("build servers index key", err)
	}

	serverIDs, err := s.client.SMembers(ctx, indexKey).Result()
	if err != nil {
		return nil, redisUnavailable("list game server IDs", err)
	}

	servers := make([]domain.GameServer, 0, len(serverIDs))
	for _, serverID := range serverIDs {
		serverID, err := validate.RequiredID("server ID", serverID)
		if err != nil {
			continue
		}

		server, err := s.getGameServer(ctx, serverID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			return nil, err
		}

		servers = append(servers, server)
	}

	sort.Slice(servers, func(i, j int) bool {
		return servers[i].ID < servers[j].ID
	})

	return servers, nil
}

// GetGameServerByID returns the game server details by ID.
func (s *GameServerStore) GetGameServerByID(ctx context.Context, serverID string) (domain.GameServer, error) {
	if s == nil || s.client == nil {
		return domain.GameServer{}, domain.ErrUnavailable
	}

	serverID, err := validate.RequiredID("server ID", serverID)
	if err != nil {
		return domain.GameServer{}, err
	}

	server, err := s.getGameServer(ctx, serverID)
	if err != nil {
		return domain.GameServer{}, err
	}

	return server, nil
}

func (s *GameServerStore) getGameServer(ctx context.Context, serverID string) (domain.GameServer, error) {
	serverKey, err := s.keys.Server(serverID)
	if err != nil {
		return domain.GameServer{}, redisKeyError("build server key", err)
	}

	raw, err := s.client.Get(ctx, serverKey).Result()
	if errors.Is(err, goredis.Nil) {
		return domain.GameServer{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.GameServer{}, redisUnavailable("get game server", err)
	}

	var payload gameServerPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return domain.GameServer{}, redisUnavailable("decode game server", err)
	}

	if _, err := validate.RequiredID("server ID", payload.ID); err != nil {
		return domain.GameServer{}, domain.ErrNotFound
	}
	if _, err := validate.RequiredID("server address", payload.Addr); err != nil {
		return domain.GameServer{}, domain.ErrNotFound
	}

	return domain.GameServer{
		ID:        payload.ID,
		Addr:      payload.Addr,
		ExpiresAt: payload.ExpiresAt,
	}, nil
}
