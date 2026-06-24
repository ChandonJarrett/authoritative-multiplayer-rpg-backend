package api

import (
	"log/slog"
	"net/http"
)

// RecoveryMiddleware returns middleware that prevents handler panics from crashing the API process.
func RecoveryMiddleware(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
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
}
