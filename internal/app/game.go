package app

import (
	"fmt"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/game"
)

const gameServiceName = "game"

// RunGame initializes the shared runtime for the game server and blocks until shutdown.
func RunGame() error {
	rt, err := NewRuntime(gameServiceName)
	if err != nil {
		return fmt.Errorf("initialize runtime: %w", err)
	}
	defer rt.Close()

	gs, err := NewGameServer(rt)
	if err != nil {
		return fmt.Errorf("create game server: %w", err)
	}

	rt.Log.Info(
		"game server starting",
		"enet_addr", rt.Config.GameENetAddr,
		"http_addr", rt.Config.GameHTTPAddr,
	)

	if err := gs.Run(rt.Context); err != nil {
		return err
	}

	rt.Log.Info("game server stopped")
	return nil
}

// NewGameServer wires game server dependencies and returns a runnable game server.
func NewGameServer(rt *Runtime) (*game.GameServer, error) {
	if rt == nil {
		return nil, fmt.Errorf("runtime is nil")
	}

	cfg, err := newGameConfig(rt)
	if err != nil {
		return nil, err
	}

	return game.NewGameServer(rt.Log, cfg)
}
