// Package main is the entry point for the game server.
package main

import "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/app"

func main() {
	if err := app.RunGame(); err != nil {
		app.Fatal("game server failed", err)
	}
}
