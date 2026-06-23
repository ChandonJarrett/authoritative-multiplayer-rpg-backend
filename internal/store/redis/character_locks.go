package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/validate"
	goredis "github.com/redis/go-redis/v9"
)

const (
	renewCharacterLockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`

	releaseCharacterLockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
end
return 0
`
)

// CharacterLockStore manages short-lived Redis locks for active characters.
//
// The lock value is an owner ID, normally the game server ID or game session ID.
// Acquire uses SET NX with TTL.
// Renew and Release are owner-checked with Lua so one server cannot release another server's lock.
type CharacterLockStore struct {
	client  cache.Client
	keys    cache.KeyBuilder
	lockTTL time.Duration
}

// NewCharacterLockStore creates a Redis-backed character lock store.
func NewCharacterLockStore(client cache.Client, keys cache.KeyBuilder, lockTTL time.Duration) (*CharacterLockStore, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is required: %w", domain.ErrInvalidArgument)
	}
	if lockTTL <= 0 {
		return nil, fmt.Errorf("lock TTL must be > 0: %w", domain.ErrInvalidArgument)
	}
	return &CharacterLockStore{
		client:  client,
		keys:    keys,
		lockTTL: lockTTL,
	}, nil
}

// validateLockIDs validates and normalizes character and lock owner IDs.
func (s *CharacterLockStore) validateLockIDs(characterID, ownerID string) (string, string, error) {
	var err error
	characterID, err = validate.RequiredID("character ID", characterID)
	if err != nil {
		return "", "", err
	}
	ownerID, err = validate.RequiredID("lock owner ID", ownerID)
	if err != nil {
		return "", "", err
	}
	return characterID, ownerID, nil
}

// AcquireCharacterLock attempts to acquire a character lock for ownerID.
// It returns false when the character is already locked by another owner.
func (s *CharacterLockStore) AcquireCharacterLock(
	ctx context.Context,
	characterID string,
	ownerID string,
	ttl time.Duration,
) (bool, error) {
	if s == nil || s.client == nil {
		return false, domain.ErrUnavailable
	}

	var err error
	characterID, ownerID, err = s.validateLockIDs(characterID, ownerID)
	if err != nil {
		return false, err
	}

	ttl = s.effectiveTTL(ttl)

	key, err := s.keys.CharacterLock(characterID)
	if err != nil {
		return false, redisKeyError("build character lock key", err)
	}

	locked, err := s.client.SetNX(ctx, key, ownerID, ttl).Result()
	if err != nil {
		return false, redisUnavailable("acquire character lock", err)
	}

	return locked, nil
}

// RenewCharacterLock extends a lock only if ownerID still owns it.
func (s *CharacterLockStore) RenewCharacterLock(
	ctx context.Context,
	characterID string,
	ownerID string,
	ttl time.Duration,
) (bool, error) {
	if s == nil || s.client == nil {
		return false, domain.ErrUnavailable
	}

	var err error
	characterID, ownerID, err = s.validateLockIDs(characterID, ownerID)
	if err != nil {
		return false, err
	}

	ttl = s.effectiveTTL(ttl)

	key, err := s.keys.CharacterLock(characterID)
	if err != nil {
		return false, redisKeyError("build character lock key", err)
	}

	result, err := s.client.Eval(
		ctx,
		renewCharacterLockScript,
		[]string{key},
		ownerID,
		strconv.FormatInt(ttl.Milliseconds(), 10),
	).Result()
	if err != nil {
		return false, redisUnavailable("renew character lock", err)
	}

	return truthy(result), nil
}

// ReleaseCharacterLock releases a lock only if ownerID still owns it.
func (s *CharacterLockStore) ReleaseCharacterLock(
	ctx context.Context,
	characterID string,
	ownerID string,
) (bool, error) {
	if s == nil || s.client == nil {
		return false, domain.ErrUnavailable
	}

	var err error
	characterID, ownerID, err = s.validateLockIDs(characterID, ownerID)
	if err != nil {
		return false, err
	}

	key, err := s.keys.CharacterLock(characterID)
	if err != nil {
		return false, redisKeyError("build character lock key", err)
	}

	result, err := s.client.Eval(
		ctx,
		releaseCharacterLockScript,
		[]string{key},
		ownerID,
	).Result()
	if err != nil {
		return false, redisUnavailable("release character lock", err)
	}

	return truthy(result), nil
}

// GetCharacterLockOwner returns the current lock owner.
func (s *CharacterLockStore) GetCharacterLockOwner(ctx context.Context, characterID string) (string, error) {
	if s == nil || s.client == nil {
		return "", domain.ErrUnavailable
	}

	characterID, err := validate.RequiredID("character ID", characterID)
	if err != nil {
		return "", err
	}

	key, err := s.keys.CharacterLock(characterID)
	if err != nil {
		return "", redisKeyError("build character lock key", err)
	}

	ownerID, err := s.client.Get(ctx, key).Result()
	if errors.Is(err, goredis.Nil) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", redisUnavailable("get character lock owner", err)
	}

	ownerID, err = validate.RequiredID("lock owner ID", ownerID)
	if err != nil {
		return "", domain.ErrNotFound
	}

	return ownerID, nil
}

// effectiveTTL returns the requested TTL, falling back to the configured store default.
func (s *CharacterLockStore) effectiveTTL(requested time.Duration) time.Duration {
	if requested <= 0 {
		return s.lockTTL
	}
	return requested
}
