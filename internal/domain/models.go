// Package domain defines the core data models used throughout the application.
package domain

import "time"

// User represents a registered user in the system.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Character represents a player's in-game avatar, linked to a User.
type Character struct {
	ID        string
	UserID    string
	Name      string
	MapID     string
	Position  Vec2
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Vec2 represents a 2D vector for position or movement in the game world.
type Vec2 struct {
	X float64
	Y float64
}

// Session represents a user's active session in the system.
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

// JoinToken represents a temporary token that allows a user to join a game server with a specific character.
type JoinToken struct {
	Token       string
	UserID      string
	CharacterID string
	ServerID    string
	ServerAddr  string
	ExpiresAt   time.Time
}

// GameServer represents a game server in the system.
type GameServer struct {
	ID        string
	Addr      string
	ExpiresAt time.Time
}
