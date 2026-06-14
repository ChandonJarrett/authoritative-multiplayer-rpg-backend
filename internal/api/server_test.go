package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	rpgv1 "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1"
	rpgv1connect "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1/rpgv1connect"
)

func TestHealthz(t *testing.T) {
	server := newTestServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())
	if body != `{"status":"ok"}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestReadyzSuccess(t *testing.T) {
	server := newTestServer(t, func(context.Context) error { return nil })
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())
	if body != `{"status":"ready"}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestSystemPing(t *testing.T) {
	fixedTime := time.Date(2026, 6, 14, 18, 0, 0, 0, time.UTC)

	server := newTestServerWithSystemHandler(t, &SystemHandler{
		ServiceName: "api-test",
		Now:         func() time.Time { return fixedTime },
	})

	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	client := rpgv1connect.NewSystemServiceClient(httpServer.Client(), httpServer.URL)

	res, err := client.Ping(context.Background(), connect.NewRequest(&rpgv1.PingRequest{}))
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	if res.Msg.Service != "api-test" {
		t.Fatalf("expected service api-test, got %q", res.Msg.Service)
	}

	if res.Msg.Message != "pong" {
		t.Fatalf("expected message pong, got %q", res.Msg.Message)
	}

	if res.Msg.ServerTime != fixedTime.Format(time.RFC3339Nano) {
		t.Fatalf("unexpected server time: %q", res.Msg.ServerTime)
	}
}

func TestCORSPreflightAllowedOrigin(t *testing.T) {
	server := newTestServer(t, nil)

	req := httptest.NewRequest(http.MethodOptions, "/rpg.v1.SystemService/Ping", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type,Connect-Protocol-Version")

	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("unexpected allow origin: %q", got)
	}
}

func TestCORSPreflightBlockedOrigin(t *testing.T) {
	server := newTestServer(t, nil)

	req := httptest.NewRequest(http.MethodOptions, "/rpg.v1.SystemService/Ping", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Access-Control-Request-Method", "POST")

	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func newTestServer(t *testing.T, readyCheck ReadyCheck) *Server {
	t.Helper()

	return newTestServerWithOptions(t, Options{
		Addr:            "127.0.0.1:0",
		Log:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		ShutdownTimeout: time.Second,
		ReadyCheck:      readyCheck,
		AllowedOrigins: []string{
			"http://localhost:5173",
		},
	})
}

func newTestServerWithSystemHandler(t *testing.T, handler rpgv1connect.SystemServiceHandler) *Server {
	t.Helper()

	return newTestServerWithOptions(t, Options{
		Addr:            "127.0.0.1:0",
		Log:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		ShutdownTimeout: time.Second,
		SystemHandler:   handler,
		AllowedOrigins: []string{
			"http://localhost:5173",
		},
	})
}

func newTestServerWithOptions(t *testing.T, opts Options) *Server {
	t.Helper()

	server, err := NewServer(opts)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	return server
}
