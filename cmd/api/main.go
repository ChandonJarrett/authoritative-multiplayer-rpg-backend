// Package main is the entry point for the API server.
package main

import "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/app"

func main() {
	if err := app.RunAPI(); err != nil {
		app.Fatal("api server failed", err)
	}
}
