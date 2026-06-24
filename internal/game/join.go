package game

import (
	"context"
	"log/slog"
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	rpgv1 "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store"
	"github.com/nhh/go-enet"
)

// JoinTokenStore is the subset of store.JoinTokenStore needed by the game server.
type joinTokenStore interface {
	ConsumeJoinToken(ctx context.Context, token string) (domain.JoinToken, error)
}

// CharacterLocker manages character locks for active game sessions.
type characterLocker interface {
	AcquireCharacterLock(ctx context.Context, characterID, ownerID string, ttl time.Duration) (bool, error)
	ReleaseCharacterLock(ctx context.Context, characterID, ownerID string) (bool, error)
}

// CharacterProvider loads character data from durable storage.
type characterProvider interface {
	GetCharacterByID(ctx context.Context, characterID string) (domain.Character, error)
}

// JoinHandler processes ENet join requests and manages the join handshake.
type JoinHandler struct {
	log            *slog.Logger
	joinTokens     joinTokenStore
	characterLocks characterLocker
	characters     characterProvider
	world          *World
	sessions       *SessionManager
	lockTTL        time.Duration
	serverID       string
}

// NewJoinHandler creates a join handshake handler.
func NewJoinHandler(
	log *slog.Logger,
	joinTokens store.JoinTokenStore,
	characterLocks characterLocker,
	characters store.CharacterStore,
	world *World,
	sessions *SessionManager,
	lockTTL time.Duration,
	serverID string,
) *JoinHandler {
	return &JoinHandler{
		log:            log,
		joinTokens:     joinTokens,
		characterLocks: characterLocks,
		characters:     characters,
		world:          world,
		sessions:       sessions,
		lockTTL:        lockTTL,
		serverID:       serverID,
	}
}

// HandleJoinRequest processes a JoinRequest from a newly connected peer.
// On success, it creates a session and adds the character entity to the world.
// On failure, it sends a JoinResponse with ok=false and disconnects the peer.
func (h *JoinHandler) HandleJoinRequest(ctx context.Context, peer enet.Peer, data []byte) {
	log := h.log.With("peer_addr", peer.GetAddress().String())

	req, err := unmarshalJoinRequest(data)
	if err != nil {
		log.Warn("failed to unmarshal join request", "error", err)
		h.sendJoinResponse(peer, false, "invalid join request")
		return
	}

	log = log.With("character_id", req.GetCharacterId())

	// 1. Consume the join token (single-use).
	joinToken, err := h.joinTokens.ConsumeJoinToken(ctx, req.GetJoinToken())
	if err != nil {
		log.Warn("failed to consume join token", "error", err)
		h.sendJoinResponse(peer, false, "invalid or expired join token")
		return
	}

	if joinToken.CharacterID != req.GetCharacterId() {
		log.Warn(
			"character ID mismatch",
			"token_character_id", joinToken.CharacterID,
			"request_character_id", req.GetCharacterId(),
		)
		h.sendJoinResponse(peer, false, "character ID mismatch")
		return
	}

	if joinToken.ServerID != h.serverID {
		log.Warn(
			"server ID mismatch",
			"token_server_id", joinToken.ServerID,
			"this_server_id", h.serverID,
		)
		h.sendJoinResponse(peer, false, "join token is for a different server")
		return
	}

	// 2. Acquire character lock to prevent double-join.
	acquired, err := h.characterLocks.AcquireCharacterLock(ctx, joinToken.CharacterID, h.serverID, h.lockTTL)
	if err != nil {
		log.Error("failed to acquire character lock", "error", err)
		h.sendJoinResponse(peer, false, "internal error")
		return
	}
	if !acquired {
		log.Warn("character already locked")
		h.sendJoinResponse(peer, false, "character is already in-game")
		return
	}

	// 3. Load character from PostgreSQL.
	character, err := h.characters.GetCharacterByID(ctx, joinToken.CharacterID)
	if err != nil {
		log.Error("failed to load character", "error", err)
		// Release the lock we just acquired since the load failed.
		_, _ = h.characterLocks.ReleaseCharacterLock(ctx, joinToken.CharacterID, h.serverID)
		h.sendJoinResponse(peer, false, "failed to load character")
		return
	}

	// 4. Add entity to the world.
	entityID := "char_" + character.ID
	entity := &Entity{
		ID:       entityID,
		Type:     EntityTypePlayer,
		Position: character.Position,
	}
	h.world.AddEntity(entity)

	// 5. Register the session.
	session := &Session{
		Peer:        peer,
		UserID:      joinToken.UserID,
		CharacterID: character.ID,
		EntityID:    entityID,
	}
	h.sessions.Add(session)

	log.Info(
		"player joined",
		"user_id", joinToken.UserID,
		"entity_id", entityID,
	)

	// 6. Send success response.
	h.sendJoinResponse(peer, true, "")
}

// HandleDisconnect cleans up when a peer disconnects.
func (h *JoinHandler) HandleDisconnect(ctx context.Context, peer enet.Peer) {
	session := h.sessions.Remove(peer)
	if session == nil {
		return
	}

	log := h.log.With(
		"user_id", session.UserID,
		"character_id", session.CharacterID,
		"entity_id", session.EntityID,
	)

	// Remove entity from world.
	if session.EntityID != "" {
		entity := h.world.GetEntity(session.EntityID)
		if entity != nil {
			h.world.RemoveEntity(entity)
		}
	}

	// Release character lock.
	released, err := h.characterLocks.ReleaseCharacterLock(ctx, session.CharacterID, h.serverID)
	if err != nil {
		log.Error("failed to release character lock on disconnect", "error", err)
	} else if released {
		log.Info("character lock released")
	}

	log.Info("player disconnected")
}

func (h *JoinHandler) sendJoinResponse(peer enet.Peer, ok bool, reason string) {
	resp := &rpgv1.JoinResponse{
		Ok:     ok,
		Reason: reason,
	}

	data, err := marshalJoinResponse(resp)
	if err != nil {
		h.log.Error("failed to marshal join response", "error", err)
		peer.Disconnect(0)
		return
	}

	if err := peer.SendBytes(data, channelReliable, enet.PacketFlagReliable); err != nil {
		h.log.Error("failed to send join response", "error", err)
	}

	if !ok {
		peer.Disconnect(0)
	}
}
