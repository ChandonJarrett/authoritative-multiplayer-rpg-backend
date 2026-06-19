package observability

import (
	"context"
	"time"

	"connectrpc.com/connect"
)

const (
	rpcRequestsMetric       = "rpg_rpc_requests_total"
	rpcRequestLatencyMetric = "rpg_rpc_request_duration"
)

// RPCInterceptor records ConnectRPC request counts, status codes, and durations.
func RPCInterceptor(metrics *Metrics) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			started := time.Now()

			res, err := next(ctx, req)

			code := connect.CodeOf(err).String()

			labels := map[string]string{
				"procedure": req.Spec().Procedure,
				"code":      code,
			}

			metrics.Inc(rpcRequestsMetric, labels)
			metrics.ObserveDuration(rpcRequestLatencyMetric, time.Since(started), labels)

			return res, err
		}
	}
}
