// Package api implements the ConnectRPC API server and all related HTTP middleware.
package api

import (
	"net/http"

	"connectrpc.com/connect"

	rpgv1connect "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1/rpgv1connect"
)

func mountHealthRoutes(mux *http.ServeMux, readyCheck ReadyCheck) {
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/readyz", readyHandler(readyCheck, defaultReadyCheckTimeout))
}

func mountMetricsRoute(mux *http.ServeMux, metricsHandler http.Handler) {
	if metricsHandler != nil {
		mux.Handle("/metrics", metricsHandler)
	}
}

func mountRPCRoutes(
	mux *http.ServeMux,
	handlers Handlers,
	interceptors []connect.Interceptor,
) {
	connectOptions := make([]connect.HandlerOption, 0, len(interceptors))
	for _, interceptor := range interceptors {
		connectOptions = append(connectOptions, connect.WithInterceptors(interceptor))
	}

	systemPath, systemHTTPHandler := rpgv1connect.NewSystemServiceHandler(
		handlers.System,
		connectOptions...,
	)
	mux.Handle(systemPath, systemHTTPHandler)

	if handlers.Auth != nil {
		authPath, authHTTPHandler := rpgv1connect.NewAuthServiceHandler(
			handlers.Auth,
			connectOptions...,
		)
		mux.Handle(authPath, authHTTPHandler)
	}

	if handlers.Character != nil {
		characterPath, characterHTTPHandler := rpgv1connect.NewCharacterServiceHandler(
			handlers.Character,
			connectOptions...,
		)
		mux.Handle(characterPath, characterHTTPHandler)
	}

	if handlers.Game != nil {
		gamePath, gameHTTPHandler := rpgv1connect.NewGameServiceHandler(
			handlers.Game,
			connectOptions...,
		)
		mux.Handle(gamePath, gameHTTPHandler)
	}
}
