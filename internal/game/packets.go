package game

import (
	"fmt"

	rpgv1 "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1"
	"google.golang.org/protobuf/proto"
)

// marshalJoinResponse serializes a JoinResponse for sending on the reliable channel.
func marshalJoinResponse(resp *rpgv1.JoinResponse) ([]byte, error) {
	data, err := proto.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshal join response: %w", err)
	}
	return data, nil
}

// unmarshalJoinRequest deserializes a JoinRequest received on the reliable channel.
func unmarshalJoinRequest(data []byte) (*rpgv1.JoinRequest, error) {
	req := &rpgv1.JoinRequest{}
	if err := proto.Unmarshal(data, req); err != nil {
		return nil, fmt.Errorf("unmarshal join request: %w", err)
	}
	return req, nil
}

// unmarshalInputPacket deserializes an InputPacket received on the unreliable channel.
func unmarshalInputPacket(data []byte) (*rpgv1.InputPacket, error) {
	pkt := &rpgv1.InputPacket{}
	if err := proto.Unmarshal(data, pkt); err != nil {
		return nil, fmt.Errorf("unmarshal input packet: %w", err)
	}
	return pkt, nil
}

// marshalSnapshotPacket serializes a SnapshotPacket for broadcasting on the unreliable channel.
func marshalSnapshotPacket(pkt *rpgv1.SnapshotPacket) ([]byte, error) {
	data, err := proto.Marshal(pkt)
	if err != nil {
		return nil, fmt.Errorf("marshal snapshot packet: %w", err)
	}
	return data, nil
}
