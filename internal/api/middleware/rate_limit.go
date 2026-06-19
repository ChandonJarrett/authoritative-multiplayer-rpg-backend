package middleware

import (
	"context"
	"errors"
	"sync"
	"time"

	"connectrpc.com/connect"
)

const (
	defaultAuthRateLimitWindow = time.Minute
	defaultAuthRateLimitBurst  = 10
)

// RateLimitConfig configures the in-memory RPC rate limiter.
type RateLimitConfig struct {
	Window     time.Duration
	Burst      int
	Procedures map[string]struct{}
	KeyFunc    func(context.Context, connect.AnyRequest) string
}

// NewAuthRateLimitInterceptor rate-limits public auth endpoints.
//
// This limiter is per-process. It is enough to stop accidental local abuse and
// basic single-instance attacks. Replace it with a Redis-backed limiter before
// running multiple API replicas behind a load balancer.
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

	limiter := newWindowLimiter(cfg.Window, cfg.Burst)

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if _, ok := cfg.Procedures[req.Spec().Procedure]; !ok {
				return next(ctx, req)
			}

			key := cfg.KeyFunc(ctx, req)
			if key == "" {
				key = "unknown"
			}

			if !limiter.Allow(req.Spec().Procedure + ":" + key) {
				return nil, connect.NewError(
					connect.CodeResourceExhausted,
					errors.New("rate limit exceeded"),
				)
			}

			return next(ctx, req)
		}
	}
}

func defaultRateLimitKey(_ context.Context, req connect.AnyRequest) string {
	for _, header := range []string{
		"X-Forwarded-For",
		"X-Real-IP",
	} {
		value := req.Header().Get(header)
		if value != "" {
			return value
		}
	}

	return "anonymous"
}

type windowLimiter struct {
	mu     sync.Mutex
	window time.Duration
	burst  int
	now    func() time.Time
	items  map[string]windowCounter
}

type windowCounter struct {
	start time.Time
	count int
}

func newWindowLimiter(window time.Duration, burst int) *windowLimiter {
	return &windowLimiter{
		window: window,
		burst:  burst,
		now:    time.Now,
		items:  make(map[string]windowCounter),
	}
}

func (l *windowLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	counter := l.items[key]

	if counter.start.IsZero() || now.Sub(counter.start) >= l.window {
		l.items[key] = windowCounter{
			start: now,
			count: 1,
		}
		l.deleteExpiredLocked(now)
		return true
	}

	if counter.count >= l.burst {
		return false
	}

	counter.count++
	l.items[key] = counter
	return true
}

func (l *windowLimiter) deleteExpiredLocked(now time.Time) {
	for key, counter := range l.items {
		if now.Sub(counter.start) >= l.window {
			delete(l.items, key)
		}
	}
}
