package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

type MatchHistoryItemResponse struct {
	ID          string `json:"id"`
	ShortCode   string `json:"shortCode"`
	DraftMode   string `json:"draftMode"`
	CompletedAt string `json:"completedAt"`
	IsTeamDraft bool   `json:"isTeamDraft"`
	YourSide    string `json:"yourSide"`
	BluePicks   []string `json:"bluePicks"`
	RedPicks    []string `json:"redPicks"`
}

type MatchDetailResponse struct {
	ID        string   `json:"id"`
	ShortCode string   `json:"shortCode"`
	DraftMode string   `json:"draftMode"`
	BluePicks []string `json:"bluePicks"`
	RedPicks  []string `json:"redPicks"`
	BlueBans  []string `json:"blueBans"`
	RedBans   []string `json:"redBans"`
	Actions   []struct {
		PhaseIndex int    `json:"phaseIndex"`
		Team       string `json:"team"`
		ActionType string `json:"actionType"`
		ChampionID string `json:"championId"`
	} `json:"actions"`
}

func TestMatchHistoryHandler_List(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create and authenticate a user
	user, token := testutil.NewUserBuilder().
		WithDisplayName("historyuser").
		BuildAndAuthenticate(t, ts)

	// Create a completed room
	room := createCompletedRoom(t, ts, user.ID)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		checkResponse  func(*testing.T, *http.Response)
	}{
		{
			name:           "list completed matches",
			token:          token,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result []MatchHistoryItemResponse
				testutil.AssertJSONResponse(t, resp, &result)
				assert.Len(t, result, 1)
				assert.Equal(t, room.ID.String(), result[0].ID)
				assert.Equal(t, "pro_play", result[0].DraftMode)
				assert.NotEmpty(t, result[0].CompletedAt)
			},
		},
		{
			name:           "unauthorized request",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CreateAuthenticatedRequest(t, "GET", ts.APIURL("/match-history"), nil, tt.token)

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestMatchHistoryHandler_List_Empty(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create and authenticate a user with no completed matches
	_, token := testutil.NewUserBuilder().
		WithDisplayName("newuser").
		BuildAndAuthenticate(t, ts)

	req := testutil.CreateAuthenticatedRequest(t, "GET", ts.APIURL("/match-history"), nil, token)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []MatchHistoryItemResponse
	testutil.AssertJSONResponse(t, resp, &result)
	assert.Len(t, result, 0)
}

func TestMatchHistoryHandler_GetDetail(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create and authenticate a user
	user, token := testutil.NewUserBuilder().
		WithDisplayName("detailuser").
		BuildAndAuthenticate(t, ts)

	// Create another user
	otherUser, otherToken := testutil.NewUserBuilder().
		WithDisplayName("otheruser").
		BuildAndAuthenticate(t, ts)

	// Create a completed room
	room := createCompletedRoomWithActions(t, ts, user.ID)

	tests := []struct {
		name           string
		roomID         string
		token          string
		expectedStatus int
		checkResponse  func(*testing.T, *http.Response)
	}{
		{
			name:           "get match detail",
			roomID:         room.ID.String(),
			token:          token,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result MatchDetailResponse
				testutil.AssertJSONResponse(t, resp, &result)
				assert.Equal(t, room.ID.String(), result.ID)
				assert.Equal(t, "pro_play", result.DraftMode)
				assert.NotEmpty(t, result.BluePicks)
				assert.NotEmpty(t, result.Actions)
			},
		},
		{
			name:           "forbidden - user not in match",
			roomID:         room.ID.String(),
			token:          otherToken,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "not found - invalid room ID",
			roomID:         uuid.New().String(),
			token:          token,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "bad request - invalid UUID",
			roomID:         "invalid-uuid",
			token:          token,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unauthorized request",
			roomID:         room.ID.String(),
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CreateAuthenticatedRequest(t, "GET", ts.APIURL("/match-history/"+tt.roomID), nil, tt.token)

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}

	// Test that otherUser is now forbidden
	_ = otherUser
}

func TestMatchHistoryHandler_Pagination(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create and authenticate a user
	user, token := testutil.NewUserBuilder().
		WithDisplayName("paginationuser").
		BuildAndAuthenticate(t, ts)

	// Create multiple completed rooms
	for i := 0; i < 5; i++ {
		createCompletedRoom(t, ts, user.ID)
	}

	// Test limit
	req := testutil.CreateAuthenticatedRequest(t, "GET", ts.APIURL("/match-history?limit=2"), nil, token)
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result []MatchHistoryItemResponse
	testutil.AssertJSONResponse(t, resp, &result)
	assert.Len(t, result, 2)

	// Test offset
	req2 := testutil.CreateAuthenticatedRequest(t, "GET", ts.APIURL("/match-history?limit=2&offset=2"), nil, token)
	resp2, err := client.Do(req2)
	require.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var result2 []MatchHistoryItemResponse
	testutil.AssertJSONResponse(t, resp2, &result2)
	assert.Len(t, result2, 2)
}

// Helper functions

func createCompletedRoom(t *testing.T, ts *testutil.TestServer, userID uuid.UUID) *domain.Room {
	t.Helper()
	ctx := context.Background()

	now := time.Now()
	room := &domain.Room{
		ID:                   uuid.New(),
		ShortCode:            generateShortCode(),
		CreatedBy:            userID,
		DraftMode:            domain.DraftModeProPlay,
		TimerDurationSeconds: 30,
		Status:               domain.RoomStatusCompleted,
		BlueSideUserID:       &userID,
		CompletedAt:          &now,
	}

	err := ts.Repos.Room.Create(ctx, room)
	require.NoError(t, err)

	// Create draft state
	bluePicks, _ := json.Marshal([]string{"Aatrox", "LeeSin", "Ahri", "Jinx", "Thresh"})
	redPicks, _ := json.Marshal([]string{"Garen", "Elise", "Zed", "Caitlyn", "Leona"})
	blueBans, _ := json.Marshal([]string{"Yasuo", "Yone", "Zeri", "KSante", "Ksante"})
	redBans, _ := json.Marshal([]string{"Yuumi", "Nilah", "Aphelios", "Samira", "Draven"})

	draftState := &domain.DraftState{
		RoomID:       room.ID,
		CurrentPhase: 20,
		BluePicks:    datatypes.JSON(bluePicks),
		RedPicks:     datatypes.JSON(redPicks),
		BlueBans:     datatypes.JSON(blueBans),
		RedBans:      datatypes.JSON(redBans),
		IsComplete:   true,
	}

	err = ts.Repos.DraftState.Create(ctx, draftState)
	require.NoError(t, err)

	return room
}

func createCompletedRoomWithActions(t *testing.T, ts *testutil.TestServer, userID uuid.UUID) *domain.Room {
	t.Helper()
	room := createCompletedRoom(t, ts, userID)
	ctx := context.Background()

	// Add some draft actions
	actions := []*domain.DraftAction{
		{RoomID: room.ID, PhaseIndex: 0, Team: domain.SideBlue, ActionType: domain.ActionTypeBan, ChampionID: "Yasuo", ActionTime: time.Now()},
		{RoomID: room.ID, PhaseIndex: 1, Team: domain.SideRed, ActionType: domain.ActionTypeBan, ChampionID: "Yuumi", ActionTime: time.Now()},
		{RoomID: room.ID, PhaseIndex: 6, Team: domain.SideBlue, ActionType: domain.ActionTypePick, ChampionID: "Aatrox", ActionTime: time.Now()},
		{RoomID: room.ID, PhaseIndex: 7, Team: domain.SideRed, ActionType: domain.ActionTypePick, ChampionID: "Garen", ActionTime: time.Now()},
	}

	for _, action := range actions {
		err := ts.Repos.DraftAction.Create(ctx, action)
		require.NoError(t, err)
	}

	return room
}

func generateShortCode() string {
	return uuid.New().String()[:8]
}
