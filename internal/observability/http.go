package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api"
)

const (
	httpRequestsMetric       = "rpg_http_requests_total"
	httpRequestLatencyMetric = "rpg_http_request_duration"
)

// HTTPMiddleware records HTTP request counts and durations.
func HTTPMiddleware(metrics *Metrics, next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		rec := &api.ResponseRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		status := rec.StatusCode
		if status == 0 {
			status = http.StatusOK
		}

		labels := map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
			"status": strconv.Itoa(status),
		}

		metrics.Inc(httpRequestsMetric, labels)
		metrics.ObserveDuration(httpRequestLatencyMetric, time.Since(started), labels)
	})
}
