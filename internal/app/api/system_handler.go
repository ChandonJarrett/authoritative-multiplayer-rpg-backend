// Package api contains HTTP and ConnectRPC server wiring.
package api

import (
	"context"
	"time"

	"connectrpc.com/connect"

	rpgv1 "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1"
	rpgv1connect "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1/rpgv1connect"
)

var _ rpgv1connect.SystemServiceHandler = (*SystemHandler)(nil)

// SystemHandler implements basic API connectivity RPCs.
type SystemHandler struct {
	ServiceName string
	Now         func() time.Time
}

// NewSystemHandler creates a SystemHandler.
func NewSystemHandler(serviceName string) *SystemHandler {
	return &SystemHandler{
		ServiceName: serviceName,
		Now:         time.Now,
	}
}

// Ping verifies that the API server is reachable through ConnectRPC.
func (h *SystemHandler) Ping(
	_ context.Context,
	_ *connect.Request[rpgv1.PingRequest],
) (*connect.Response[rpgv1.PingResponse], error) {
	nowFn := h.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	serviceName := h.ServiceName
	if serviceName == "" {
		serviceName = "api"
	}

	return connect.NewResponse(&rpgv1.PingResponse{
		Service:    serviceName,
		Message:    "pong",
		ServerTime: nowFn().UTC().Format(time.RFC3339Nano),
	}), nil
}
