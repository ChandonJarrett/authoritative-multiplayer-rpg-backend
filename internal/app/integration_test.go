//go:build integration

package app_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/nhh/go-enet"
	"google.golang.org/protobuf/proto"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/app"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/config"
	rpgv1 "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1"
	rpgv1connect "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/protocol/rpg/v1/rpgv1connect"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/testutil"
)

const (
	testAPIHTTPAddr  = ":19876"
	testGameHTTPAddr = ":19877"
	testGameENetAddr = ":17777"
	testGameENetPort = 17777
)

// testEnv returns a minimal valid configuration for integration tests.
// Postgres and Redis values are read from the process environment.
func testEnv() config.MapEnv {
	os := config.OSEnv{}
	return config.MapEnv{
		"APP_NAME":          "rpg-test",
		"APP_ENV":           "testing",
		"LOG_LEVEL":         "warn",
		"LOG_FORMAT":        "text",
		"API_HTTP_ADDR":     testAPIHTTPAddr,
		"GAME_ENET_ADDR":    testGameENetAddr,
		"GAME_HTTP_ADDR":    testGameHTTPAddr,
		"POSTGRES_HOST":     lookupEnvOrDefault(os, "POSTGRES_HOST", "localhost"),
		"POSTGRES_PORT":     lookupEnvOrDefault(os, "POSTGRES_PORT", "5432"),
		"POSTGRES_USER":     lookupEnvOrDefault(os, "POSTGRES_USER", "postgres"),
		"POSTGRES_PASSWORD": lookupEnvOrDefault(os, "POSTGRES_PASSWORD", "postgres"),
		"POSTGRES_DB":       lookupEnvOrDefault(os, "POSTGRES_DB", "rpg"),
		"REDIS_HOST":        lookupEnvOrDefault(os, "REDIS_HOST", "localhost"),
		"REDIS_PORT":        lookupEnvOrDefault(os, "REDIS_PORT", "6379"),
		"REDIS_PASSWORD":    lookupEnvOrDefault(os, "REDIS_PASSWORD", ""),
	}
}

