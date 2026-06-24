package game

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nhh/go-enet"
)

const (
	// maxPeers is the maximum number of concurrent connections the game server accepts.
	maxPeers = 32

	// channelCount is the total number of ENet channels allocated per connection.
	// Channel 0: reliable (JoinRequest, JoinResponse)
	// Channel 1: unreliable (InputPacket, SnapshotPacket)
	channelCount = 2

	// Channel IDs.
	channelReliable   = 0
	channelUnreliable = 1

	// enetServiceTimeoutMS is the timeout for each Host.Service() call in milliseconds.
	enetServiceTimeoutMS = 1
)

// ENetHost wraps an ENet Host for the game server.
type ENetHost struct {
	host enet.Host
}

// NewENetHost creates and returns an ENet host bound to the given address.
// The addr must be in the form "host:port", e.g. ":7777".
func NewENetHost(addr string) (*ENetHost, error) {
	_, port, err := splitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("parse enet address: %w", err)
	}

	listenAddr := enet.NewListenAddress(port)

	h, err := enet.NewHost(listenAddr, maxPeers, channelCount, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("create enet host on %s: %w", addr, err)
	}

	return &ENetHost{host: h}, nil
}

// Service blocks for up to timeoutMS milliseconds and returns the next ENet event.
// Returns EventNone if no event occurred within the timeout.
func (h *ENetHost) Service() enet.Event {
	return h.host.Service(enetServiceTimeoutMS)
}

// Broadcast sends data to all connected peers on the given channel with the given flags.
func (h *ENetHost) Broadcast(data []byte, channel uint8, flags enet.PacketFlags) error {
	return h.host.BroadcastBytes(data, channel, flags)
}

// Destroy destroys the ENet host and frees all associated resources.
func (h *ENetHost) Destroy() {
	if h.host != nil {
		h.host.Destroy()
	}
}

// splitHostPort splits a "host:port" address string into host and port components.
func splitHostPort(addr string) (string, uint16, error) {
	if addr == "" {
		return "", 0, fmt.Errorf("address is empty")
	}

	// Handle ":port" format (empty host means all interfaces).
	if strings.HasPrefix(addr, ":") {
		port, err := strconv.ParseUint(addr[1:], 10, 16)
		if err != nil {
			return "", 0, fmt.Errorf("invalid port in %q: %w", addr, err)
		}
		return "", uint16(port), nil
	}

	host, portStr, err := splitLastColon(addr)
	if err != nil {
		return "", 0, err
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in %q: %w", addr, err)
	}

	return host, uint16(port), nil
}

func splitLastColon(s string) (string, string, error) {
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return "", "", fmt.Errorf("address %q missing port", s)
	}
	return s[:idx], s[idx+1:], nil
}
