package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/dom/league-draft-website/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthHandler_Register(t *testing.T) {
	ts := testutil.NewTestServer(t)

	tests := []struct {
		name           string
		request        map[string]string
		setup          func()
		expectedStatus int
		checkResponse  func(*testing.T, *http.Response)
	}{
		{
			name: "successful registration",
			request: map[string]string{
				"displayName": "newuser",
				"password":    "password123",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result testutil.AuthResponse
				testutil.AssertJSONResponse(t, resp, &result)
				assert.Equal(t, "newuser", result.User.DisplayName)
				assert.NotEmpty(t, result.AccessToken)
				assert.NotEmpty(t, result.RefreshToken)
			},
		},
		{
			name: "missing display name",
			request: map[string]string{
				"password": "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			request: map[string]string{
				"displayName": "testuser",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "duplicate display name",
			request: map[string]string{
				"displayName": "existinguser",
				"password":    "password123",
			},
			setup: func() {
				testutil.NewUserBuilder().
					WithDisplayName("existinguser").
					Build(t, ts.DB.DB)
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "empty request body",
			request:        map[string]string{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts.DB.Truncate(t)

			if tt.setup != nil {
				tt.setup()
			}

			body, _ := json.Marshal(tt.request)
			resp, err := http.Post(ts.APIURL("/auth/register"), "application/json", bytes.NewBuffer(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create a user for login tests
	user, rawPassword := testutil.NewUserBuilder().
		WithDisplayName("loginuser").
		WithPassword("correctpassword").
		Build(t, ts.DB.DB)

	tests := []struct {
		name           string
		request        map[string]string
		expectedStatus int
		checkResponse  func(*testing.T, *http.Response)
	}{
		{
			name: "successful login",
			request: map[string]string{
				"displayName": user.DisplayName,
				"password":    rawPassword,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result testutil.AuthResponse
				testutil.AssertJSONResponse(t, resp, &result)
				assert.Equal(t, user.DisplayName, result.User.DisplayName)
				assert.NotEmpty(t, result.AccessToken)
			},
		},
		{
			name: "invalid password",
			request: map[string]string{
				"displayName": user.DisplayName,
				"password":    "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "non-existent user",
			request: map[string]string{
				"displayName": "nonexistent",
				"password":    "anypassword",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing display name",
			request: map[string]string{
				"password": "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			request: map[string]string{
				"displayName": "testuser",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			resp, err := http.Post(ts.APIURL("/auth/login"), "application/json", bytes.NewBuffer(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestAuthHandler_Me(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create and authenticate a user
	user, token := testutil.NewUserBuilder().
		WithDisplayName("meuser").
		BuildAndAuthenticate(t, ts)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		checkResponse  func(*testing.T, *http.Response)
	}{
		{
			name:           "successful fetch with valid token",
			token:          token,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result struct {
					ID          string `json:"id"`
					DisplayName string `json:"displayName"`
				}
				testutil.AssertJSONResponse(t, resp, &result)
				assert.Equal(t, user.ID.String(), result.ID)
				assert.Equal(t, user.DisplayName, result.DisplayName)
			},
		},
		{
			name:           "missing authorization header",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			token:          "invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "malformed token",
			token:          "notajwt",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CreateAuthenticatedRequest(t, "GET", ts.APIURL("/auth/me"), nil, tt.token)

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

func TestAuthHandler_Logout(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create and authenticate a user
	_, token := testutil.NewUserBuilder().
		WithDisplayName("logoutuser").
		BuildAndAuthenticate(t, ts)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{
			name:           "successful logout",
			token:          token,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unauthorized - no token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CreateAuthenticatedRequest(t, "POST", ts.APIURL("/auth/logout"), nil, tt.token)

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}