// lookupEnvOrDefault returns the environment variable value if set, otherwise the default.
func lookupEnvOrDefault(source config.EnvSource, key, def string) string {
	if v, ok := source.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// TestFullGameServerHandoff exercises the full API-to-game-server flow:
//  1. API: register -> login -> create character -> list game servers -> issue join token
//  2. Game server: ENet connect -> send JoinRequest -> receive JoinResponse
//  3. Verification: health endpoints, Redis handoff state, successful join
func TestFullGameServerHandoff(t *testing.T) {
	// -- Load config ------------------------------------------------
	cfg, err := config.LoadWithSource(testEnv())
	if err != nil {
		t.Fatalf("load test config: %v", err)
	}

	// -- Create shared runtime --------------------------------------
	rt, err := app.NewRuntimeWithDeps("integration-test", app.RuntimeDeps{
		LoadConfig: func() (config.Config, error) { return cfg, nil },
	})
	if err != nil {
		testutil.SkipOnServiceErrorf(t, err, "create runtime")
		return
	}
	defer rt.Close()

	t.Log("runtime initialized")

	// -- Create API server -----------------------------------------
	apiSrv, err := app.NewAPIServer(rt)
	if err != nil {
		t.Fatalf("create api server: %v", err)
	}

	apiCtx, cancelAPI := context.WithCancel(rt.Context)
	defer cancelAPI()

	apiErrCh := make(chan error, 1)
	go func() {
		apiErrCh <- apiSrv.Run(apiCtx)
	}()

	t.Log("api server started on", cfg.APIHTTPAddr)

	// -- Create game server ----------------------------------------
	gameSrv, err := app.NewGameServer(rt)
	if err != nil {
		t.Fatalf("create game server: %v", err)
	}

	gameCtx, cancelGame := context.WithCancel(rt.Context)
	defer cancelGame()

	gameErrCh := make(chan error, 1)
	go func() {
		gameErrCh <- gameSrv.Run(gameCtx)
	}()

	t.Log("game server started on", cfg.GameENetAddr)

	// -- Wait for readiness ----------------------------------------
	waitReady(t, fmt.Sprintf("http://localhost%s/healthz", testAPIHTTPAddr), 5*time.Second)
	waitReady(t, fmt.Sprintf("http://localhost%s/healthz", testGameHTTPAddr), 5*time.Second)

	t.Log("both servers ready")

	// -- API flow: register -> create character -> issue join token --
	httpClient := &http.Client{Timeout: 10 * time.Second}
	apiBaseURL := fmt.Sprintf("http://localhost%s", testAPIHTTPAddr)

	authClient := rpgv1connect.NewAuthServiceClient(httpClient, apiBaseURL)
	charClient := rpgv1connect.NewCharacterServiceClient(httpClient, apiBaseURL)
	gameClient := rpgv1connect.NewGameServiceClient(httpClient, apiBaseURL)

	testEmail := fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
	testPassword := "test-password-123"

	// Register
	regResp, err := authClient.Register(rt.Context, connect.NewRequest(&rpgv1.RegisterRequest{
		Email:    testEmail,
		Password: testPassword,
	}))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	userID := regResp.Msg.UserId
	sessionToken := regResp.Msg.SessionToken
	t.Logf("registered user %s", userID)

	if userID == "" {
		t.Fatal("expected non-empty user ID")
	}
	if sessionToken == "" {
		t.Fatal("expected non-empty session token")
	}

	// Create character
	createReq := connect.NewRequest(&rpgv1.CreateCharacterRequest{
		Name: "TestHero",
	})
	createReq.Header().Set("Authorization", "Bearer "+sessionToken)

	createResp, err := charClient.CreateCharacter(rt.Context, createReq)
	if err != nil {
		t.Fatalf("create character: %v", err)
	}
	characterID := createResp.Msg.CharacterId
	t.Logf("created character %s", characterID)

	if characterID == "" {
		t.Fatal("expected non-empty character ID")
	}

	// List game servers
	listReq := connect.NewRequest(&rpgv1.ListGameServersRequest{})
	listReq.Header().Set("Authorization", "Bearer "+sessionToken)

	listResp, err := gameClient.ListGameServers(rt.Context, listReq)
	if err != nil {
		t.Fatalf("list game servers: %v", err)
	}
	if len(listResp.Msg.Servers) == 0 {
		t.Fatal("expected at least one game server")
	}
	gameServerID := listResp.Msg.Servers[0].GameServerId
	t.Logf("found game server %s", gameServerID)

	// Issue join token
	issueReq := connect.NewRequest(&rpgv1.IssueJoinTokenRequest{
		CharacterId:  characterID,
		GameServerId: gameServerID,
	})
	issueReq.Header().Set("Authorization", "Bearer "+sessionToken)

	issueResp, err := gameClient.IssueJoinToken(rt.Context, issueReq)
	if err != nil {
		t.Fatalf("issue join token: %v", err)
	}
	joinToken := issueResp.Msg.JoinToken
	t.Logf("issued join token (len=%d, expires=%ds)", len(joinToken), issueResp.Msg.ExpiresInSeconds)

	if joinToken == "" {
		t.Fatal("expected non-empty join token")
	}

	// -- ENet client: connect to game server, send join request ----
	t.Log("connecting to game server via ENet...")

	enetHost, err := enet.NewHost(nil, 1, 2, 0, 0)
	if err != nil {
		t.Fatalf("create enet client host: %v", err)
	}

	address := enet.NewAddress("127.0.0.1", testGameENetPort)
	peer, err := enetHost.Connect(address, 2, 0)
	if err != nil {
		t.Fatalf("enet connect: %v", err)
	}
	if peer == nil {
		t.Fatal("enet connect returned nil peer")
	}

	// Wait for connect event.
	connected := false
	for i := 0; i < 50; i++ {
		evt := enetHost.Service(100)
		if evt == nil {
			continue
		}
		if evt.GetType() == enet.EventConnect {
			connected = true
			t.Log("enet client connected")
			break
		}
	}
	if !connected {
		t.Fatal("enet client did not receive connect event")
	}

	// Send JoinRequest.
	joinReq := &rpgv1.JoinRequest{
		JoinToken:   joinToken,
		CharacterId: characterID,
	}
	joinReqData, err := proto.Marshal(joinReq)
	if err != nil {
		t.Fatalf("marshal join request: %v", err)
	}

	if err := peer.SendBytes(joinReqData, 0, enet.PacketFlagReliable); err != nil {
		t.Fatalf("send join request: %v", err)
	}
	t.Log("sent join request")

	// Wait for JoinResponse.
	var joinResp rpgv1.JoinResponse
	gotResponse := false
	for i := 0; i < 100; i++ {
		evt := enetHost.Service(100)
		if evt == nil {
			continue
		}
		switch evt.GetType() {
		case enet.EventReceive:
			data := evt.GetPacket().GetData()
			if err := proto.Unmarshal(data, &joinResp); err != nil {
				evt.GetPacket().Destroy()
				t.Fatalf("unmarshal join response: %v", err)
			}
			evt.GetPacket().Destroy()
			gotResponse = true
			t.Logf("received join response: ok=%v", joinResp.Ok)
		case enet.EventDisconnect:
			t.Log("enet client disconnected by server")
		}
		if gotResponse {
			break
		}
	}
	if !gotResponse {
		t.Fatal("did not receive join response")
	}

	if !joinResp.Ok {
		t.Fatalf("join rejected: %s", joinResp.Reason)
	}

	t.Log("join successful - full API-to-game-server handoff verified")

	// -- World state verification: wait for initial snapshot ------
	entityID := "char_" + characterID

	initialSnap, err := recvSnapshot(t, enetHost, 3*time.Second)
	if err != nil {
		t.Fatalf("initial snapshot: %v", err)
	}
	t.Logf("initial snapshot: tick=%d, entities=%d", initialSnap.Tick, len(initialSnap.Entities))

	var found *rpgv1.EntitySnapshot
	for _, e := range initialSnap.Entities {
		if e.EntityId == entityID {
			found = e
			break
		}
	}
	if found == nil {
		t.Fatalf("character entity %s not found in snapshot - world state verification failed", entityID)
	}
	t.Logf("world state verified: entity %s at spawn (%.4f, %.4f)", entityID, found.Position.X, found.Position.Y)

	// -- Input + snapshot test: send movement, verify position ---
	movement, err := proto.Marshal(&rpgv1.InputPacket{
		Sequence:      1,
		MoveDirection: &rpgv1.Vec2{X: 1.0, Y: 0.0},
	})
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	if err := peer.SendBytes(movement, 1, enet.PacketFlagUnsequenced); err != nil {
		t.Fatalf("send movement input: %v", err)
	}
	t.Log("sent movement input (right)")

	movedSnap, err := recvSnapshot(t, enetHost, 3*time.Second)
	if err != nil {
		t.Fatalf("post-movement snapshot: %v", err)
	}
	t.Logf("post-movement snapshot: tick=%d, entities=%d", movedSnap.Tick, len(movedSnap.Entities))

	var movedFound bool
	for _, e := range movedSnap.Entities {
		if e.EntityId == entityID {
			movedFound = true
			t.Logf("entity %s moved to (%.4f, %.4f)", entityID, e.Position.X, e.Position.Y)
			if e.Position.X <= 0 {
				t.Errorf("expected entity to have moved right (X > 0), got X=%.4f", e.Position.X)
			}
			break
		}
	}
	if !movedFound {
		t.Fatalf("character entity %s not found in post-movement snapshot", entityID)
	}
	t.Log("input + snapshot test passed")

	// -- Disconnect and cleanup ----------------------------------
	peer.Disconnect(0)

	// Drain disconnect event.
	for i := 0; i < 20; i++ {
		evt := enetHost.Service(50)
		if evt == nil {
			continue
		}
		if evt.GetType() == enet.EventDisconnect {
			t.Log("enet client disconnected cleanly")
			break
		}
	}

	// Destroy client host before shutting down game server.
	enetHost.Destroy()

	// Shut down both servers.
	cancelAPI()
	cancelGame()

	// Wait for servers to stop (drain error channels).
	select {
	case err := <-apiErrCh:
		if err != nil {
			t.Errorf("api server error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("api server did not shut down in time")
	}

	select {
	case err := <-gameErrCh:
		if err != nil {
			t.Errorf("game server error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("game server did not shut down in time")
	}

	t.Log("both servers shut down cleanly")
}

// waitReady polls a URL until it returns 200 OK or the timeout expires.
func waitReady(t *testing.T, url string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("service at %s did not become ready within %v", url, timeout)
}

// recvSnapshot waits for a SnapshotPacket on the unreliable ENet channel.
func recvSnapshot(t *testing.T, host enet.Host, timeout time.Duration) (*rpgv1.SnapshotPacket, error) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		evt := host.Service(50)
		if evt == nil {
			continue
		}
		switch evt.GetType() {
		case enet.EventReceive:
			pkt := evt.GetPacket()
			if evt.GetChannelID() != 1 {
				pkt.Destroy()
				continue
			}
			data := pkt.GetData()
			var snap rpgv1.SnapshotPacket
			if err := proto.Unmarshal(data, &snap); err != nil {
				pkt.Destroy()
				continue
			}
			pkt.Destroy()
			return &snap, nil
		case enet.EventDisconnect:
			return nil, fmt.Errorf("unexpected disconnect while waiting for snapshot")
		}
	}
	return nil, fmt.Errorf("timed out waiting for snapshot")
}
