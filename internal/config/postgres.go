package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPostgresHost    = "localhost"
	defaultPostgresPort    = 5432
	defaultPostgresSSLMode = "disable"

	defaultPostgresMaxConns          = 10
	defaultPostgresMinConns          = 1
	defaultPostgresMaxConnLifetime   = time.Hour
	defaultPostgresMaxConnIdleTime   = 30 * time.Minute
	defaultPostgresHealthCheckPeriod = time.Minute
)

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
