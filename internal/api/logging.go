package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
)

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
