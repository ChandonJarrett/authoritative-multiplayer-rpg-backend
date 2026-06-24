// Package game contains the core game logic for the authoritative multiplayer RPG backend.
package game

import (
	"sync"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

// Entity represents a game entity, such as a player or NPC.
type Entity struct {
	ID       string
	Type     EntityType
	Position domain.Vec2
	Velocity domain.Vec2
}

// EntityType represents the type of an entity in the game world.
type EntityType uint32

// Constants for different entity types.
const (
	EntityTypePlayer EntityType = 0
	EntityTypeNPC    EntityType = 1
)

// World represents the game world, containing all entities and managing their state.
type World struct {
	mu       sync.RWMutex
	entities map[string]*Entity
	tick     uint64
}

// NewWorld creates an empty world.
func NewWorld() *World {
	return &World{
		entities: make(map[string]*Entity),
	}
}

// AddEntity inserts an entity into the world.
func (w *World) AddEntity(entity *Entity) {
	w.entities[entity.ID] = entity
}

// GetEntity returns the entity with the given ID, or nil if not found.
func (w *World) GetEntity(id string) *Entity {
	return w.entities[id]
}

// RemoveEntity deletes an entity from the world.
func (w *World) RemoveEntity(entity *Entity) {
	delete(w.entities, entity.ID)
}

// Snapshot returns an immutable copy of all entities for broadcasting.
func (w *World) Snapshot() ([]Entity, uint64) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	entities := make([]Entity, 0, len(w.entities))
	for _, e := range w.entities {
		entities = append(entities, *e)
	}
	return entities, w.tick
}

// Tick returns the current simulation tick of the world.
func (w *World) Tick() uint64 {
	return w.tick
}

// IncrementTick increments the simulation tick of the world.
func (w *World) IncrementTick() {
	w.tick++
}
