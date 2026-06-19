package api

import (
	"context"
	"net/http"
	"time"
)

const defaultReadyCheckTimeout = 2 * time.Second

// ReadyCheck verifies whether dependencies required by the API are available.
type ReadyCheck func(ctx context.Context) error

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, `{"status":"ok"}`)
}

func readyHandler(check ReadyCheck, timeout time.Duration) http.HandlerFunc {
	if timeout <= 0 {
		timeout = defaultReadyCheckTimeout
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			if err := check(ctx); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, `{"status":"not_ready"}`)
				return
			}
		}

		writeJSON(w, http.StatusOK, `{"status":"ready"}`)
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(body))
}
