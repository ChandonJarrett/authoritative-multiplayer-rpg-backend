package api

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-Id"

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
