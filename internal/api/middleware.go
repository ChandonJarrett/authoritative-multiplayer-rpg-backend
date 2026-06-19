package api

import (
	"log/slog"
	"net/http"
)

// Middleware wraps an HTTP handler.
type Middleware func(http.Handler) http.Handler

// ChainMiddleware applies middleware in listed order.
//
// The first middleware becomes the outermost wrapper.
func ChainMiddleware(handler http.Handler, middleware ...Middleware) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		if middleware[i] != nil {
			handler = middleware[i](handler)
		}
	}
	return handler
}

// StandardMiddleware returns the default API HTTP middleware stack.
//
// Order matters:
//  1. Request ID first, so every later middleware can read it.
//  2. Panic recovery wraps logging and handlers.
//  3. Request logging observes final status and bytes.
//  4. Optional custom middleware.
//  5. CORS closest to routes, so preflight does not trigger RPC handlers.
func StandardMiddleware(
	log *slog.Logger,
	allowedOrigins []string,
	custom Middleware,
) []Middleware {
	stack := []Middleware{
		WithRequestID,
		func(next http.Handler) http.Handler {
			return WithPanicRecovery(log, next)
		},
		func(next http.Handler) http.Handler {
			return WithRequestLogging(log, next)
		},
	}

	if custom != nil {
		stack = append(stack, custom)
	}

	stack = append(stack, func(next http.Handler) http.Handler {
		return WithCORS(next, allowedOrigins)
	})

	return stack
}
