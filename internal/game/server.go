package game

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store"
	"github.com/nhh/go-enet"
)

const (
	// heartbeatInterval is how often the game server renews its Redis registry entry.
	heartbeatInterval = 5 * time.Second

	// defaultBroadcastTickDivisor controls broadcast frequency relative to simulation.
	defaultBroadcastTickDivisor = 2
)

// GameServerConfig holds all dependencies and settings for the game server.
type GameServerConfig struct {
	// Network addresses.
	ENetAddr string
	HTTPAddr string

	// Server identity.
	ServerID string

	// Shutdown deadline for graceful connection draining.
	ShutdownTimeout time.Duration

	// Stores for join handshake and server registration.
	JoinTokens     store.JoinTokenStore
	CharacterLocks characterLocker
	Characters     store.CharacterStore
	GameServers    store.GameServerStore

	// TTLs.
	ServerTTL time.Duration
	LockTTL   time.Duration

	// Simulation tuning.
	TickRate  int
	MoveSpeed float64

	// Readiness check verifies PostgreSQL and Redis connectivity.
	ReadyCheck func(ctx context.Context) error
}

// GameServer orchestrates the authoritative real-time game simulation.
type GameServer struct {
	cfg  GameServerConfig
	log  *slog.Logger
	host *ENetHost

	world    *World
	sessions *SessionManager

	joinHandler *JoinHandler
	simulation  *Simulation
	broadcaster *Broadcaster

	httpServer *http.Server
}

// NewGameServer creates and initializes a game server.
func NewGameServer(log *slog.Logger, cfg GameServerConfig) (*GameServer, error) {
	if log == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Validate required config fields.
	if cfg.ENetAddr == "" {
		return nil, fmt.Errorf("enet address is required")
	}
	if cfg.HTTPAddr == "" {
		return nil, fmt.Errorf("http address is required")
	}
	if cfg.ServerID == "" {
		return nil, fmt.Errorf("server ID is required")
	}
	if cfg.JoinTokens == nil {
		return nil, fmt.Errorf("join token store is required")
	}
	if cfg.CharacterLocks == nil {
		return nil, fmt.Errorf("character lock store is required")
	}
	if cfg.Characters == nil {
		return nil, fmt.Errorf("character store is required")
	}
	if cfg.GameServers == nil {
		return nil, fmt.Errorf("game server store is required")
	}
	if cfg.ServerTTL <= 0 {
		return nil, fmt.Errorf("server TTL must be positive")
	}
	if cfg.LockTTL <= 0 {
		return nil, fmt.Errorf("lock TTL must be positive")
	}

	// Initialize ENet.
	enet.Initialize()

	host, err := NewENetHost(cfg.ENetAddr)
	if err != nil {
		enet.Deinitialize()
		return nil, fmt.Errorf("create enet host: %w", err)
	}

	world := NewWorld()
	sessions := NewSessionManager()

	joinHandler := NewJoinHandler(
		log, cfg.JoinTokens, cfg.CharacterLocks, cfg.Characters,
		world, sessions, cfg.LockTTL, cfg.ServerID,
	)

	simulation := NewSimulation(cfg.TickRate, cfg.MoveSpeed)
	snapshotBuilder := NewSnapshotBuilder()
	broadcaster := NewBroadcaster(log, host, sessions, snapshotBuilder, world)

	return &GameServer{
		cfg:         cfg,
		log:         log,
		host:        host,
		world:       world,
		sessions:    sessions,
		joinHandler: joinHandler,
		simulation:  simulation,
		broadcaster: broadcaster,
	}, nil
}

// Run starts the game server and blocks until the context is cancelled.
func (gs *GameServer) Run(ctx context.Context) error {
	// Start health HTTP server.
	if err := gs.startHealthServer(ctx); err != nil {
		return fmt.Errorf("start health server: %w", err)
	}

	// Register with Redis and start heartbeat.
	if err := gs.registerGameServer(ctx); err != nil {
		return fmt.Errorf("register game server: %w", err)
	}
	go gs.renewHeartbeat(ctx)

	gs.log.Info(
		"game server running",
		"server_id", gs.cfg.ServerID,
		"enet_addr", gs.cfg.ENetAddr,
		"http_addr", gs.cfg.HTTPAddr,
	)

	// Run the main game loop.
	gs.runGameLoop(ctx)

	// Shutdown.
	gs.log.Info("game server shutting down")
	gs.shutdown(ctx)

	return nil
}

