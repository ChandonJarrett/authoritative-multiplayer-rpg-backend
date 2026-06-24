package game

import (
	"sync"

	rpgv1 "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1"

	"github.com/nhh/go-enet"
)

// Session represents a connected client playing with a specific character.
type Session struct {
	Peer        enet.Peer
	UserID      string
	CharacterID string
	EntityID    string

	// LatestInput is the most recent input packet received from the client.
	LatestInput *rpgv1.InputPacket
}

// SessionManager tracks all active game sessions keyed by ENet peer.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[enet.Peer]*Session
}

// NewSessionManager creates an empty session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[enet.Peer]*Session),
	}
}

// Add registers a new session for the given peer.
func (sm *SessionManager) Add(session *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[session.Peer] = session
}

// Get returns the session for the given peer, or nil if not found.
func (sm *SessionManager) Get(peer enet.Peer) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[peer]
}

// Remove deletes the session for the given peer. Returns the removed session or nil.
func (sm *SessionManager) Remove(peer enet.Peer) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sess, ok := sm.sessions[peer]
	if ok {
		delete(sm.sessions, peer)
	}
	return sess
}

// All returns a snapshot of all active sessions.
func (sm *SessionManager) All() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// Count returns the number of active sessions.
func (sm *SessionManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}
