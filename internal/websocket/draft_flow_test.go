package websocket_test

import (
	"testing"
	"time"

	"github.com/dom/league-draft-website/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultTimeout is used for all message expectations in tests
const defaultTimeout = 5 * time.Second

func TestDraftFlow_JoinRoom(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create and authenticate users
	_, blueToken := testutil.NewUserBuilder().
		WithDisplayName("bluePlayer").
		BuildAndAuthenticate(t, ts)

	// Create a room with WebSocket hub
	room := testutil.NewRoomBuilder().BuildWithHub(t, ts)

	// Seed champions
	testutil.SeedRealChampions(t, ts.DB.DB)

	// Connect WebSocket client
	wsClient := testutil.NewWSClient(t, ts.WebSocketURL(blueToken))

	// Join room
	wsClient.JoinRoom(room.ID.String(), "blue")

	// Expect state sync
	stateSync := wsClient.ExpectStateSync(defaultTimeout)

	assert.Equal(t, room.ID.String(), stateSync.Room.ID)
	assert.Equal(t, "waiting", stateSync.Room.Status)
	assert.Equal(t, "blue", stateSync.YourSide)
}

func TestDraftFlow_ReadyUp(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create users
	_, blueToken := testutil.NewUserBuilder().
		WithDisplayName("bluePlayer").
		BuildAndAuthenticate(t, ts)

	_, redToken := testutil.NewUserBuilder().
		WithDisplayName("redPlayer").
		BuildAndAuthenticate(t, ts)

	// Create room
	room := testutil.NewRoomBuilder().BuildWithHub(t, ts)
	testutil.SeedRealChampions(t, ts.DB.DB)

	// Connect both players
	blueClient := testutil.NewWSClient(t, ts.WebSocketURL(blueToken))
	redClient := testutil.NewWSClient(t, ts.WebSocketURL(redToken))

	// Join room
	blueClient.JoinRoom(room.ID.String(), "blue")
	blueClient.ExpectStateSync(defaultTimeout)

	redClient.JoinRoom(room.ID.String(), "red")
	redClient.ExpectStateSync(defaultTimeout)

	// Drain join notifications - DrainMessages now properly waits for messages to settle
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Blue player readies up
	blueClient.Ready(true)
	blueUpdate := blueClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)
	assert.Equal(t, "ready_changed", blueUpdate.Action)
	assert.True(t, blueUpdate.Player.Ready)

	// Red should also receive blue's ready notification
	redClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)

	// Drain any extra messages before red readies
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Red player readies up
	redClient.Ready(true)
	redUpdate := redClient.ExpectPlayerUpdateForSide("red", defaultTimeout)
	assert.Equal(t, "ready_changed", redUpdate.Action)
	assert.True(t, redUpdate.Player.Ready)

	// Blue should also receive red's ready notification
	blueClient.ExpectPlayerUpdateForSide("red", defaultTimeout)
}

func TestDraftFlow_StartDraft(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create users
	_, blueToken := testutil.NewUserBuilder().
		WithDisplayName("bluePlayer").
		BuildAndAuthenticate(t, ts)

	_, redToken := testutil.NewUserBuilder().
		WithDisplayName("redPlayer").
		BuildAndAuthenticate(t, ts)

	// Create room
	room := testutil.NewRoomBuilder().BuildWithHub(t, ts)
	testutil.SeedRealChampions(t, ts.DB.DB)

	// Connect and join
	blueClient := testutil.NewWSClient(t, ts.WebSocketURL(blueToken))
	redClient := testutil.NewWSClient(t, ts.WebSocketURL(redToken))

	blueClient.JoinRoom(room.ID.String(), "blue")
	blueClient.ExpectStateSync(defaultTimeout)

	redClient.JoinRoom(room.ID.String(), "red")
	redClient.ExpectStateSync(defaultTimeout)

	// Drain join notifications
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Both ready up - expect player updates to broadcast
	blueClient.Ready(true)
	blueClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)
	redClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)

	redClient.Ready(true)
	redClient.ExpectPlayerUpdateForSide("red", defaultTimeout)
	blueClient.ExpectPlayerUpdateForSide("red", defaultTimeout)

	// Start draft
	blueClient.StartDraft()

	// Both clients should receive draft started
	draftStarted := blueClient.ExpectDraftStarted(defaultTimeout)
	assert.Equal(t, 0, draftStarted.CurrentPhase)
	assert.Equal(t, "blue", draftStarted.CurrentTeam)
	assert.Equal(t, "ban", draftStarted.ActionType)

	// Red should also receive draft started
	redDraftStarted := redClient.ExpectDraftStarted(defaultTimeout)
	assert.Equal(t, 0, redDraftStarted.CurrentPhase)
}

