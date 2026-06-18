package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-Id"

type requestIDContextKey struct{}

// WithRequestID attaches a request ID to every HTTP request context and response.
func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := normalizeRequestID(r.Header.Get(requestIDHeader))
		w.Header().Set(requestIDHeader, requestID)

		ctx := ContextWithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ContextWithRequestID returns a new context with the request ID attached.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestIDFromContext returns the request ID stored in context, if present.
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func normalizeRequestID(raw string) string {
	requestID := strings.TrimSpace(raw)
	if requestID != "" {
		return requestID
	}

	return uuid.NewString()
}
