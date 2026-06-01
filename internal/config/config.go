package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

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

func (p PostgresConfig) DSN() string {
	if p.URL != "" {
		return p.URL
	}

	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(p.User, p.Password),
		Host:   fmt.Sprintf("%s:%d", p.Host, p.Port),
		Path:   "/" + p.Database,
	}

	q := u.Query()
	q.Set("sslmode", p.SSLMode)
	u.RawQuery = q.Encode()

	return u.String()
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int

	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
}

func Load() (Config, error) {
	_ = godotenv.Load()

	var err error

	captureErr := func(e error) {
		if err == nil && e != nil {
			err = e
		}
	}

	shutdownTimeout, e := durEnv("SHUTDOWN_TIMEOUT", 10*time.Second)
	captureErr(e)

	postgresPort, e := intEnv("POSTGRES_PORT", 5432)
	captureErr(e)

	postgresMaxConns, e := int32Env("POSTGRES_MAX_CONNS", 10)
	captureErr(e)

	postgresMinConns, e := int32Env("POSTGRES_MIN_CONNS", 1)
	captureErr(e)

	postgresMaxConnLifetime, e := durEnv("POSTGRES_MAX_CONN_LIFETIME", time.Hour)
	captureErr(e)

	postgresMaxConnIdleTime, e := durEnv("POSTGRES_MAX_CONN_IDLE_TIME", 30*time.Minute)
	captureErr(e)

	postgresHealthCheckPeriod, e := durEnv("POSTGRES_HEALTH_CHECK_PERIOD", time.Minute)
	captureErr(e)

	redisDB, e := intEnv("REDIS_DB", 0)
	captureErr(e)

	redisDialTimeout, e := durEnv("REDIS_DIAL_TIMEOUT", 5*time.Second)
	captureErr(e)

	redisReadTimeout, e := durEnv("REDIS_READ_TIMEOUT", 3*time.Second)
	captureErr(e)

	redisWriteTimeout, e := durEnv("REDIS_WRITE_TIMEOUT", 3*time.Second)
	captureErr(e)

	redisPoolSize, e := intEnv("REDIS_POOL_SIZE", 10)
	captureErr(e)

	redisMinIdleConns, e := intEnv("REDIS_MIN_IDLE_CONNS", 1)
	captureErr(e)

	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppName:   env("APP_NAME", "rpg-backend"),
		Env:       env("APP_ENV", "local"),
		LogLevel:  env("LOG_LEVEL", "info"),
		LogFormat: env("LOG_FORMAT", "text"),

		APIHTTPAddr:     env("API_HTTP_ADDR", ":8080"),
		GameENetAddr:    env("GAME_ENET_ADDR", ":7777"),
		GameHTTPAddr:    env("GAME_HTTP_ADDR", ":8081"),
		ShutdownTimeout: shutdownTimeout,

		Postgres: PostgresConfig{
			URL:      os.Getenv("POSTGRES_URL"),
			Host:     env("POSTGRES_HOST", "localhost"),
			Port:     postgresPort,
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Database: os.Getenv("POSTGRES_DB"),
			SSLMode:  env("POSTGRES_SSLMODE", "disable"),

			MaxConns:          postgresMaxConns,
			MinConns:          postgresMinConns,
			MaxConnLifetime:   postgresMaxConnLifetime,
			MaxConnIdleTime:   postgresMaxConnIdleTime,
			HealthCheckPeriod: postgresHealthCheckPeriod,
		},

		Redis: RedisConfig{
			Addr:     env("REDIS_ADDR", "localhost:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,

			DialTimeout:  redisDialTimeout,
			ReadTimeout:  redisReadTimeout,
			WriteTimeout: redisWriteTimeout,
			PoolSize:     redisPoolSize,
			MinIdleConns: redisMinIdleConns,
		},
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// --- VALIDATION ---

func (c Config) validate() error {
	if c.AppName == "" {
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
	if c.APIHTTPAddr == "" {
		return fmt.Errorf("API_HTTP_ADDR is required")
	}
	if c.GameENetAddr == "" {
		return fmt.Errorf("GAME_ENET_ADDR is required")
	}
	if c.GameHTTPAddr == "" {
		return fmt.Errorf("GAME_HTTP_ADDR is required")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT must be positive")
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
	if p.URL != "" {
		if err := validatePostgresURL(p.URL); err != nil {
			return err
		}
	} else {
		if p.Host == "" {
			return fmt.Errorf("POSTGRES_HOST is required when POSTGRES_URL is not set")
		}
		if p.Port <= 0 || p.Port > 65535 {
			return fmt.Errorf("POSTGRES_PORT must be between 1 and 65535, got %d", p.Port)
		}
		if p.User == "" {
			return fmt.Errorf("POSTGRES_USER is required when POSTGRES_URL is not set")
		}
		if p.Password == "" {
			return fmt.Errorf("POSTGRES_PASSWORD is required when POSTGRES_URL is not set")
		}
		if p.Database == "" {
			return fmt.Errorf("POSTGRES_DB is required when POSTGRES_URL is not set")
		}
		if !oneOf(p.SSLMode, "disable", "require", "verify-ca", "verify-full") {
			return fmt.Errorf("invalid POSTGRES_SSLMODE: %q", p.SSLMode)
		}
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

func validatePostgresURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid POSTGRES_URL: %w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("POSTGRES_URL must use postgres or postgresql scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("POSTGRES_URL must include host")
	}
	if u.User == nil || u.User.Username() == "" {
		return fmt.Errorf("POSTGRES_URL must include user")
	}
	if strings.TrimPrefix(u.Path, "/") == "" {
		return fmt.Errorf("POSTGRES_URL must include database name")
	}

	return nil
}

func (r RedisConfig) validate() error {
	if r.Addr == "" {
		return fmt.Errorf("REDIS_ADDR is required")
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

// --- HELPERS ---

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func intEnv(key string, def int) (int, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def, nil
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	return n, nil
}

func int32Env(key string, def int32) (int32, error) {
	n, err := intEnv(key, int(def))
	if err != nil {
		return 0, err
	}
	if n < -2147483648 || n > 2147483647 {
		return 0, fmt.Errorf("%s must fit in int32, got %d", key, n)
	}
	return int32(n), nil
}

func durEnv(key string, def time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def, nil
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration: %w", key, err)
	}

	return d, nil
}

func oneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}