func TestDraftFlow_SelectAndLockIn(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create users
	_, blueToken := testutil.NewUserBuilder().
		WithDisplayName("bluePlayer").
		BuildAndAuthenticate(t, ts)

	_, redToken := testutil.NewUserBuilder().
		WithDisplayName("redPlayer").
		BuildAndAuthenticate(t, ts)

	// Create room
	room := testutil.NewRoomBuilder().BuildWithHub(t, ts)
	champions := testutil.SeedRealChampions(t, ts.DB.DB)

	// Connect and join
	blueClient := testutil.NewWSClient(t, ts.WebSocketURL(blueToken))
	redClient := testutil.NewWSClient(t, ts.WebSocketURL(redToken))

	blueClient.JoinRoom(room.ID.String(), "blue")
	blueClient.ExpectStateSync(defaultTimeout)

	redClient.JoinRoom(room.ID.String(), "red")
	redClient.ExpectStateSync(defaultTimeout)

	// Drain join notifications
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Ready up - wait for broadcasts
	blueClient.Ready(true)
	blueClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)
	redClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)

	redClient.Ready(true)
	redClient.ExpectPlayerUpdateForSide("red", defaultTimeout)
	blueClient.ExpectPlayerUpdateForSide("red", defaultTimeout)

	// Start draft
	blueClient.StartDraft()
	blueClient.ExpectDraftStarted(defaultTimeout)
	redClient.ExpectDraftStarted(defaultTimeout)

	// Blue selects champion (phase 0 - blue ban)
	blueClient.SelectChampion(champions[0].ID)

	// Lock in
	blueClient.LockIn()

	// Both clients should receive champion selected
	selected := blueClient.ExpectChampionSelected(defaultTimeout)
	assert.Equal(t, 0, selected.Phase)
	assert.Equal(t, "blue", selected.Team)
	assert.Equal(t, "ban", selected.ActionType)
	assert.Equal(t, champions[0].ID, selected.ChampionID)

	// Red should also receive champion selected
	redClient.ExpectChampionSelected(defaultTimeout)

	// Both clients should receive phase changed
	phaseChanged := blueClient.ExpectPhaseChanged(defaultTimeout)
	assert.Equal(t, 1, phaseChanged.CurrentPhase)
	assert.Equal(t, "red", phaseChanged.CurrentTeam)
	assert.Equal(t, "ban", phaseChanged.ActionType)

	// Red should also receive phase changed
	redClient.ExpectPhaseChanged(defaultTimeout)
}

