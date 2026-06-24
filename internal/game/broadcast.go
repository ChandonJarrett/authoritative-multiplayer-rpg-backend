package game

import (
	"log/slog"

	"github.com/nhh/go-enet"
)

// Broadcaster sends world snapshots to all connected peers.
type Broadcaster struct {
	log       *slog.Logger
	host      *ENetHost
	sessions  *SessionManager
	snapshots *SnapshotBuilder
	world     *World
}

// NewBroadcaster creates a snapshot broadcaster.
func NewBroadcaster(
	log *slog.Logger,
	host *ENetHost,
	sessions *SessionManager,
	snapshots *SnapshotBuilder,
	world *World,
) *Broadcaster {
	return &Broadcaster{
		log:       log,
		host:      host,
		sessions:  sessions,
		snapshots: snapshots,
		world:     world,
	}
}

// Broadcast sends the current world snapshot to all connected peers on the unreliable channel.
func (b *Broadcaster) Broadcast() {
	pkt := b.snapshots.BuildSnapshot(b.world)

	data, err := marshalSnapshotPacket(pkt)
	if err != nil {
		b.log.Error("failed to marshal snapshot", "error", err)
		return
	}

	if err := b.host.Broadcast(data, channelUnreliable, enet.PacketFlagUnsequenced); err != nil {
		b.log.Error("failed to broadcast snapshot", "error", err)
	}
}
