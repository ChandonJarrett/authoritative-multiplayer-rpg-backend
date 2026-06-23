package config

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRedisHost     = "localhost"
	defaultRedisPort     = 6379
	defaultRedisPassword = ""
	defaultRedisDB       = 0

	defaultRedisDialTimeout  = 5 * time.Second
	defaultRedisReadTimeout  = 3 * time.Second
	defaultRedisWriteTimeout = 3 * time.Second
	defaultRedisPoolSize     = 10
	defaultRedisMinIdleConns = 1
)

// RedisConfig holds Redis connection configuration.
type RedisConfig struct {
	// Addr is derived from Host and Port and must not be set directly.
	Addr string

	Host     string
	Port     int
	Password string
	DB       int

	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
}

func (r RedisConfig) validate() error {
	var errs []error

	if strings.TrimSpace(r.Host) == "" {
		errs = append(errs, errors.New("REDIS_HOST is required"))
	}
	if r.Port <= 0 || r.Port > 65535 {
		errs = append(errs, fmt.Errorf("REDIS_PORT must be between 1 and 65535, got %d", r.Port))
	}
	if r.DB < 0 {
		errs = append(errs, fmt.Errorf("REDIS_DB must be >= 0, got %d", r.DB))
	}
	if r.PoolSize < 1 {
		errs = append(errs, fmt.Errorf("REDIS_POOL_SIZE must be >= 1, got %d", r.PoolSize))
	}
	if r.MinIdleConns < 0 {
		errs = append(errs, fmt.Errorf("REDIS_MIN_IDLE_CONNS must be >= 0, got %d", r.MinIdleConns))
	}
	if r.MinIdleConns > r.PoolSize {
		errs = append(errs, fmt.Errorf("REDIS_MIN_IDLE_CONNS (%d) must be <= REDIS_POOL_SIZE (%d)", r.MinIdleConns, r.PoolSize))
	}
	if r.DialTimeout <= 0 {
		errs = append(errs, fmt.Errorf("REDIS_DIAL_TIMEOUT must be > 0, got %s", r.DialTimeout))
	}
	if r.ReadTimeout <= 0 {
		errs = append(errs, fmt.Errorf("REDIS_READ_TIMEOUT must be > 0, got %s", r.ReadTimeout))
	}
	if r.WriteTimeout <= 0 {
		errs = append(errs, fmt.Errorf("REDIS_WRITE_TIMEOUT must be > 0, got %s", r.WriteTimeout))
	}

	return errors.Join(errs...)
}

func redisAddrFromConfig(r RedisConfig) string {
	return net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
}
