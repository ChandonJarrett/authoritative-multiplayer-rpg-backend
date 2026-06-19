package app

import "fmt"

const gameServiceName = "game"

// RunGame initializes the shared runtime for the game server and blocks until shutdown.
func RunGame() error {
	rt, err := NewRuntime(gameServiceName)
	if err != nil {
		return fmt.Errorf("initialize runtime: %w", err)
	}
	defer rt.Close()

	rt.Log.Info(
		"game server ready",
		"enet_addr", rt.Config.GameENetAddr,
		"http_addr", rt.Config.GameHTTPAddr,
	)

	// TODO: initialize ENet host, simulation loop, snapshot broadcast loop, and health server.
	<-rt.Context.Done()

	rt.Log.Info("game server stopped")
	return nil
}
