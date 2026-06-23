// Package config loads, validates, and provides application configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultAppName   = "rpg-backend"
	defaultAppEnv    = "local"
	defaultLogLevel  = "info"
	defaultLogFormat = "text"

	defaultAPIHTTPAddr       = ":8080"
	defaultAPIAllowedOrigins = "http://localhost:3000,http://localhost:5173,http://127.0.0.1:3000,http://127.0.0.1:5173"
	defaultGameENetAddr      = ":7777"
	defaultGameHTTPAddr      = ":8081"

	defaultShutdownTimeout = 10 * time.Second

	defaultAuthRateLimitWindow = time.Minute
	defaultAuthRateLimitBurst  = 10
)

// Config holds all application configuration values.
type Config struct {
	AppName   string
	Env       string
	LogLevel  string
	LogFormat string

	APIHTTPAddr       string
	APIAllowedOrigins []string
	GameENetAddr      string
	GameHTTPAddr      string

	ShutdownTimeout time.Duration

	AuthRateLimitWindow time.Duration
	AuthRateLimitBurst  int

	Postgres PostgresConfig
	Redis    RedisConfig
}

// Load reads .env if present, then loads configuration from the process environment.
func Load() (Config, error) {
	_ = godotenv.Load()
	return LoadWithSource(OSEnv{})
}

// LoadWithSource loads configuration from the provided environment source.
// A nil source falls back to the process environment.
func LoadWithSource(source EnvSource) (Config, error) {
	if source == nil {
		source = OSEnv{}
	}

	if v, ok := lookupTrimmed(source, "POSTGRES_URL"); ok && v != "" {
		return Config{}, errors.New("POSTGRES_URL must not be set directly; use POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB, and POSTGRES_SSLMODE")
	}
	if v, ok := lookupTrimmed(source, "REDIS_ADDR"); ok && v != "" {
		return Config{}, errors.New("REDIS_ADDR must not be set directly; use REDIS_HOST and REDIS_PORT")
	}

	parsed, err := parseValues(source)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppName:   stringEnv(source, "APP_NAME", defaultAppName),
		Env:       stringEnv(source, "APP_ENV", defaultAppEnv),
		LogLevel:  stringEnv(source, "LOG_LEVEL", defaultLogLevel),
		LogFormat: stringEnv(source, "LOG_FORMAT", defaultLogFormat),

		APIHTTPAddr:       stringEnv(source, "API_HTTP_ADDR", defaultAPIHTTPAddr),
		APIAllowedOrigins: csvEnv(source, "API_ALLOWED_ORIGINS", defaultAPIAllowedOrigins),
		GameENetAddr:      stringEnv(source, "GAME_ENET_ADDR", defaultGameENetAddr),
		GameHTTPAddr:      stringEnv(source, "GAME_HTTP_ADDR", defaultGameHTTPAddr),

		ShutdownTimeout: parsed.shutdownTimeout,

		AuthRateLimitWindow: parsed.authRateLimitWindow,
		AuthRateLimitBurst:  parsed.authRateLimitBurst,

		Postgres: PostgresConfig{
			Host:     stringEnv(source, "POSTGRES_HOST", defaultPostgresHost),
			Port:     parsed.postgresPort,
			User:     stringEnv(source, "POSTGRES_USER", ""),
			Password: rawEnv(source, "POSTGRES_PASSWORD", ""),
			Database: stringEnv(source, "POSTGRES_DB", ""),
			SSLMode:  stringEnv(source, "POSTGRES_SSLMODE", defaultPostgresSSLMode),

			MaxConns:          parsed.postgresMaxConns,
			MinConns:          parsed.postgresMinConns,
			MaxConnLifetime:   parsed.postgresMaxConnLifetime,
			MaxConnIdleTime:   parsed.postgresMaxConnIdleTime,
			HealthCheckPeriod: parsed.postgresHealthCheckPeriod,
		},

		Redis: RedisConfig{
			Host:     stringEnv(source, "REDIS_HOST", defaultRedisHost),
			Port:     parsed.redisPort,
			Password: rawEnv(source, "REDIS_PASSWORD", defaultRedisPassword),
			DB:       parsed.redisDB,

			DialTimeout:  parsed.redisDialTimeout,
			ReadTimeout:  parsed.redisReadTimeout,
			WriteTimeout: parsed.redisWriteTimeout,
			PoolSize:     parsed.redisPoolSize,
			MinIdleConns: parsed.redisMinIdleConns,
		},
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	cfg.Postgres.URL = postgresURLFromConfig(cfg.Postgres)
	cfg.Redis.Addr = redisAddrFromConfig(cfg.Redis)

	return cfg, nil
}

// validate collects all top-level configuration errors so callers see the full
// picture in a single error rather than discovering problems one at a time.
func (c Config) validate() error {
	var errs []error

	if strings.TrimSpace(c.AppName) == "" {
		errs = append(errs, errors.New("APP_NAME is required"))
	}
	if !oneOf(c.Env, "local", "development", "testing", "staging", "production") {
		errs = append(errs, fmt.Errorf("invalid APP_ENV %q: must be one of: local, development, testing, staging, production", c.Env))
	}
	if !oneOf(c.LogLevel, "debug", "info", "warn", "error") {
		errs = append(errs, fmt.Errorf("invalid LOG_LEVEL %q: must be one of: debug, info, warn, error", c.LogLevel))
	}
	if !oneOf(c.LogFormat, "text", "json") {
		errs = append(errs, fmt.Errorf("invalid LOG_FORMAT %q: must be one of: text, json", c.LogFormat))
	}
	if strings.TrimSpace(c.APIHTTPAddr) == "" {
		errs = append(errs, errors.New("API_HTTP_ADDR is required"))
	}
	if len(c.APIAllowedOrigins) == 0 {
		errs = append(errs, errors.New("API_ALLOWED_ORIGINS must include at least one origin"))
	}
	if strings.TrimSpace(c.GameENetAddr) == "" {
		errs = append(errs, errors.New("GAME_ENET_ADDR is required"))
	}
	if strings.TrimSpace(c.GameHTTPAddr) == "" {
		errs = append(errs, errors.New("GAME_HTTP_ADDR is required"))
	}
	if c.ShutdownTimeout <= 0 {
		errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT must be > 0, got %s", c.ShutdownTimeout))
	}
	if c.AuthRateLimitWindow <= 0 {
		errs = append(errs, fmt.Errorf("AUTH_RATE_LIMIT_WINDOW must be > 0, got %s", c.AuthRateLimitWindow))
	}
	if c.AuthRateLimitBurst < 1 {
		errs = append(errs, fmt.Errorf("AUTH_RATE_LIMIT_BURST must be >= 1, got %d", c.AuthRateLimitBurst))
	}
	if err := c.Postgres.validate(); err != nil {
		errs = append(errs, fmt.Errorf("postgres: %w", err))
	}
	if err := c.Redis.validate(); err != nil {
		errs = append(errs, fmt.Errorf("redis: %w", err))
	}

	return errors.Join(errs...)
}
