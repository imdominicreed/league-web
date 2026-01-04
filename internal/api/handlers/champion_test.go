package handlers_test

import (
	"net/http"
	"testing"

	"github.com/dom/league-draft-website/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ChampionResponse struct {
	ID       string   `json:"id"`
	Key      string   `json:"key"`
	Name     string   `json:"name"`
	Title    string   `json:"title"`
	ImageURL string   `json:"imageUrl"`
	Tags     []string `json:"tags"`
}

type ChampionsListResponse struct {
	Champions []ChampionResponse `json:"champions"`
	Version   string             `json:"version"`
}

func TestChampionHandler_GetAll(t *testing.T) {
	ts := testutil.NewTestServer(t)

	tests := []struct {
		name           string
		setup          func()
		expectedStatus int
		checkResponse  func(*testing.T, *http.Response)
	}{
		{
			name:           "empty database",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result ChampionsListResponse
				testutil.AssertJSONResponse(t, resp, &result)
				assert.Empty(t, result.Champions)
			},
		},
		{
			name: "with champions",
			setup: func() {
				testutil.SeedChampions(t, ts.DB.DB, 5)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result ChampionsListResponse
				testutil.AssertJSONResponse(t, resp, &result)
				assert.Len(t, result.Champions, 5)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts.DB.Truncate(t)

			if tt.setup != nil {
				tt.setup()
			}

			resp, err := http.Get(ts.APIURL("/champions"))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestChampionHandler_Get(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create a champion
	champion := testutil.NewChampionBuilder().
		WithID("Ezreal").
		WithName("Ezreal").
		WithTitle("The Prodigal Explorer").
		Build(t, ts.DB.DB)

	tests := []struct {
		name           string
		id             string
		expectedStatus int
		checkResponse  func(*testing.T, *http.Response)
	}{
		{
			name:           "existing champion",
			id:             champion.ID,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var result ChampionResponse
				testutil.AssertJSONResponse(t, resp, &result)
				assert.Equal(t, champion.ID, result.ID)
				assert.Equal(t, champion.Name, result.Name)
			},
		},
		{
			name:           "non-existent champion",
			id:             "NonExistent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(ts.APIURL("/champions/" + tt.id))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestChampionHandler_SortedByName(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create champions with specific names to test sorting
	testutil.NewChampionBuilder().WithID("Zed").WithName("Zed").Build(t, ts.DB.DB)
	testutil.NewChampionBuilder().WithID("Ahri").WithName("Ahri").Build(t, ts.DB.DB)
	testutil.NewChampionBuilder().WithID("MissFortune").WithName("Miss Fortune").Build(t, ts.DB.DB)

	resp, err := http.Get(ts.APIURL("/champions"))
	require.NoError(t, err)
	defer resp.Body.Close()

	var result ChampionsListResponse
	testutil.AssertJSONResponse(t, resp, &result)

	assert.Len(t, result.Champions, 3)
	// Verify sorted by name
	assert.Equal(t, "Ahri", result.Champions[0].Name)
	assert.Equal(t, "Miss Fortune", result.Champions[1].Name)
	assert.Equal(t, "Zed", result.Champions[2].Name)
}
