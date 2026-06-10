// Package main is the entry point for the API server.
package main

import "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/app"

func main() {
	rt, err := app.NewRuntime("api")
	if err != nil {
		app.Fatal("failed to initialize runtime", err)
	}
	defer rt.Close()

	rt.Log.Info("api server ready", "addr", rt.Config.APIHTTPAddr)

	// TODO: start ConnectRPC handler

	<-rt.Context.Done()

	rt.Log.Info("shutdown signal received")
}
