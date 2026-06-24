package game

import (
	"time"

	rpgv1 "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1"
)

// SnapshotBuilder converts world state into a protobuf SnapshotPacket.
type SnapshotBuilder struct{}

// NewSnapshotBuilder creates a new snapshot builder.
func NewSnapshotBuilder() *SnapshotBuilder {
	return &SnapshotBuilder{}
}

// BuildSnapshot creates a SnapshotPacket from the current world state.
func (sb *SnapshotBuilder) BuildSnapshot(world *World) *rpgv1.SnapshotPacket {
	entities, tick := world.Snapshot()

	pbEntities := make([]*rpgv1.EntitySnapshot, 0, len(entities))
	for _, e := range entities {
		pbEntities = append(pbEntities, &rpgv1.EntitySnapshot{
			EntityId: e.ID,
			Type:     uint32(e.Type),
			Position: &rpgv1.Vec2{
				X: e.Position.X,
				Y: e.Position.Y,
			},
			Velocity: &rpgv1.Vec2{
				X: e.Velocity.X,
				Y: e.Velocity.Y,
			},
		})
	}

	return &rpgv1.SnapshotPacket{
		Tick:         tick,
		ServerTimeMs: uint64(time.Now().UnixMilli()),
		Entities:     pbEntities,
	}
}
