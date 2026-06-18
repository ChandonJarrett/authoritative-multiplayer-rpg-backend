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
	rpgv1connect "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1/rpgv1connect"
)

const defaultReadyCheckTimeout = 2 * time.Second

// ReadyCheck verifies whether dependencies required by the API are available.
type ReadyCheck func(ctx context.Context) error

// Options configures the API HTTP server.
type Options struct {
	Addr              string
	Log               *slog.Logger
	ShutdownTimeout   time.Duration
	AllowedOrigins    []string
	ReadyCheck        ReadyCheck
	SystemHandler     rpgv1connect.SystemServiceHandler
	AuthHandler       rpgv1connect.AuthServiceHandler
	UnaryInterceptors []connect.Interceptor
	CharacterHandler  rpgv1connect.CharacterServiceHandler
	GameHandler       rpgv1connect.GameServiceHandler
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

	systemHandler := opts.SystemHandler
	if systemHandler == nil {
		systemHandler = NewSystemHandler("api")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/readyz", readyHandler(opts.ReadyCheck, defaultReadyCheckTimeout))

	connectOptions := make([]connect.HandlerOption, 0, len(opts.UnaryInterceptors))
	for _, interceptor := range opts.UnaryInterceptors {
		connectOptions = append(connectOptions, connect.WithInterceptors(interceptor))
	}

	systemPath, systemHTTPHandler := rpgv1connect.NewSystemServiceHandler(systemHandler, connectOptions...)
	mux.Handle(systemPath, systemHTTPHandler)

	if opts.AuthHandler != nil {
		authPath, authHTTPHandler := rpgv1connect.NewAuthServiceHandler(opts.AuthHandler, connectOptions...)
		mux.Handle(authPath, authHTTPHandler)
	}

	if opts.CharacterHandler != nil {
		characterPath, characterHTTPHandler := rpgv1connect.NewCharacterServiceHandler(opts.CharacterHandler, connectOptions...)
		mux.Handle(characterPath, characterHTTPHandler)
	}

	if opts.GameHandler != nil {
		gamePath, gameHTTPHandler := rpgv1connect.NewGameServiceHandler(opts.GameHandler, connectOptions...)
		mux.Handle(gamePath, gameHTTPHandler)
	}

	handler := withRequestLogging(log, withRequestID(withCORS(mux, allowedOrigins)))

	httpServer := &http.Server{
		Addr:    opts.Addr,
		Handler: handler,

		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Enable HTTP/1 + unencrypted HTTP/2 (h2c)
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
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
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"status":"not_ready"}`))
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}
}

func withCORS(next http.Handler, allowedOrigins []string) http.Handler {
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

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || strings.EqualFold(origin, allowed) {
			return true
		}
	}

	return false
}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := normalizeRequestID(r.Header.Get(requestIDHeader))
		w.Header().Set(requestIDHeader, requestID)

		ctx := ContextWithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
