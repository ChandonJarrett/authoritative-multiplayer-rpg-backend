package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/auth"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/validate"
)

// GameService provides game-related operations.
type GameService struct {
	characters CharacterStore
	joinTokens JoinTokenStore
	servers    GameServerStore
}

// IssueJoinTokenResult represents the result of issuing a join token.
type IssueJoinTokenResult struct {
	JoinToken        string
	GameServerID     string
	GameServerAddr   string
	ExpiresInSeconds int64
}

// NewGameService creates a new instance of GameService with the provided dependencies.
func NewGameService(
	characters CharacterStore,
	joinTokens JoinTokenStore,
	servers GameServerStore,
) (*GameService, error) {
	if characters == nil {
		return nil, fmt.Errorf("game service character store: %w", domain.ErrInternal)
	}
	if joinTokens == nil {
		return nil, fmt.Errorf("game service join token store: %w", domain.ErrInternal)
	}
	if servers == nil {
		return nil, fmt.Errorf("game service server store: %w", domain.ErrInternal)
	}

	return &GameService{
		characters: characters,
		joinTokens: joinTokens,
		servers:    servers,
	}, nil
}

// ListGameServers returns the currently registered, non-expired game servers.
func (s *GameService) ListGameServers(ctx context.Context) ([]domain.GameServer, error) {
	if s == nil {
		return nil, domain.ErrInternal
	}

	servers, err := s.servers.ListGameServers(ctx)
	if err != nil {
		return nil, err
	}

	return servers, nil
}

// IssueJoinToken creates a short-lived join token for one character and one game server.
// If gameServerID is empty, it auto-picks the first available server for backward compatibility.
func (s *GameService) IssueJoinToken(
	ctx context.Context,
	userID string,
	characterID string,
	gameServerID string,
) (IssueJoinTokenResult, error) {
	if s == nil {
		return IssueJoinTokenResult{}, domain.ErrInternal
	}

	userID, err := validate.RequiredID("user ID", userID)
	if err != nil {
		return IssueJoinTokenResult{}, err
	}

	characterID, err = validate.RequiredID("character ID", characterID)
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

	server, err := s.selectServer(ctx, gameServerID)
	if err != nil {
		return IssueJoinTokenResult{}, err
	}

	token, err := auth.NewJoinToken()
	if err != nil {
		return IssueJoinTokenResult{}, err
	}

	expiresAt := time.Now().UTC().Add(cache.DefaultJoinTokenTTL)

	joinToken := domain.JoinToken{
		Token:       token,
		UserID:      userID,
		CharacterID: characterID,
		ServerID:    server.ID,
		ServerAddr:  server.Addr,
		ExpiresAt:   expiresAt,
	}

	if err := s.joinTokens.CreateJoinToken(ctx, joinToken); err != nil {
		return IssueJoinTokenResult{}, err
	}

	return IssueJoinTokenResult{
		JoinToken:        token,
		GameServerID:     server.ID,
		GameServerAddr:   server.Addr,
		ExpiresInSeconds: int64(cache.DefaultJoinTokenTTL.Seconds()),
	}, nil
}

func (s *GameService) selectServer(ctx context.Context, requestedServerID string) (domain.GameServer, error) {
	servers, err := s.servers.ListGameServers(ctx)
	if err != nil {
		return domain.GameServer{}, err
	}

	if len(servers) == 0 {
		return domain.GameServer{}, domain.ErrUnavailable
	}

	if requestedServerID == "" {
		return servers[0], nil
	}

	requestedServerID, err = validate.RequiredID("game server ID", requestedServerID)
	if err != nil {
		return domain.GameServer{}, err
	}

	for _, server := range servers {
		if server.ID == requestedServerID {
			return server, nil
		}
	}

	return domain.GameServer{}, domain.ErrNotFound
}
