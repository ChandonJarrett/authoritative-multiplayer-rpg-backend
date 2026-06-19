package observability

import (
	"net/http"
	"strconv"
	"time"
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
		rec := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		status := rec.statusCode
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

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	if r.statusCode != 0 {
		return
	}

	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}

	return r.ResponseWriter.Write(data)
}
