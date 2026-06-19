package app

import (
	"fmt"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api"
)

const apiServiceName = "api"

// RunAPI initializes and runs the API server.
func RunAPI() error {
	rt, err := NewRuntime(apiServiceName)
	if err != nil {
		return fmt.Errorf("initialize runtime: %w", err)
	}
	defer rt.Close()

	server, err := NewAPIServer(rt)
	if err != nil {
		return fmt.Errorf("create api server: %w", err)
	}

	if err := server.Run(rt.Context); err != nil {
		return err
	}

	rt.Log.Info("api server stopped")
	return nil
}

// NewAPIServer wires API dependencies and returns a runnable API server.
func NewAPIServer(rt *Runtime) (*api.Server, error) {
	if rt == nil {
		return nil, fmt.Errorf("runtime is nil")
	}

	deps, err := newAPIDeps(rt)
	if err != nil {
		return nil, err
	}

	return api.NewServer(newAPIServerOptions(rt, deps))
}
