package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient handles HTTP communication with the backend
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL + "/api/v1",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Response types matching backend

type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type AuthResponse struct {
	User         User   `json:"user"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type Lobby struct {
	ID                   string        `json:"id"`
	ShortCode            string        `json:"shortCode"`
	CreatedBy            string        `json:"createdBy"`
	Status               string        `json:"status"`
	SelectedMatchOption  *int          `json:"selectedMatchOption"`
	DraftMode            string        `json:"draftMode"`
	TimerDurationSeconds int           `json:"timerDurationSeconds"`
	RoomID               *string       `json:"roomId"`
	VotingEnabled        bool          `json:"votingEnabled"`
	VotingMode           string        `json:"votingMode"`
	Players              []LobbyPlayer `json:"players"`
}

type LobbyPlayer struct {
	ID           string  `json:"id"`
	UserID       string  `json:"userId"`
	DisplayName  string  `json:"displayName"`
	Team         *string `json:"team"`
	AssignedRole *string `json:"assignedRole"`
	IsReady      bool    `json:"isReady"`
	IsCaptain    bool    `json:"isCaptain"`
}

type MatchOption struct {
	OptionNumber   int          `json:"optionNumber"`
	BlueTeamAvgMMR int          `json:"blueTeamAvgMmr"`
	RedTeamAvgMMR  int          `json:"redTeamAvgMmr"`
	MMRDifference  int          `json:"mmrDifference"`
	BalanceScore   float64      `json:"balanceScore"`
	Assignments    []Assignment `json:"assignments"`
}

type Assignment struct {
	UserID        string `json:"userId"`
	DisplayName   string `json:"displayName"`
	Team          string `json:"team"`
	AssignedRole  string `json:"assignedRole"`
	RoleMMR       int    `json:"roleMmr"`
	ComfortRating int    `json:"comfortRating"`
}

type RoleProfile struct {
	Role          string `json:"role"`
	LeagueRank    string `json:"leagueRank"`
	MMR           int    `json:"mmr"`
	ComfortRating int    `json:"comfortRating"`
}

// Login authenticates an existing user
func (c *APIClient) Login(displayName, password string) (string, error) {
	body := map[string]string{
		"displayName": displayName,
		"password":    password,
	}

	resp, err := c.post("/auth/login", body, "")
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.AccessToken, nil
}

// RegisterUser creates a new user account
func (c *APIClient) RegisterUser(baseName string) (*User, string, error) {
	displayName := fmt.Sprintf("%s_%d", baseName, time.Now().UnixNano()%100000)

	body := map[string]string{
		"displayName": displayName,
		"password":    "asdf",
	}

	resp, err := c.post("/auth/register", body, "")
	if err != nil {
		return nil, "", fmt.Errorf("register request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("register failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("failed to decode response: %w", err)
	}

	return &result.User, result.AccessToken, nil
}

// CreateLobbyOptions specifies options for creating a lobby
type CreateLobbyOptions struct {
	VotingEnabled bool
	VotingMode    string // "majority", "unanimous", "captain_override"
}

// CreateLobby creates a new 10-man lobby
func (c *APIClient) CreateLobby(token string) (*Lobby, error) {
	return c.CreateLobbyWithOptions(token, CreateLobbyOptions{})
}

// CreateLobbyWithOptions creates a new 10-man lobby with options
func (c *APIClient) CreateLobbyWithOptions(token string, opts CreateLobbyOptions) (*Lobby, error) {
	body := map[string]interface{}{
		"draftMode":            "pro_play",
		"timerDurationSeconds": 30,
		"votingEnabled":        opts.VotingEnabled,
	}
	if opts.VotingMode != "" {
		body["votingMode"] = opts.VotingMode
	}

	resp, err := c.post("/lobbies", body, token)
	if err != nil {
		return nil, fmt.Errorf("create lobby request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create lobby failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var lobby Lobby
	if err := json.NewDecoder(resp.Body).Decode(&lobby); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &lobby, nil
}

// GetLobby fetches lobby details
func (c *APIClient) GetLobby(idOrCode string) (*Lobby, error) {
	resp, err := c.get("/lobbies/"+idOrCode, "")
	if err != nil {
		return nil, fmt.Errorf("get lobby request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get lobby failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var lobby Lobby
	if err := json.NewDecoder(resp.Body).Decode(&lobby); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &lobby, nil
}

// JoinLobby joins a user to a lobby
func (c *APIClient) JoinLobby(token, lobbyID string) error {
	resp, err := c.post("/lobbies/"+lobbyID+"/join", nil, token)
	if err != nil {
		return fmt.Errorf("join lobby request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("join lobby failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// SetReady sets the player's ready status
func (c *APIClient) SetReady(token, lobbyID string, ready bool) error {
	body := map[string]bool{
		"ready": ready,
	}

	resp, err := c.post("/lobbies/"+lobbyID+"/ready", body, token)
	if err != nil {
		return fmt.Errorf("set ready request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set ready failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// GenerateTeams generates match options for a lobby
func (c *APIClient) GenerateTeams(token, lobbyID string) ([]MatchOption, error) {
	resp, err := c.post("/lobbies/"+lobbyID+"/generate-teams", nil, token)
	if err != nil {
		return nil, fmt.Errorf("generate teams request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("generate teams failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var options []MatchOption
	if err := json.NewDecoder(resp.Body).Decode(&options); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return options, nil
}

// SelectOption selects a match option
func (c *APIClient) SelectOption(token, lobbyID string, optionNumber int) error {
	body := map[string]int{
		"optionNumber": optionNumber,
	}

	resp, err := c.post("/lobbies/"+lobbyID+"/select-option", body, token)
	if err != nil {
		return fmt.Errorf("select option request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("select option failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// VotingStatus represents the current voting state
type VotingStatus struct {
	VotingEnabled bool           `json:"votingEnabled"`
	VotingMode    string         `json:"votingMode"`
	TotalPlayers  int            `json:"totalPlayers"`
	VotesCast     int            `json:"votesCast"`
	VoteCounts    map[string]int `json:"voteCounts"`
	UserVote      *int           `json:"userVote,omitempty"`
	WinningOption *int           `json:"winningOption,omitempty"`
	CanFinalize   bool           `json:"canFinalize"`
}

// Vote casts a vote for a match option
func (c *APIClient) Vote(token, lobbyID string, optionNumber int) (*VotingStatus, error) {
	body := map[string]int{
		"optionNumber": optionNumber,
	}

	resp, err := c.post("/lobbies/"+lobbyID+"/vote", body, token)
	if err != nil {
		return nil, fmt.Errorf("vote request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vote failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var status VotingStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// GetVotingStatus gets the current voting status for a lobby
func (c *APIClient) GetVotingStatus(token, lobbyID string) (*VotingStatus, error) {
	resp, err := c.get("/lobbies/"+lobbyID+"/voting-status", token)
	if err != nil {
		return nil, fmt.Errorf("get voting status request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get voting status failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var status VotingStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// EndVoting ends voting and optionally forces a specific option (captain_override mode)
func (c *APIClient) EndVoting(token, lobbyID string, forceOption *int) (*Lobby, error) {
	body := map[string]interface{}{}
	if forceOption != nil {
		body["forceOption"] = *forceOption
	}

	resp, err := c.post("/lobbies/"+lobbyID+"/end-voting", body, token)
	if err != nil {
		return nil, fmt.Errorf("end voting request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("end voting failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var lobby Lobby
	if err := json.NewDecoder(resp.Body).Decode(&lobby); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &lobby, nil
}

// InitializeProfiles creates default role profiles for a user
func (c *APIClient) InitializeProfiles(token string) error {
	resp, err := c.post("/profile/roles/initialize", nil, token)
	if err != nil {
		return fmt.Errorf("initialize profiles request failed: %w", err)
	}
	defer resp.Body.Close()

	// 201 for created, 200 if already exist
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("initialize profiles failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// SetVariedProfiles updates user's role profiles with varied MMR and comfort ratings
func (c *APIClient) SetVariedProfiles(token string, playerIndex int) error {
	roles := []string{"top", "jungle", "mid", "adc", "support"}
	ranks := []string{
		"Silver IV", "Silver III", "Silver II", "Silver I",
		"Gold IV", "Gold III", "Gold II", "Gold I",
		"Platinum IV", "Platinum III",
	}

	for i, role := range roles {
		// Create varied stats based on player and role
		rankIndex := (playerIndex + i) % len(ranks)
		comfort := ((playerIndex + i) % 5) + 1 // 1-5

		body := map[string]interface{}{
			"leagueRank":    ranks[rankIndex],
			"comfortRating": comfort,
		}

		resp, err := c.put("/profile/roles/"+role, body, token)
		if err != nil {
			return fmt.Errorf("update %s profile failed: %w", role, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("update %s profile failed (status %d)", role, resp.StatusCode)
		}
	}

	return nil
}

// HTTP helpers

func (c *APIClient) get(path, token string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func (c *APIClient) post(path string, body interface{}, token string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest("POST", c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func (c *APIClient) put(path string, body interface{}, token string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest("PUT", c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}
