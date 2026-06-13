// Package config loads, validates, and provides application configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultAppName   = "rpg-backend"
	defaultAppEnv    = "local"
	defaultLogLevel  = "info"
	defaultLogFormat = "text"

	defaultAPIHTTPAddr     = ":8080"
	defaultGameENetAddr    = ":7777"
	defaultGameHTTPAddr    = ":8081"
	defaultShutdownTimeout = 10 * time.Second

	defaultPostgresHost    = "localhost"
	defaultPostgresPort    = 5432
	defaultPostgresSSLMode = "disable"

	defaultPostgresMaxConns          = 10
	defaultPostgresMinConns          = 1
	defaultPostgresMaxConnLifetime   = time.Hour
	defaultPostgresMaxConnIdleTime   = 30 * time.Minute
	defaultPostgresHealthCheckPeriod = time.Minute

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

// Config holds all application configuration values.
type Config struct {
	AppName   string
	Env       string
	LogLevel  string
	LogFormat string

	APIHTTPAddr     string
	GameENetAddr    string
	GameHTTPAddr    string
	ShutdownTimeout time.Duration

	Postgres PostgresConfig
	Redis    RedisConfig
}

// PostgresConfig holds PostgreSQL connection configuration.
type PostgresConfig struct {
	// URL is derived from the individual fields and must not be set directly.
	URL string

	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string

	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

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

// Load reads .env (if present) then loads configuration from the process environment.
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

	// Reject direct overrides of derived fields.
	if v, ok := lookupTrimmed(source, "POSTGRES_URL"); ok && v != "" {
		return Config{}, errors.New("POSTGRES_URL must not be set directly; " +
			"use POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB, and POSTGRES_SSLMODE")
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

		APIHTTPAddr:     stringEnv(source, "API_HTTP_ADDR", defaultAPIHTTPAddr),
		GameENetAddr:    stringEnv(source, "GAME_ENET_ADDR", defaultGameENetAddr),
		GameHTTPAddr:    stringEnv(source, "GAME_HTTP_ADDR", defaultGameHTTPAddr),
		ShutdownTimeout: parsed.shutdownTimeout,

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

type parsedValues struct {
	shutdownTimeout time.Duration

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

func postgresURLFromConfig(p PostgresConfig) string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(p.User, p.Password),
		Host:   net.JoinHostPort(p.Host, strconv.Itoa(p.Port)),
		Path:   "/" + p.Database,
	}
	q := u.Query()
	q.Set("sslmode", p.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

func redisAddrFromConfig(r RedisConfig) string {
	return net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
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
	if strings.TrimSpace(c.GameENetAddr) == "" {
		errs = append(errs, errors.New("GAME_ENET_ADDR is required"))
	}
	if strings.TrimSpace(c.GameHTTPAddr) == "" {
		errs = append(errs, errors.New("GAME_HTTP_ADDR is required"))
	}
	if c.ShutdownTimeout <= 0 {
		errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT must be > 0, got %s", c.ShutdownTimeout))
	}

	if err := c.Postgres.validate(); err != nil {
		errs = append(errs, fmt.Errorf("postgres: %w", err))
	}
	if err := c.Redis.validate(); err != nil {
		errs = append(errs, fmt.Errorf("redis: %w", err))
	}

	return errors.Join(errs...)
}

func (p PostgresConfig) validate() error {
	var errs []error

	if strings.TrimSpace(p.Host) == "" {
		errs = append(errs, errors.New("POSTGRES_HOST is required"))
	}
	if p.Port <= 0 || p.Port > 65535 {
		errs = append(errs, fmt.Errorf("POSTGRES_PORT must be between 1 and 65535, got %d", p.Port))
	}
	if strings.TrimSpace(p.User) == "" {
		errs = append(errs, errors.New("POSTGRES_USER is required"))
	}
	if p.Password == "" {
		errs = append(errs, errors.New("POSTGRES_PASSWORD is required"))
	}
	if strings.TrimSpace(p.Database) == "" {
		errs = append(errs, errors.New("POSTGRES_DB is required"))
	}
	if !oneOf(p.SSLMode, "disable", "require", "verify-ca", "verify-full") {
		errs = append(errs, fmt.Errorf("invalid POSTGRES_SSLMODE %q: must be one of: disable, require, verify-ca, verify-full", p.SSLMode))
	}
	if p.MaxConns < 1 {
		errs = append(errs, fmt.Errorf("POSTGRES_MAX_CONNS must be >= 1, got %d", p.MaxConns))
	}
	if p.MinConns < 0 {
		errs = append(errs, fmt.Errorf("POSTGRES_MIN_CONNS must be >= 0, got %d", p.MinConns))
	}
	if p.MaxConns < p.MinConns {
		errs = append(errs, fmt.Errorf("POSTGRES_MAX_CONNS (%d) must be >= POSTGRES_MIN_CONNS (%d)", p.MaxConns, p.MinConns))
	}
	if p.MaxConnLifetime < 0 {
		errs = append(errs, fmt.Errorf("POSTGRES_MAX_CONN_LIFETIME must be >= 0, got %s", p.MaxConnLifetime))
	}
	if p.MaxConnIdleTime < 0 {
		errs = append(errs, fmt.Errorf("POSTGRES_MAX_CONN_IDLE_TIME must be >= 0, got %s", p.MaxConnIdleTime))
	}
	if p.HealthCheckPeriod <= 0 {
		errs = append(errs, fmt.Errorf("POSTGRES_HEALTH_CHECK_PERIOD must be > 0, got %s", p.HealthCheckPeriod))
	}

	return errors.Join(errs...)
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

// --- env helpers ---

// stringEnv returns the trimmed value of key, or def if key is absent or blank.
func stringEnv(source EnvSource, key, def string) string {
	if v, ok := lookupTrimmed(source, key); ok {
		return v
	}
	return def
}

// rawEnv returns the raw (untrimmed) value of key, or def if key is absent.
// Use for secrets such as passwords where whitespace is significant.
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

func oneOf(value string, allowed ...string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}
