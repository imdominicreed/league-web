package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/dom/league-draft-website/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type RoomResponse struct {
	ID            string  `json:"id"`
	ShortCode     string  `json:"shortCode"`
	DraftMode     string  `json:"draftMode"`
	Status        string  `json:"status"`
	TimerDuration int     `json:"timerDurationSeconds"`
	BlueSideUser  *string `json:"blueSideUser"`
	RedSideUser   *string `json:"redSideUser"`
}

type JoinResponse struct {
	Room         RoomResponse `json:"room"`
	YourSide     string       `json:"yourSide"`
	WebsocketURL string       `json:"websocketUrl"`
}

func TestRoomHandler_Create(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create and authenticate a user
	_, token := testutil.NewUserBuilder().
		WithDisplayName("roomcreator").
		BuildAndAuthenticate(t, ts)

	tests := []struct {
		name           string
		token          string
		request        map[string]interface{}
		expectedStatus int
		checkResponse  func(*testing.T, *http.Response)
	}{
		{
			name:  "successful room creation with pro play mode",
			token: token,
			request: map[string]interface{}{
				"draftMode":     "pro_play",
				"timerDuration": 30,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result RoomResponse
				testutil.AssertJSONResponse(t, resp, &result)
				assert.NotEmpty(t, result.ID)
				assert.NotEmpty(t, result.ShortCode)
				assert.Equal(t, "pro_play", result.DraftMode)
				assert.Equal(t, "waiting", result.Status)
				assert.Equal(t, 30, result.TimerDuration)
			},
		},
		{
			name:  "successful room creation with fearless mode",
			token: token,
			request: map[string]interface{}{
				"draftMode":     "fearless",
				"timerDuration": 45,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result RoomResponse
				testutil.AssertJSONResponse(t, resp, &result)
				assert.Equal(t, "fearless", result.DraftMode)
				assert.Equal(t, 45, result.TimerDuration)
			},
		},
		{
			name:  "unauthorized request",
			token: "",
			request: map[string]interface{}{
				"draftMode": "pro_play",
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := testutil.CreateAuthenticatedRequest(t, "POST", ts.APIURL("/rooms"), tt.request, tt.token)
			req.Body = io.NopCloser(bytes.NewBuffer(body))

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

func TestRoomHandler_Get(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create and authenticate a user
	_, token := testutil.NewUserBuilder().
		WithDisplayName("roomgetter").
		BuildAndAuthenticate(t, ts)

	// Create a room using the API
	createResp := createRoom(t, ts, token)

	tests := []struct {
		name           string
		idOrCode       string
		token          string
		expectedStatus int
	}{
		{
			name:           "get room by UUID",
			idOrCode:       createResp.ID,
			token:          token,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get room by short code",
			idOrCode:       createResp.ShortCode,
			token:          token,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "room not found - invalid UUID",
			idOrCode:       "00000000-0000-0000-0000-000000000000",
			token:          token,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "room not found - invalid short code",
			idOrCode:       "INVALID",
			token:          token,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "unauthorized request",
			idOrCode:       createResp.ID,
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CreateAuthenticatedRequest(t, "GET", ts.APIURL("/rooms/"+tt.idOrCode), nil, tt.token)

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestRoomHandler_Join(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create users
	_, creatorToken := testutil.NewUserBuilder().
		WithDisplayName("creator").
		BuildAndAuthenticate(t, ts)

	_, user1Token := testutil.NewUserBuilder().
		WithDisplayName("user1").
		BuildAndAuthenticate(t, ts)

	_, user2Token := testutil.NewUserBuilder().
		WithDisplayName("user2").
		BuildAndAuthenticate(t, ts)

	tests := []struct {
		name           string
		setup          func() string // Returns room ID
		token          string
		request        map[string]string
		expectedStatus int
		checkResponse  func(*testing.T, *http.Response)
	}{
		{
			name: "join blue side",
			setup: func() string {
				room := createRoom(t, ts, creatorToken)
				return room.ID
			},
			token: user1Token,
			request: map[string]string{
				"side": "blue",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result JoinResponse
				testutil.AssertJSONResponse(t, resp, &result)
				assert.Equal(t, "blue", result.YourSide)
			},
		},
		{
			name: "join red side",
			setup: func() string {
				room := createRoom(t, ts, creatorToken)
				return room.ID
			},
			token: user1Token,
			request: map[string]string{
				"side": "red",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result JoinResponse
				testutil.AssertJSONResponse(t, resp, &result)
				assert.Equal(t, "red", result.YourSide)
			},
		},
		{
			name: "auto-assign to blue when empty",
			setup: func() string {
				room := createRoom(t, ts, creatorToken)
				return room.ID
			},
			token: user1Token,
			request: map[string]string{
				"side": "auto",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result JoinResponse
				testutil.AssertJSONResponse(t, resp, &result)
				assert.Equal(t, "blue", result.YourSide)
			},
		},
		{
			name: "auto-assign to red when blue taken",
			setup: func() string {
				room := createRoom(t, ts, creatorToken)
				// User1 takes blue side
				joinRoom(t, ts, room.ID, user1Token, "blue")
				return room.ID
			},
			token: user2Token,
			request: map[string]string{
				"side": "auto",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result JoinResponse
				testutil.AssertJSONResponse(t, resp, &result)
				assert.Equal(t, "red", result.YourSide)
			},
		},
		{
			name: "room not found",
			setup: func() string {
				return "00000000-0000-0000-0000-000000000000"
			},
			token: user1Token,
			request: map[string]string{
				"side": "blue",
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "unauthorized request",
			setup: func() string {
				room := createRoom(t, ts, creatorToken)
				return room.ID
			},
			token: "",
			request: map[string]string{
				"side": "blue",
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roomID := tt.setup()

			body, _ := json.Marshal(tt.request)
			req := testutil.CreateAuthenticatedRequest(t, "POST", ts.APIURL("/rooms/"+roomID+"/join"), nil, tt.token)
			req.Body = io.NopCloser(bytes.NewBuffer(body))

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

func TestRoomHandler_GetUserRooms(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create and authenticate a user
	_, token := testutil.NewUserBuilder().
		WithDisplayName("userrooms").
		BuildAndAuthenticate(t, ts)

	// Create some rooms
	for i := 0; i < 3; i++ {
		createRoom(t, ts, token)
	}

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		checkResponse  func(*testing.T, *http.Response)
	}{
		{
			name:           "get user rooms",
			token:          token,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result []RoomResponse
				testutil.AssertJSONResponse(t, resp, &result)
				assert.Len(t, result, 3)
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
			req := testutil.CreateAuthenticatedRequest(t, "GET", ts.APIURL("/users/me/drafts"), nil, tt.token)

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

// Helper functions

func createRoom(t *testing.T, ts *testutil.TestServer, token string) RoomResponse {
	t.Helper()

	body, _ := json.Marshal(map[string]interface{}{
		"draftMode":     "pro_play",
		"timerDuration": 30,
	})

	req := testutil.CreateAuthenticatedRequest(t, "POST", ts.APIURL("/rooms"), nil, token)
	req.Body = io.NopCloser(bytes.NewBuffer(body))

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result RoomResponse
	testutil.AssertJSONResponse(t, resp, &result)
	return result
}

func joinRoom(t *testing.T, ts *testutil.TestServer, roomID, token, side string) JoinResponse {
	t.Helper()

	body, _ := json.Marshal(map[string]string{
		"side": side,
	})

	req := testutil.CreateAuthenticatedRequest(t, "POST", ts.APIURL(fmt.Sprintf("/rooms/%s/join", roomID)), nil, token)
	req.Body = io.NopCloser(bytes.NewBuffer(body))

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var result JoinResponse
	testutil.AssertJSONResponse(t, resp, &result)
	return result
}