func TestDraftFlow_WrongTurnError(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create users
	_, blueToken := testutil.NewUserBuilder().
		WithDisplayName("bluePlayer").
		BuildAndAuthenticate(t, ts)

	_, redToken := testutil.NewUserBuilder().
		WithDisplayName("redPlayer").
		BuildAndAuthenticate(t, ts)

	// Create room
	room := testutil.NewRoomBuilder().BuildWithHub(t, ts)
	champions := testutil.SeedRealChampions(t, ts.DB.DB)

	// Connect and join
	blueClient := testutil.NewWSClient(t, ts.WebSocketURL(blueToken))
	redClient := testutil.NewWSClient(t, ts.WebSocketURL(redToken))

	blueClient.JoinRoom(room.ID.String(), "blue")
	blueClient.ExpectStateSync(defaultTimeout)

	redClient.JoinRoom(room.ID.String(), "red")
	redClient.ExpectStateSync(defaultTimeout)

	// Drain join notifications
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Ready up - wait for broadcasts
	blueClient.Ready(true)
	blueClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)
	redClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)

	redClient.Ready(true)
	redClient.ExpectPlayerUpdateForSide("red", defaultTimeout)
	blueClient.ExpectPlayerUpdateForSide("red", defaultTimeout)

	// Start draft and wait for both clients to receive it
	blueClient.StartDraft()
	blueClient.ExpectDraftStarted(defaultTimeout)
	redClient.ExpectDraftStarted(defaultTimeout)

	// Red tries to select on blue's turn (phase 0 is blue's turn)
	redClient.SelectChampion(champions[0].ID)

	// Expect error
	errorPayload := redClient.ExpectErrorWithCode("NOT_YOUR_TURN", defaultTimeout)
	assert.Contains(t, errorPayload.Message, "not your turn")
}

func TestDraftFlow_ChampionUnavailable(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create users
	_, blueToken := testutil.NewUserBuilder().
		WithDisplayName("bluePlayer").
		BuildAndAuthenticate(t, ts)

	_, redToken := testutil.NewUserBuilder().
		WithDisplayName("redPlayer").
		BuildAndAuthenticate(t, ts)

	// Create room
	room := testutil.NewRoomBuilder().BuildWithHub(t, ts)
	champions := testutil.SeedRealChampions(t, ts.DB.DB)

	// Connect and start draft
	blueClient := testutil.NewWSClient(t, ts.WebSocketURL(blueToken))
	redClient := testutil.NewWSClient(t, ts.WebSocketURL(redToken))

	blueClient.JoinRoom(room.ID.String(), "blue")
	blueClient.ExpectStateSync(defaultTimeout)

	redClient.JoinRoom(room.ID.String(), "red")
	redClient.ExpectStateSync(defaultTimeout)

	// Drain join notifications
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Ready up - wait for broadcasts
	blueClient.Ready(true)
	blueClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)
	redClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)

	redClient.Ready(true)
	redClient.ExpectPlayerUpdateForSide("red", defaultTimeout)
	blueClient.ExpectPlayerUpdateForSide("red", defaultTimeout)

	// Start draft
	blueClient.StartDraft()
	blueClient.ExpectDraftStarted(defaultTimeout)
	redClient.ExpectDraftStarted(defaultTimeout)

	// Blue bans a champion
	blueClient.SelectChampion(champions[0].ID)
	blueClient.LockIn()

	// Both clients receive champion selected and phase changed
	blueClient.ExpectChampionSelected(defaultTimeout)
	redClient.ExpectChampionSelected(defaultTimeout)
	blueClient.ExpectPhaseChanged(defaultTimeout)
	redClient.ExpectPhaseChanged(defaultTimeout)

	// Red tries to ban the same champion
	redClient.SelectChampion(champions[0].ID)

	// Expect error
	errorPayload := redClient.ExpectErrorWithCode("CHAMPION_UNAVAILABLE", defaultTimeout)
	assert.Contains(t, errorPayload.Message, "already picked or banned")
}

