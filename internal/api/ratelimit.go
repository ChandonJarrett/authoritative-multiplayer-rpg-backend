package api

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
)

const (
	defaultAuthRateLimitWindow = time.Minute
	defaultAuthRateLimitBurst  = 10
)

// Limiter is the rate-limit dependency used by RPC middleware.
type Limiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// RateLimitConfig configures RPC rate limiting.
type RateLimitConfig struct {
	Window     time.Duration
	Burst      int
	Procedures map[string]struct{}
	KeyFunc    func(context.Context, connect.AnyRequest) string
	Limiter    Limiter
}

// NewAuthRateLimitInterceptor rate-limits public auth endpoints.
func NewAuthRateLimitInterceptor(cfg RateLimitConfig) connect.UnaryInterceptorFunc {
	if cfg.Window <= 0 {
		cfg.Window = defaultAuthRateLimitWindow
	}
	if cfg.Burst <= 0 {
		cfg.Burst = defaultAuthRateLimitBurst
	}
	if cfg.Procedures == nil {
		cfg.Procedures = map[string]struct{}{
			"/rpg.v1.AuthService/Register": {},
			"/rpg.v1.AuthService/Login":    {},
		}
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = defaultRateLimitKey
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if _, ok := cfg.Procedures[req.Spec().Procedure]; !ok {
				return next(ctx, req)
			}

			if cfg.Limiter == nil {
				return nil, connect.NewError(connect.CodeUnavailable, errors.New("rate limiter unavailable"))
			}

			key := strings.TrimSpace(cfg.KeyFunc(ctx, req))
			if key == "" {
				key = "unknown"
			}

			allowed, err := cfg.Limiter.Allow(ctx, req.Spec().Procedure+":"+key)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnavailable, errors.New("rate limiter unavailable"))
			}
			if !allowed {
				return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("rate limit exceeded"))
			}

			return next(ctx, req)
		}
	}
}

func defaultRateLimitKey(_ context.Context, req connect.AnyRequest) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := req.Header().Get(header)
		if value != "" {
			return firstForwardedIP(value)
		}
	}
	return "anonymous"
}

func firstForwardedIP(value string) string {
	beforeComma, _, _ := strings.Cut(value, ",")
	return strings.TrimSpace(beforeComma)
}
