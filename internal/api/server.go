// Package api contains HTTP and ConnectRPC server wiring.
package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api/middleware"
	rpgv1connect "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1/rpgv1connect"
)

const defaultReadyCheckTimeout = 2 * time.Second

// ReadyCheck verifies whether dependencies required by the API are available.
type ReadyCheck func(ctx context.Context) error

// Handlers groups all ConnectRPC handlers for the API server.
type Handlers struct {
	System    rpgv1connect.SystemServiceHandler
	Auth      rpgv1connect.AuthServiceHandler
	Character rpgv1connect.CharacterServiceHandler
	Game      rpgv1connect.GameServiceHandler
}

// Options configures the API HTTP server.
type Options struct {
	Addr              string
	Log               *slog.Logger
	ShutdownTimeout   time.Duration
	AllowedOrigins    []string
	UnaryInterceptors []connect.Interceptor
	ReadyCheck        ReadyCheck
	Handlers          Handlers
}

// Server owns the API HTTP server lifecycle.
type Server struct {
	httpServer      *http.Server
	handler         http.Handler
	log             *slog.Logger
	shutdownTimeout time.Duration
}

// NewServer builds the API server and mounts all HTTP and ConnectRPC routes.
func NewServer(opts Options) (*Server, error) {
	if strings.TrimSpace(opts.Addr) == "" {
		return nil, errors.New("api server addr is required")
	}

	log := opts.Log
	if log == nil {
		log = slog.Default()
	}

	shutdownTimeout := opts.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 10 * time.Second
	}

	allowedOrigins := opts.AllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:5173",
		}
	}

	if opts.Handlers.System == nil {
		return nil, errors.New("system handler is required")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/readyz", readyHandler(opts.ReadyCheck, defaultReadyCheckTimeout))

	connectOptions := make([]connect.HandlerOption, 0, len(opts.UnaryInterceptors))
	for _, interceptor := range opts.UnaryInterceptors {
		connectOptions = append(connectOptions, connect.WithInterceptors(interceptor))
	}

	systemPath, systemHTTPHandler := rpgv1connect.NewSystemServiceHandler(opts.Handlers.System, connectOptions...)
	mux.Handle(systemPath, systemHTTPHandler)

	if opts.Handlers.Auth != nil {
		authPath, authHTTPHandler := rpgv1connect.NewAuthServiceHandler(opts.Handlers.Auth, connectOptions...)
		mux.Handle(authPath, authHTTPHandler)
	}

	if opts.Handlers.Character != nil {
		characterPath, characterHTTPHandler := rpgv1connect.NewCharacterServiceHandler(opts.Handlers.Character, connectOptions...)
		mux.Handle(characterPath, characterHTTPHandler)
	}

	if opts.Handlers.Game != nil {
		gamePath, gameHTTPHandler := rpgv1connect.NewGameServiceHandler(opts.Handlers.Game, connectOptions...)
		mux.Handle(gamePath, gameHTTPHandler)
	}

	handler := middleware.WithPanicRecovery(
		log,
		middleware.WithRequestLogging(
			log,
			middleware.WithRequestID(
				middleware.WithCORS(mux, allowedOrigins),
			),
		),
	)

	httpServer := &http.Server{
		Addr:              opts.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	httpServer.Protocols = new(http.Protocols)
	httpServer.Protocols.SetHTTP1(true)
	httpServer.Protocols.SetUnencryptedHTTP2(true)

	return &Server{
		httpServer:      httpServer,
		handler:         handler,
		log:             log,
		shutdownTimeout: shutdownTimeout,
	}, nil
}

// Handler returns the root HTTP handler. It is primarily useful for tests.
func (s *Server) Handler() http.Handler {
	return s.handler
}

// Run starts the server and blocks until the context is cancelled or the server fails.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.log.Info("api http server listening", "addr", s.httpServer.Addr)

		err := s.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}

		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		s.log.Info("api http server shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			_ = s.httpServer.Close()
			return fmt.Errorf("shutdown api http server: %w", err)
		}

		return nil

	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("run api http server: %w", err)
		}

		return nil
	}
}

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