func TestDraftFlow_NoSelectionError(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create users
	_, blueToken := testutil.NewUserBuilder().
		WithDisplayName("bluePlayer").
		BuildAndAuthenticate(t, ts)

	_, redToken := testutil.NewUserBuilder().
		WithDisplayName("redPlayer").
		BuildAndAuthenticate(t, ts)

	// Create room
	room := testutil.NewRoomBuilder().BuildWithHub(t, ts)
	testutil.SeedRealChampions(t, ts.DB.DB)

	// Connect and start draft
	blueClient := testutil.NewWSClient(t, ts.WebSocketURL(blueToken))
	redClient := testutil.NewWSClient(t, ts.WebSocketURL(redToken))

	blueClient.JoinRoom(room.ID.String(), "blue")
	blueClient.ExpectStateSync(defaultTimeout)

	redClient.JoinRoom(room.ID.String(), "red")
	redClient.ExpectStateSync(defaultTimeout)

	// Drain join notifications
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Ready up - wait for broadcasts
	blueClient.Ready(true)
	blueClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)
	redClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)

	redClient.Ready(true)
	redClient.ExpectPlayerUpdateForSide("red", defaultTimeout)
	blueClient.ExpectPlayerUpdateForSide("red", defaultTimeout)

	// Start draft
	blueClient.StartDraft()
	blueClient.ExpectDraftStarted(defaultTimeout)
	redClient.ExpectDraftStarted(defaultTimeout)

	// Blue tries to lock in without selecting
	blueClient.LockIn()

	// Expect error
	errorPayload := blueClient.ExpectErrorWithCode("NO_SELECTION", defaultTimeout)
	assert.Contains(t, errorPayload.Message, "No champion selected")
}

func TestDraftFlow_FullDraft(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full draft test in short mode")
	}

	ts := testutil.NewTestServer(t)

	// Create users
	_, blueToken := testutil.NewUserBuilder().
		WithDisplayName("bluePlayer").
		BuildAndAuthenticate(t, ts)

	_, redToken := testutil.NewUserBuilder().
		WithDisplayName("redPlayer").
		BuildAndAuthenticate(t, ts)

	// Create room with fast timer
	room := testutil.NewRoomBuilder().
		WithTimerDuration(1). // 1 second timer for fast test
		BuildWithHub(t, ts)

	champions := testutil.SeedRealChampions(t, ts.DB.DB)
	require.GreaterOrEqual(t, len(champions), 20, "need at least 20 champions for full draft")

	// Connect and start draft
	blueClient := testutil.NewWSClient(t, ts.WebSocketURL(blueToken))
	redClient := testutil.NewWSClient(t, ts.WebSocketURL(redToken))

	blueClient.JoinRoom(room.ID.String(), "blue")
	blueClient.ExpectStateSync(defaultTimeout)

	redClient.JoinRoom(room.ID.String(), "red")
	redClient.ExpectStateSync(defaultTimeout)

	// Drain join notifications
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Ready up with proper synchronization
	blueClient.Ready(true)
	blueClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)
	redClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)

	redClient.Ready(true)
	redClient.ExpectPlayerUpdateForSide("red", defaultTimeout)
	blueClient.ExpectPlayerUpdateForSide("red", defaultTimeout)

	blueClient.StartDraft()
	blueClient.ExpectDraftStarted(defaultTimeout)
	redClient.ExpectDraftStarted(defaultTimeout)

	// Pro play phases: 20 total
	// Phases 0-5:   Ban (B-R-B-R-B-R)
	// Phases 6-11:  Pick (B-R-R-B-B-R)
	// Phases 12-15: Ban (R-B-R-B)
	// Phases 16-19: Pick (R-B-B-R)
	phaseTeams := []string{
		"blue", "red", "blue", "red", "blue", "red", // Bans 0-5
		"blue", "red", "red", "blue", "blue", "red", // Picks 6-11
		"red", "blue", "red", "blue", // Bans 12-15
		"red", "blue", "blue", "red", // Picks 16-19
	}

	championIndex := 0
	for phase := 0; phase < 20; phase++ {
		currentTeam := phaseTeams[phase]

		var activeClient *testutil.WSClient
		if currentTeam == "blue" {
			activeClient = blueClient
		} else {
			activeClient = redClient
		}

		// Select and lock in
		activeClient.SelectChampion(champions[championIndex].ID)
		activeClient.LockIn()
		championIndex++

		// Expect champion selected
		blueClient.ExpectChampionSelected(defaultTimeout)
		redClient.ExpectChampionSelected(defaultTimeout)

		if phase < 19 {
			// Expect phase changed
			blueClient.ExpectPhaseChanged(defaultTimeout)
			redClient.ExpectPhaseChanged(defaultTimeout)
		}
	}

	// Expect draft completed
	completed := blueClient.ExpectDraftCompleted(defaultTimeout)
	redClient.ExpectDraftCompleted(defaultTimeout)

	// Verify final state
	assert.Len(t, completed.BlueBans, 5)
	assert.Len(t, completed.RedBans, 5)
	assert.Len(t, completed.BluePicks, 5)
	assert.Len(t, completed.RedPicks, 5)
}

