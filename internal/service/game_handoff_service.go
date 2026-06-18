package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/auth"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/validate"
)

// GameHandoffService handles the logic for issuing join tokens to clients for game servers.
type GameHandoffService struct {
	characters store.CharacterStore
	joinTokens store.JoinTokenStore
	servers    store.GameServerStore
	now        func() time.Time
}

// NewGameHandoffService creates a new GameHandoffService with the given dependencies.
func NewGameHandoffService(
	characters store.CharacterStore,
	joinTokens store.JoinTokenStore,
	servers store.GameServerStore,
) *GameHandoffService {
	return &GameHandoffService{
		characters: characters,
		joinTokens: joinTokens,
		servers:    servers,
		now:        time.Now,
	}
}

// IssueJoinTokenResult contains the result of issuing a join token, including the token and game server info.
type IssueJoinTokenResult struct {
	JoinToken        string
	GameServerID     string
	GameServerAddr   string
	ExpiresInSeconds int64
}

// IssueJoinToken validates the user's request to join a game, selects a game server, creates a join token, and returns the token and server info.
func (s *GameHandoffService) IssueJoinToken(
	ctx context.Context,
	userID string,
	characterID string,
) (IssueJoinTokenResult, error) {
	characterID, err := validate.RequiredID("character_id", characterID)
	if err != nil {
		return IssueJoinTokenResult{}, err
	}

	character, err := s.characters.GetCharacterByID(ctx, characterID)
	if err != nil {
		return IssueJoinTokenResult{}, err
	}

	if character.UserID != userID {
		return IssueJoinTokenResult{}, domain.ErrPermissionDenied
	}

	servers, err := s.servers.ListGameServers(ctx)
	if err != nil {
		return IssueJoinTokenResult{}, err
	}
	if len(servers) == 0 {
		return IssueJoinTokenResult{}, domain.ErrUnavailable
	}

	selected := servers[0]

	token, err := auth.NewJoinToken()
	if err != nil {
		return IssueJoinTokenResult{}, err
	}

	joinToken := domain.JoinToken{
		Token:       token,
		UserID:      userID,
		CharacterID: characterID,
		ServerID:    selected.ID,
		ServerAddr:  selected.Addr,
		ExpiresAt:   s.now().UTC().Add(cache.DefaultJoinTokenTTL),
	}

	if err := s.joinTokens.CreateJoinToken(ctx, joinToken); err != nil {
		return IssueJoinTokenResult{}, fmt.Errorf("create join token: %w", err)
	}

	return IssueJoinTokenResult{
		JoinToken:        token,
		GameServerID:     selected.ID,
		GameServerAddr:   selected.Addr,
		ExpiresInSeconds: int64(cache.DefaultJoinTokenTTL.Seconds()),
	}, nil
}
