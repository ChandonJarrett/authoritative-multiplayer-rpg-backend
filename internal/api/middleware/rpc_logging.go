package middleware

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
)

// NewRPCLoggingInterceptor logs every unary RPC completion.
// It intentionally does not log request bodies because auth requests contain secrets.
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
				attrs = append(
					attrs,
					"code", connect.CodeOf(err).String(),
					"error", err,
				)
				log.WarnContext(ctx, "rpc failed", attrs...)
				return res, err
			}

			log.InfoContext(ctx, "rpc completed", attrs...)
			return res, nil
		}
	}
}
