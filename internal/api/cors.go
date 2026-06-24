package api

import (
	"net/http"
	"strings"
)

// CORSMiddleware returns middleware that adds CORS headers for browser-based ConnectRPC clients.
func CORSMiddleware(allowedOrigins []string) Middleware {
	return func(next http.Handler) http.Handler {
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
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || strings.EqualFold(origin, allowed) {
			return true
		}
	}

	return false
}
