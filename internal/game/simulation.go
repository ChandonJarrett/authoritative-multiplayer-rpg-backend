package game

import (
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	rpgv1 "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1"
)

const (
	// defaultSimulationTickRate is the target simulation frequency in Hz.
	defaultSimulationTickRate = 64

	// defaultMoveSpeed is the base movement speed in world units per tick.
	defaultMoveSpeed = 0.05
)

// Simulation processes player inputs and updates the authoritative world state.
type Simulation struct {
	tickRate  int
	moveSpeed float64
}

// NewSimulation creates a simulation with the given tick rate and movement speed.
func NewSimulation(tickRate int, moveSpeed float64) *Simulation {
	if tickRate <= 0 {
		tickRate = defaultSimulationTickRate
	}
	if moveSpeed <= 0 {
		moveSpeed = defaultMoveSpeed
	}
	return &Simulation{
		tickRate:  tickRate,
		moveSpeed: moveSpeed,
	}
}

// TickInterval returns the duration between simulation ticks.
func (s *Simulation) TickInterval() time.Duration {
	return time.Second / time.Duration(s.tickRate)
}

// ProcessInput applies a single client input packet to the corresponding world entity.
func (s *Simulation) ProcessInput(world *World, entityID string, input *rpgv1.InputPacket) {
	entity := world.GetEntity(entityID)
	if entity == nil {
		return
	}

	dir := input.GetMoveDirection()
	if dir == nil {
		return
	}

	dx := clampFloat(dir.GetX(), -1.0, 1.0)
	dy := clampFloat(dir.GetY(), -1.0, 1.0)

	entity.Position.X += dx * s.moveSpeed
	entity.Position.Y += dy * s.moveSpeed

	entity.Velocity = domain.Vec2{
		X: dx * s.moveSpeed,
		Y: dy * s.moveSpeed,
	}
}

func clampFloat(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

// DrainInputs applies the latest queued input from each active session to the world.
func (s *Simulation) DrainInputs(world *World, sessions *SessionManager) {
	for _, session := range sessions.All() {
		if session.LatestInput == nil {
			continue
		}
		s.ProcessInput(world, session.EntityID, session.LatestInput)
		session.LatestInput = nil
	}
}
