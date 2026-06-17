// Package main is the entry point for the game server.
package main

import "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/app"

func main() {
	rt, err := app.NewRuntime("game")
	if err != nil {
		app.Fatal("failed to initialize runtime", err)
	}
	defer rt.Close()

	rt.Log.Info(
		"game server ready",
		"enet_addr", rt.Config.GameENetAddr,
		"http_addr", rt.Config.GameHTTPAddr,
	)

	// TODO: initialize ENet host, simulation loop, snapshot broadcast loop, and health server.

	<-rt.Context.Done()

	rt.Log.Info("shutdown signal received")
}