func TestDraftFlow_SpectatorView(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create users
	_, blueToken := testutil.NewUserBuilder().
		WithDisplayName("bluePlayer").
		BuildAndAuthenticate(t, ts)

	_, redToken := testutil.NewUserBuilder().
		WithDisplayName("redPlayer").
		BuildAndAuthenticate(t, ts)

	_, spectatorToken := testutil.NewUserBuilder().
		WithDisplayName("spectator").
		BuildAndAuthenticate(t, ts)

	// Create room
	room := testutil.NewRoomBuilder().BuildWithHub(t, ts)
	champions := testutil.SeedRealChampions(t, ts.DB.DB)

	// Connect players
	blueClient := testutil.NewWSClient(t, ts.WebSocketURL(blueToken))
	redClient := testutil.NewWSClient(t, ts.WebSocketURL(redToken))

	blueClient.JoinRoom(room.ID.String(), "blue")
	blueClient.ExpectStateSync(defaultTimeout)

	redClient.JoinRoom(room.ID.String(), "red")
	redClient.ExpectStateSync(defaultTimeout)

	// Connect spectator
	spectatorClient := testutil.NewWSClient(t, ts.WebSocketURL(spectatorToken))
	spectatorClient.JoinRoom(room.ID.String(), "spectator")

	// Spectator receives state sync
	stateSync := spectatorClient.ExpectStateSync(defaultTimeout)
	assert.Equal(t, "spectator", stateSync.YourSide)
	assert.GreaterOrEqual(t, stateSync.SpectatorCount, 1)

	// Drain any join notifications
	blueClient.DrainMessages()
	redClient.DrainMessages()
	spectatorClient.DrainMessages()

	// Ready up - all clients receive player updates
	blueClient.Ready(true)
	blueClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)
	redClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)
	spectatorClient.ExpectPlayerUpdateForSide("blue", defaultTimeout)

	redClient.Ready(true)
	redClient.ExpectPlayerUpdateForSide("red", defaultTimeout)
	blueClient.ExpectPlayerUpdateForSide("red", defaultTimeout)
	spectatorClient.ExpectPlayerUpdateForSide("red", defaultTimeout)

	// Start draft - all clients receive draft started
	blueClient.StartDraft()
	blueClient.ExpectDraftStarted(defaultTimeout)
	redClient.ExpectDraftStarted(defaultTimeout)
	spectatorClient.ExpectDraftStarted(defaultTimeout)

	// Blue makes a selection
	blueClient.SelectChampion(champions[0].ID)
	blueClient.LockIn()

	// All clients receive champion selected and phase changed
	blueClient.ExpectChampionSelected(defaultTimeout)
	redClient.ExpectChampionSelected(defaultTimeout)
	spectatorClient.ExpectChampionSelected(defaultTimeout)

	blueClient.ExpectPhaseChanged(defaultTimeout)
	redClient.ExpectPhaseChanged(defaultTimeout)
	spectatorClient.ExpectPhaseChanged(defaultTimeout)
}
