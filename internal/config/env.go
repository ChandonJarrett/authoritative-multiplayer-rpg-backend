// Package config loads, validates, and provides application configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

// EnvSource provides environment variable lookup.
type EnvSource interface {
	LookupEnv(key string) (string, bool)
}

// OSEnv reads environment variables from the process environment.
type OSEnv struct{}

// LookupEnv returns a process environment variable.
func (OSEnv) LookupEnv(key string) (string, bool) { return os.LookupEnv(key) }

// MapEnv is an in-memory environment source for tests.
type MapEnv map[string]string

// LookupEnv returns a value from the map.
func (m MapEnv) LookupEnv(key string) (string, bool) {
	v, ok := m[key]
	return v, ok
}

// parsedValues holds all typed environment variable values after parsing.
type parsedValues struct {
	shutdownTimeout time.Duration

	authRateLimitWindow time.Duration
	authRateLimitBurst  int

	postgresPort              int
	postgresMaxConns          int32
	postgresMinConns          int32
	postgresMaxConnLifetime   time.Duration
	postgresMaxConnIdleTime   time.Duration
	postgresHealthCheckPeriod time.Duration

	redisPort         int
	redisDB           int
	redisDialTimeout  time.Duration
	redisReadTimeout  time.Duration
	redisWriteTimeout time.Duration
	redisPoolSize     int
	redisMinIdleConns int
}

// parseValues parses all typed environment variables, collecting every error
// rather than stopping at the first. Callers receive the full set of problems.
func parseValues(source EnvSource) (parsedValues, error) {
	var (
		v    parsedValues
		errs []error
		err  error
	)

	collect := func(e error) {
		if e != nil {
			errs = append(errs, e)
		}
	}

	v.shutdownTimeout, err = durationEnv(source, "SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	collect(err)

	v.authRateLimitWindow, err = durationEnv(source, "AUTH_RATE_LIMIT_WINDOW", defaultAuthRateLimitWindow)
	collect(err)
	v.authRateLimitBurst, err = intEnv(source, "AUTH_RATE_LIMIT_BURST", defaultAuthRateLimitBurst)
	collect(err)

	v.postgresPort, err = intEnv(source, "POSTGRES_PORT", defaultPostgresPort)
	collect(err)
	v.postgresMaxConns, err = int32Env(source, "POSTGRES_MAX_CONNS", defaultPostgresMaxConns)
	collect(err)
	v.postgresMinConns, err = int32Env(source, "POSTGRES_MIN_CONNS", defaultPostgresMinConns)
	collect(err)
	v.postgresMaxConnLifetime, err = durationEnv(source, "POSTGRES_MAX_CONN_LIFETIME", defaultPostgresMaxConnLifetime)
	collect(err)
	v.postgresMaxConnIdleTime, err = durationEnv(source, "POSTGRES_MAX_CONN_IDLE_TIME", defaultPostgresMaxConnIdleTime)
	collect(err)
	v.postgresHealthCheckPeriod, err = durationEnv(source, "POSTGRES_HEALTH_CHECK_PERIOD", defaultPostgresHealthCheckPeriod)
	collect(err)

	v.redisPort, err = intEnv(source, "REDIS_PORT", defaultRedisPort)
	collect(err)
	v.redisDB, err = intEnv(source, "REDIS_DB", defaultRedisDB)
	collect(err)
	v.redisDialTimeout, err = durationEnv(source, "REDIS_DIAL_TIMEOUT", defaultRedisDialTimeout)
	collect(err)
	v.redisReadTimeout, err = durationEnv(source, "REDIS_READ_TIMEOUT", defaultRedisReadTimeout)
	collect(err)
	v.redisWriteTimeout, err = durationEnv(source, "REDIS_WRITE_TIMEOUT", defaultRedisWriteTimeout)
	collect(err)
	v.redisPoolSize, err = intEnv(source, "REDIS_POOL_SIZE", defaultRedisPoolSize)
	collect(err)
	v.redisMinIdleConns, err = intEnv(source, "REDIS_MIN_IDLE_CONNS", defaultRedisMinIdleConns)
	collect(err)

	return v, errors.Join(errs...)
}

func stringEnv(source EnvSource, key, def string) string {
	if v, ok := lookupTrimmed(source, key); ok {
		return v
	}
	return def
}

func rawEnv(source EnvSource, key, def string) string {
	if v, ok := source.LookupEnv(key); ok {
		return v
	}
	return def
}

func lookupTrimmed(source EnvSource, key string) (string, bool) {
	v, ok := source.LookupEnv(key)
	return strings.TrimSpace(v), ok
}

func intEnv(source EnvSource, key string, def int) (int, error) {
	v, ok := lookupTrimmed(source, key)
	if !ok || v == "" {
		return def, nil
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	return n, nil
}

func int32Env(source EnvSource, key string, def int32) (int32, error) {
	n, err := intEnv(source, key, int(def))
	if err != nil {
		return 0, err
	}
	if n < math.MinInt32 || n > math.MaxInt32 {
		return 0, fmt.Errorf("%s must fit in int32, got %d", key, n)
	}

	return int32(n), nil
}

func durationEnv(source EnvSource, key string, def time.Duration) (time.Duration, error) {
	v, ok := lookupTrimmed(source, key)
	if !ok || v == "" {
		return def, nil
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid Go duration (e.g. 10s, 1m): %w", key, err)
	}

	return d, nil
}

func csvEnv(source EnvSource, key, def string) []string {
	raw := stringEnv(source, key, def)
	parts := strings.Split(raw, ",")

	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}

	return values
}

func oneOf(value string, allowed ...string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}

	return false
}
