package websocket_test

import (
	"testing"
	"time"

	"github.com/dom/league-draft-website/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	// Drain any additional messages before test ends
	wsClient.DrainMessages()
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

	// Drain join notifications before testing ready
	time.Sleep(100 * time.Millisecond)
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Blue player readies up
	blueClient.Ready(true)
	blueUpdate := blueClient.ExpectPlayerUpdate(defaultTimeout)
	assert.Equal(t, "blue", blueUpdate.Side)
	assert.Equal(t, "ready_changed", blueUpdate.Action)
	assert.True(t, blueUpdate.Player.Ready)

	// Drain blue's ready notification from red client before red readies
	time.Sleep(100 * time.Millisecond)
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Red player readies up
	redClient.Ready(true)
	redUpdate := redClient.ExpectPlayerUpdate(defaultTimeout)
	assert.Equal(t, "red", redUpdate.Side)
	assert.Equal(t, "ready_changed", redUpdate.Action)
	assert.True(t, redUpdate.Player.Ready)
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

	// Both ready up
	blueClient.Ready(true)
	blueClient.DrainMessages()
	redClient.DrainMessages()

	redClient.Ready(true)
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Start draft
	blueClient.StartDraft()

	// Expect draft started
	draftStarted := blueClient.ExpectDraftStarted(defaultTimeout)
	assert.Equal(t, 0, draftStarted.CurrentPhase)
	assert.Equal(t, "blue", draftStarted.CurrentTeam)
	assert.Equal(t, "ban", draftStarted.ActionType)
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

	// Ready up and start
	blueClient.Ready(true)
	redClient.Ready(true)
	blueClient.DrainMessages()
	redClient.DrainMessages()

	blueClient.StartDraft()
	blueClient.ExpectDraftStarted(defaultTimeout)
	redClient.ExpectDraftStarted(defaultTimeout)

	// Blue selects champion (phase 0 - blue ban)
	blueClient.SelectChampion(champions[0].ID)

	// Lock in
	blueClient.LockIn()

	// Expect champion selected
	selected := blueClient.ExpectChampionSelected(defaultTimeout)
	assert.Equal(t, 0, selected.Phase)
	assert.Equal(t, "blue", selected.Team)
	assert.Equal(t, "ban", selected.ActionType)
	assert.Equal(t, champions[0].ID, selected.ChampionID)

	// Expect phase changed
	phaseChanged := blueClient.ExpectPhaseChanged(defaultTimeout)
	assert.Equal(t, 1, phaseChanged.CurrentPhase)
	assert.Equal(t, "red", phaseChanged.CurrentTeam)
	assert.Equal(t, "ban", phaseChanged.ActionType)
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

	// Ready up
	blueClient.Ready(true)
	time.Sleep(100 * time.Millisecond)
	redClient.Ready(true)
	time.Sleep(100 * time.Millisecond)
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Start draft
	blueClient.StartDraft()
	time.Sleep(200 * time.Millisecond)
	blueClient.DrainMessages()
	redClient.DrainMessages()

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

	// Ready up
	blueClient.Ready(true)
	time.Sleep(100 * time.Millisecond)
	redClient.Ready(true)
	time.Sleep(100 * time.Millisecond)
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Start draft
	blueClient.StartDraft()
	time.Sleep(200 * time.Millisecond)
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Blue bans a champion
	blueClient.SelectChampion(champions[0].ID)
	blueClient.LockIn()
	time.Sleep(200 * time.Millisecond)
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Red tries to ban the same champion
	redClient.SelectChampion(champions[0].ID)

	// Expect error
	errorPayload := redClient.ExpectErrorWithCode("CHAMPION_UNAVAILABLE", defaultTimeout)
	assert.Contains(t, errorPayload.Message, "already picked or banned")
}

func TestDraftFlow_NoSelection_UsesNone(t *testing.T) {
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

	// Ready up
	blueClient.Ready(true)
	time.Sleep(100 * time.Millisecond)
	redClient.Ready(true)
	time.Sleep(100 * time.Millisecond)
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Start draft
	blueClient.StartDraft()
	time.Sleep(200 * time.Millisecond)
	blueClient.DrainMessages()
	redClient.DrainMessages()

	// Blue locks in without selecting - should use "None"
	blueClient.LockIn()

	// Expect champion selected with "None"
	selected := blueClient.ExpectChampionSelected(defaultTimeout)
	assert.Equal(t, "None", selected.ChampionID)
	assert.Equal(t, "ban", selected.ActionType)
	assert.Equal(t, "blue", selected.Team)
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

	blueClient.Ready(true)
	redClient.Ready(true)
	blueClient.DrainMessages()
	redClient.DrainMessages()

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

	// Ready up
	blueClient.Ready(true)
	time.Sleep(100 * time.Millisecond)
	redClient.Ready(true)
	time.Sleep(100 * time.Millisecond)
	blueClient.DrainMessages()
	redClient.DrainMessages()
	spectatorClient.DrainMessages()

	// Start draft
	blueClient.StartDraft()
	time.Sleep(200 * time.Millisecond)

	// Spectator should have received messages, drain first
	spectatorClient.DrainMessages()

	// Blue makes a selection
	blueClient.SelectChampion(champions[0].ID)
	blueClient.LockIn()

	// Give time for message propagation
	time.Sleep(200 * time.Millisecond)

	// Spectator receives champion selected and phase changed
	spectatorClient.ExpectChampionSelected(defaultTimeout)
	spectatorClient.ExpectPhaseChanged(defaultTimeout)
}