// runGameLoop is the main game loop that interleaves ENet event processing with
// simulation ticks and snapshot broadcasts.
//
// Input packets are queued per-session as they arrive and drained in batch during
// each simulation tick. Only the latest input per session is kept; intermediate
// inputs between ticks are overwritten, matching the authoritative model where
// the server decides state at tick boundaries.
func (gs *GameServer) runGameLoop(ctx context.Context) {
	tickInterval := gs.simulation.TickInterval()

	lastTick := time.Now()
	tickSinceBroadcast := uint64(0)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Service ENet events.
		event := gs.host.Service()
		gs.handleENetEvent(ctx, event)

		now := time.Now()

		// Run simulation tick - drain queued inputs then advance the tick counter.
		if now.Sub(lastTick) >= tickInterval {
			gs.simulation.DrainInputs(gs.world, gs.sessions)
			gs.world.IncrementTick()
			lastTick = now
			tickSinceBroadcast++
		}

		// Broadcast snapshot at the broadcast rate.
		if tickSinceBroadcast >= defaultBroadcastTickDivisor {
			if gs.sessions.Count() > 0 {
				gs.broadcaster.Broadcast()
			}
			tickSinceBroadcast = 0
		}
	}
}

// handleENetEvent dispatches a single ENet event.
func (gs *GameServer) handleENetEvent(ctx context.Context, event enet.Event) {
	switch event.GetType() {
	case enet.EventConnect:
		gs.log.Info("peer connected", "peer_addr", event.GetPeer().GetAddress().String())

	case enet.EventReceive:
		gs.handleReceive(ctx, event)

	case enet.EventDisconnect:
		gs.joinHandler.HandleDisconnect(ctx, event.GetPeer())

	case enet.EventNone:
		// No event - nothing to do.
	}
}

// handleReceive processes an incoming packet based on the channel.
func (gs *GameServer) handleReceive(ctx context.Context, event enet.Event) {
	peer := event.GetPeer()
	packet := event.GetPacket()
	defer packet.Destroy()

	data := packet.GetData()

	switch event.GetChannelID() {
	case channelReliable:
		if len(data) == 0 {
			return
		}
		gs.joinHandler.HandleJoinRequest(ctx, peer, data)

	case channelUnreliable:
		session := gs.sessions.Get(peer)
		if session == nil {
			return
		}

		input, err := unmarshalInputPacket(data)
		if err != nil {
			return
		}

		// Queue for the next simulation tick; only the latest input is kept.
		session.LatestInput = input
	}
}

// startHealthServer starts an HTTP server for health and readiness checks.
func (gs *GameServer) startHealthServer(_ context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	if gs.cfg.ReadyCheck != nil {
		mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
			checkCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()

			if err := gs.cfg.ReadyCheck(checkCtx); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"status":"not_ready"}`))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ready"}`))
		})
	}

	gs.httpServer = &http.Server{
		Addr:              gs.cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		gs.log.Info("health server listening", "addr", gs.cfg.HTTPAddr)
		if err := gs.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			gs.log.Error("health server error", "error", err)
		}
	}()

	return nil
}

// registerGameServer registers this game server in Redis.
func (gs *GameServer) registerGameServer(ctx context.Context) error {
	server := domain.GameServer{
		ID:        gs.cfg.ServerID,
		Addr:      gs.cfg.ENetAddr,
		ExpiresAt: time.Now().UTC().Add(gs.cfg.ServerTTL),
	}

	if err := gs.cfg.GameServers.RegisterGameServer(ctx, server); err != nil {
		return fmt.Errorf("register game server in redis: %w", err)
	}

	gs.log.Info("game server registered", "server_id", gs.cfg.ServerID)
	return nil
}

// renewHeartbeat periodically renews the game server's Redis registry entry.
func (gs *GameServer) renewHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			server := domain.GameServer{
				ID:        gs.cfg.ServerID,
				Addr:      gs.cfg.ENetAddr,
				ExpiresAt: time.Now().UTC().Add(gs.cfg.ServerTTL),
			}

			if err := gs.cfg.GameServers.RegisterGameServer(ctx, server); err != nil {
				gs.log.Error("failed to renew game server heartbeat", "error", err)
			}
		}
	}
}

// shutdown performs graceful shutdown: stop accepting new ENet connections,
// disconnect all peers, stop the health server, and deregister from Redis.
func (gs *GameServer) shutdown(_ context.Context) {
	// Stop health HTTP server.
	if gs.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), gs.cfg.ShutdownTimeout)
		defer cancel()

		if err := gs.httpServer.Shutdown(shutdownCtx); err != nil {
			gs.log.Error("health server shutdown error", "error", err)
		}
	}

	// Disconnect all active peers.
	for _, session := range gs.sessions.All() {
		session.Peer.Disconnect(0)
		// Release character locks.
		if _, err := gs.cfg.CharacterLocks.ReleaseCharacterLock(
			context.Background(), session.CharacterID, gs.cfg.ServerID,
		); err != nil {
			gs.log.Error(
				"failed to release lock during shutdown",
				"character_id", session.CharacterID,
				"error", err,
			)
		}
	}

	gs.host.Destroy()
	enet.Deinitialize()

	// Deregister from Redis.
	if gs.cfg.GameServers != nil {
		if err := gs.cfg.GameServers.DeregisterGameServer(context.Background(), gs.cfg.ServerID); err != nil {
			gs.log.Error("failed to deregister game server", "error", err)
		}
	}
}
