package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	redisstore "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store/redis"
)

const (
	requestIDHeader = "X-Request-Id"

	defaultAuthRateLimitWindow = time.Minute
	defaultAuthRateLimitBurst  = 10
)

// SessionReader is the Redis-backed dependency used by auth middleware.
type SessionReader interface {
	GetSessionUserID(ctx context.Context, token string) (string, error)
}

// Limiter is the rate-limit dependency used by RPC middleware.
type Limiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// PublicProcedures returns RPC procedures that do not require authentication.
func PublicProcedures() map[string]struct{} {
	return map[string]struct{}{
		"/rpg.v1.SystemService/Ping":   {},
		"/rpg.v1.AuthService/Register": {},
		"/rpg.v1.AuthService/Login":    {},
	}
}

// NewAuthInterceptor creates an authentication interceptor that validates bearer sessions.
func NewAuthInterceptor(sessions redisstore.SessionStore, publicMethods ...map[string]struct{}) connect.UnaryInterceptorFunc {
	public := PublicProcedures()
	if len(publicMethods) > 0 && publicMethods[0] != nil {
		public = publicMethods[0]
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if _, ok := public[req.Spec().Procedure]; ok {
				return next(ctx, req)
			}

			token, err := BearerToken(req.Header().Get("Authorization"))
			if err != nil {
				return nil, ToConnectError(err)
			}

			userID, err := sessions.GetSessionUserID(ctx, token)
			if err != nil {
				return nil, ToConnectError(err)
			}

			ctx = ContextWithAuthUser(ctx, AuthUser{UserID: userID})
			return next(ctx, req)
		}
	}
}

// BearerToken extracts a bearer token from an Authorization header.
func BearerToken(header string) (string, error) {
	const prefix = "Bearer "

	if !strings.HasPrefix(header, prefix) {
		return "", domain.ErrUnauthenticated
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", domain.ErrUnauthenticated
	}

	return token, nil
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

// NewRPCLoggingInterceptor logs every unary RPC completion.
// Request bodies are intentionally not logged because auth requests contain secrets.
func NewRPCLoggingInterceptor(log *slog.Logger) connect.UnaryInterceptorFunc {
	if log == nil {
		log = slog.Default()
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			started := time.Now()

			res, err := next(ctx, req)

			attrs := []any{
				"procedure", req.Spec().Procedure,
				"duration_ms", time.Since(started).Milliseconds(),
				"request_id", RequestIDFromContext(ctx),
			}

			if err != nil {
				attrs = append(attrs, "code", connect.CodeOf(err).String(), "error", err)
				log.WarnContext(ctx, "rpc failed", attrs...)
				return res, err
			}

			log.InfoContext(ctx, "rpc completed", attrs...)
			return res, nil
		}
	}
}

// WithRequestID attaches a request ID to every HTTP request context and response.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := normalizeRequestID(r.Header.Get(requestIDHeader))
		w.Header().Set(requestIDHeader, requestID)

		ctx := ContextWithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func normalizeRequestID(raw string) string {
	requestID := strings.TrimSpace(raw)
	if requestID != "" {
		return requestID
	}

	return uuid.NewString()
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	if r.statusCode != 0 {
		return
	}

	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}

	n, err := r.ResponseWriter.Write(data)
	r.bytes += n
	return n, err
}

// WithRequestLogging logs every completed HTTP request.
func WithRequestLogging(log *slog.Logger, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		rec := &responseRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		statusCode := rec.statusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		log.InfoContext(
			r.Context(),
			"http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", statusCode,
			"bytes", rec.bytes,
			"duration_ms", time.Since(started).Milliseconds(),
			"request_id", RequestIDFromContext(r.Context()),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// WithPanicRecovery prevents handler panics from crashing the API process.
func WithPanicRecovery(log *slog.Logger, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			log.ErrorContext(
				r.Context(),
				"http panic recovered",
				"panic", recovered,
				"method", r.Method,
				"path", r.URL.Path,
				"request_id", RequestIDFromContext(r.Context()),
				"remote_addr", r.RemoteAddr,
			)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"status":"error","message":"internal error"}`))
		}()

		next.ServeHTTP(w, r)
	})
}

// WithCORS adds CORS headers for browser-based ConnectRPC clients.
func WithCORS(next http.Handler, allowedOrigins []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !isOriginAllowed(origin, allowedOrigins) {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", strings.Join([]string{
				"Authorization",
				"Content-Type",
				"Connect-Protocol-Version",
				"Connect-Timeout-Ms",
				"Grpc-Timeout",
				"X-Grpc-Web",
				"X-Request-Id",
				"X-User-Agent",
			}, ", "))
			w.Header().Set("Access-Control-Expose-Headers", strings.Join([]string{
				"Connect-Protocol-Version",
				"Grpc-Message",
				"Grpc-Status",
				"Grpc-Status-Details-Bin",
				"X-Request-Id",
			}, ", "))
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || strings.EqualFold(origin, allowed) {
			return true
		}
	}

	return false
}
