package redis

import (
	"errors"
	"fmt"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

func redisUnavailable(operation string, err error) error {
	if err == nil {
		return domain.ErrUnavailable
	}
	return fmt.Errorf("%s: %w: %w", operation, domain.ErrUnavailable, err)
}

func redisInternal(operation string, err error) error {
	if err == nil {
		return domain.ErrInternal
	}
	return fmt.Errorf("%s: %w: %w", operation, domain.ErrInternal, err)
}

func redisKeyError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, cache.ErrInvalidSegment) {
		return fmt.Errorf("%s: %w", operation, domain.ErrInvalidArgument)
	}
	if errors.Is(err, cache.ErrEmptyPrefix) {
		return fmt.Errorf("%s: %w", operation, domain.ErrUnavailable)
	}
	return redisUnavailable(operation, err)
}

// truthy interprets a Redis Lua script result as a boolean.
// Redis Lua returns integer 0 for nil/false and integer 1 for true,
// or a string that some commands may produce.
func truthy(value interface{}) bool {
	switch v := value.(type) {
	case int64:
		return v > 0
	case int:
		return v > 0
	case string:
		return v != "" && v != "0"
	default:
		return false
	}
}
