// Package config loads, validates, and provides application configuration from environment variables.
package config

import (
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

	defaultPostgresHost     = "localhost"
	defaultPostgresPort     = 5432
	defaultPostgresUser     = "postgres"
	defaultPostgresPassword = "postgres"
	defaultPostgresDB       = "rpg"
	defaultPostgresSSLMode  = "disable"

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
func (OSEnv) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// MapEnv is an in-memory environment source for tests.
type MapEnv map[string]string

// LookupEnv returns a value from the map environment.
func (m MapEnv) LookupEnv(key string) (string, bool) {
	value, ok := m[key]
	return value, ok
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

// Load reads .env, then loads configuration from process environment variables.
func Load() (Config, error) {
	_ = godotenv.Load()
	return LoadWithSource(OSEnv{})
}

// LoadWithSource loads configuration from the provided environment source.
func LoadWithSource(source EnvSource) (Config, error) {
	if source == nil {
		source = OSEnv{}
	}

	if value, ok := lookupTrimmed(source, "POSTGRES_URL"); ok && value != "" {
		return Config{}, fmt.Errorf("POSTGRES_URL must not be set directly; use POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB, and POSTGRES_SSLMODE")
	}
	if value, ok := lookupTrimmed(source, "REDIS_ADDR"); ok && value != "" {
		return Config{}, fmt.Errorf("REDIS_ADDR must not be set directly; use REDIS_HOST and REDIS_PORT")
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
			User:     stringEnv(source, "POSTGRES_USER", defaultPostgresUser),
			Password: rawEnv(source, "POSTGRES_PASSWORD", defaultPostgresPassword),
			Database: stringEnv(source, "POSTGRES_DB", defaultPostgresDB),
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

	cfg.Postgres.URL = postgresURLFromFields(cfg.Postgres)
	cfg.Redis.Addr = redisAddrFromFields(cfg.Redis)

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

func parseValues(source EnvSource) (parsedValues, error) {
	var firstErr error

	capture := func(err error) {
		if firstErr == nil && err != nil {
			firstErr = err
		}
	}

	values := parsedValues{}

	var err error

	values.shutdownTimeout, err = durationEnv(source, "SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	capture(err)

	values.postgresPort, err = intEnv(source, "POSTGRES_PORT", defaultPostgresPort)
	capture(err)

	values.postgresMaxConns, err = int32Env(source, "POSTGRES_MAX_CONNS", defaultPostgresMaxConns)
	capture(err)

	values.postgresMinConns, err = int32Env(source, "POSTGRES_MIN_CONNS", defaultPostgresMinConns)
	capture(err)

	values.postgresMaxConnLifetime, err = durationEnv(source, "POSTGRES_MAX_CONN_LIFETIME", defaultPostgresMaxConnLifetime)
	capture(err)

	values.postgresMaxConnIdleTime, err = durationEnv(source, "POSTGRES_MAX_CONN_IDLE_TIME", defaultPostgresMaxConnIdleTime)
	capture(err)

	values.postgresHealthCheckPeriod, err = durationEnv(source, "POSTGRES_HEALTH_CHECK_PERIOD", defaultPostgresHealthCheckPeriod)
	capture(err)

	values.redisPort, err = intEnv(source, "REDIS_PORT", defaultRedisPort)
	capture(err)

	values.redisDB, err = intEnv(source, "REDIS_DB", defaultRedisDB)
	capture(err)

	values.redisDialTimeout, err = durationEnv(source, "REDIS_DIAL_TIMEOUT", defaultRedisDialTimeout)
	capture(err)

	values.redisReadTimeout, err = durationEnv(source, "REDIS_READ_TIMEOUT", defaultRedisReadTimeout)
	capture(err)

	values.redisWriteTimeout, err = durationEnv(source, "REDIS_WRITE_TIMEOUT", defaultRedisWriteTimeout)
	capture(err)

	values.redisPoolSize, err = intEnv(source, "REDIS_POOL_SIZE", defaultRedisPoolSize)
	capture(err)

	values.redisMinIdleConns, err = intEnv(source, "REDIS_MIN_IDLE_CONNS", defaultRedisMinIdleConns)
	capture(err)

	if firstErr != nil {
		return parsedValues{}, firstErr
	}

	return values, nil
}

func postgresURLFromFields(p PostgresConfig) string {
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

func redisAddrFromFields(r RedisConfig) string {
	return net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
}

func (c Config) validate() error {
	if strings.TrimSpace(c.AppName) == "" {
		return fmt.Errorf("APP_NAME is required")
	}
	if !oneOf(c.Env, "local", "development", "testing", "staging", "production") {
		return fmt.Errorf("invalid APP_ENV: %q", c.Env)
	}
	if !oneOf(c.LogLevel, "debug", "info", "warn", "error") {
		return fmt.Errorf("invalid LOG_LEVEL: %q", c.LogLevel)
	}
	if !oneOf(c.LogFormat, "text", "json") {
		return fmt.Errorf("invalid LOG_FORMAT: %q", c.LogFormat)
	}
	if strings.TrimSpace(c.APIHTTPAddr) == "" {
		return fmt.Errorf("API_HTTP_ADDR is required")
	}
	if strings.TrimSpace(c.GameENetAddr) == "" {
		return fmt.Errorf("GAME_ENET_ADDR is required")
	}
	if strings.TrimSpace(c.GameHTTPAddr) == "" {
		return fmt.Errorf("GAME_HTTP_ADDR is required")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT must be > 0, got %s", c.ShutdownTimeout)
	}
	if err := c.Postgres.validate(); err != nil {
		return fmt.Errorf("postgres config: %w", err)
	}
	if err := c.Redis.validate(); err != nil {
		return fmt.Errorf("redis config: %w", err)
	}

	return nil
}

func (p PostgresConfig) validate() error {
	if strings.TrimSpace(p.Host) == "" {
		return fmt.Errorf("POSTGRES_HOST is required")
	}
	if p.Port <= 0 || p.Port > 65535 {
		return fmt.Errorf("POSTGRES_PORT must be between 1 and 65535, got %d", p.Port)
	}
	if strings.TrimSpace(p.User) == "" {
		return fmt.Errorf("POSTGRES_USER is required")
	}
	if p.Password == "" {
		return fmt.Errorf("POSTGRES_PASSWORD is required")
	}
	if strings.TrimSpace(p.Database) == "" {
		return fmt.Errorf("POSTGRES_DB is required")
	}
	if !oneOf(p.SSLMode, "disable", "require", "verify-ca", "verify-full") {
		return fmt.Errorf("invalid POSTGRES_SSLMODE: %q", p.SSLMode)
	}
	if p.MaxConns < 1 {
		return fmt.Errorf("POSTGRES_MAX_CONNS must be >= 1, got %d", p.MaxConns)
	}
	if p.MinConns < 0 {
		return fmt.Errorf("POSTGRES_MIN_CONNS must be >= 0, got %d", p.MinConns)
	}
	if p.MaxConns < p.MinConns {
		return fmt.Errorf("POSTGRES_MAX_CONNS (%d) must be >= POSTGRES_MIN_CONNS (%d)", p.MaxConns, p.MinConns)
	}
	if p.MaxConnLifetime < 0 {
		return fmt.Errorf("POSTGRES_MAX_CONN_LIFETIME must be >= 0, got %s", p.MaxConnLifetime)
	}
	if p.MaxConnIdleTime < 0 {
		return fmt.Errorf("POSTGRES_MAX_CONN_IDLE_TIME must be >= 0, got %s", p.MaxConnIdleTime)
	}
	if p.HealthCheckPeriod <= 0 {
		return fmt.Errorf("POSTGRES_HEALTH_CHECK_PERIOD must be > 0, got %s", p.HealthCheckPeriod)
	}

	return nil
}

func (r RedisConfig) validate() error {
	if strings.TrimSpace(r.Host) == "" {
		return fmt.Errorf("REDIS_HOST is required")
	}
	if r.Port <= 0 || r.Port > 65535 {
		return fmt.Errorf("REDIS_PORT must be between 1 and 65535, got %d", r.Port)
	}
	if r.DB < 0 {
		return fmt.Errorf("REDIS_DB must be >= 0, got %d", r.DB)
	}
	if r.PoolSize < 1 {
		return fmt.Errorf("REDIS_POOL_SIZE must be >= 1, got %d", r.PoolSize)
	}
	if r.MinIdleConns < 0 {
		return fmt.Errorf("REDIS_MIN_IDLE_CONNS must be >= 0, got %d", r.MinIdleConns)
	}
	if r.MinIdleConns > r.PoolSize {
		return fmt.Errorf("REDIS_MIN_IDLE_CONNS (%d) must be <= REDIS_POOL_SIZE (%d)", r.MinIdleConns, r.PoolSize)
	}
	if r.DialTimeout <= 0 {
		return fmt.Errorf("REDIS_DIAL_TIMEOUT must be > 0, got %s", r.DialTimeout)
	}
	if r.ReadTimeout <= 0 {
		return fmt.Errorf("REDIS_READ_TIMEOUT must be > 0, got %s", r.ReadTimeout)
	}
	if r.WriteTimeout <= 0 {
		return fmt.Errorf("REDIS_WRITE_TIMEOUT must be > 0, got %s", r.WriteTimeout)
	}

	return nil
}

func stringEnv(source EnvSource, key, def string) string {
	value, ok := lookupTrimmed(source, key)
	if ok && value != "" {
		return value
	}

	return def
}

func rawEnv(source EnvSource, key, def string) string {
	value, ok := source.LookupEnv(key)
	if ok {
		return value
	}

	return def
}

func lookupTrimmed(source EnvSource, key string) (string, bool) {
	value, ok := source.LookupEnv(key)
	return strings.TrimSpace(value), ok
}

func intEnv(source EnvSource, key string, def int) (int, error) {
	value, ok := lookupTrimmed(source, key)
	if !ok || value == "" {
		return def, nil
	}

	n, err := strconv.Atoi(value)
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
	value, ok := lookupTrimmed(source, key)
	if !ok || value == "" {
		return def, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration: %w", key, err)
	}

	return duration, nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}

	return false
}
