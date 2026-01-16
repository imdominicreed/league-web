package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"
)

const apiBase = "http://localhost:9999/api/v1"

type User struct {
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
	Token       string `json:"token"`
	UserID      string `json:"userId"`
}

type Lobby struct {
	ID        string `json:"id"`
	ShortCode string `json:"shortCode"`
}

type RegisterResponse struct {
	User struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"user"`
	AccessToken string `json:"accessToken"`
}

func registerUser(displayName, password string) (*User, error) {
	body, _ := json.Marshal(map[string]string{
		"displayName": displayName,
		"password":    password,
	})

	resp, err := http.Post(apiBase+"/auth/register", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	return &User{
		DisplayName: result.User.DisplayName,
		Password:    password,
		Token:       result.AccessToken,
		UserID:      result.User.ID,
	}, nil
}

func createLobby(token string) (*Lobby, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"draftMode":            "pro_play",
		"timerDurationSeconds": 30,
	})

	req, _ := http.NewRequest("POST", apiBase+"/lobbies", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create lobby failed (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result Lobby
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	return &result, nil
}

func joinLobby(token, lobbyID string) error {
	req, _ := http.NewRequest("POST", apiBase+"/lobbies/"+lobbyID+"/join", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("join lobby failed (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func initializeRoleProfiles(token string) error {
	req, _ := http.NewRequest("POST", apiBase+"/profile/roles/initialize", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 200 OK or 409 Conflict (already initialized) are both fine
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("init roles failed (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func generateUsername(index int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	random := make([]byte, 4)
	for i := range random {
		random[i] = letters[rand.Intn(len(letters))]
	}
	return fmt.Sprintf("test_%d_%d_%s", index, time.Now().Unix(), string(random))
}

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println("Setting up 10-man lobby...\n")

	password := "testpassword123"
	var users []*User

	// Register 10 users
	fmt.Println("Registering 10 users...")
	for i := 1; i <= 10; i++ {
		username := generateUsername(i)
		user, err := registerUser(username, password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to register user %d: %v\n", i, err)
			os.Exit(1)
		}
		users = append(users, user)
		fmt.Printf("  ✓ User %d: %s\n", i, user.DisplayName)
	}

	// Initialize role profiles for all users (needed for matchmaking)
	fmt.Println("\nInitializing role profiles...")
	for i, user := range users {
		if err := initializeRoleProfiles(user.Token); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to init roles for user %d: %v\n", i+1, err)
			os.Exit(1)
		}
	}
	fmt.Println("  ✓ Role profiles initialized")

	// First user creates the lobby
	fmt.Println("\nCreating lobby...")
	lobby, err := createLobby(users[0].Token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create lobby: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  ✓ Lobby created: %s\n", lobby.ShortCode)

	// Other 9 users join the lobby
	fmt.Println("\nJoining users to lobby...")
	for i := 1; i < 10; i++ {
		if err := joinLobby(users[i].Token, lobby.ID); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to join user %d: %v\n", i+1, err)
			os.Exit(1)
		}
		fmt.Printf("  ✓ User %d joined\n", i+1)
	}

	// Output the setup information
	fmt.Println("\n" + "============================================================")
	fmt.Println("10-MAN LOBBY SETUP COMPLETE")
	fmt.Println("============================================================")

	fmt.Println("\nLobby Info:")
	fmt.Printf("  ID: %s\n", lobby.ID)
	fmt.Printf("  Code: %s\n", lobby.ShortCode)
	fmt.Printf("  URL: http://localhost:3000/lobby/%s\n", lobby.ShortCode)

	fmt.Println("\nUsers (all use password: testpassword123):")
	fmt.Println("\n  Blue Team (1-5):")
	for i := 0; i < 5; i++ {
		fmt.Printf("    User %d: %s\n", i+1, users[i].DisplayName)
	}
	fmt.Println("\n  Red Team (6-10):")
	for i := 5; i < 10; i++ {
		fmt.Printf("    User %d: %s\n", i+1, users[i].DisplayName)
	}

	fmt.Println("\n" + "============================================================")
	fmt.Println("QUICK START")
	fmt.Println("============================================================")
	fmt.Printf("\nLobby URL: http://localhost:3000/lobby/%s\n", lobby.ShortCode)
	fmt.Println("\nLogin at http://localhost:3000/login with any user:")
	for i, user := range users {
		fmt.Printf("  User %2d: %s / %s\n", i+1, user.DisplayName, user.Password)
	}

	fmt.Println("\nNext steps:")
	fmt.Println("  1. Login as any user")
	fmt.Println("  2. Go to lobby URL above")
	fmt.Println("  3. Click Ready (do this for all 10 users or use API)")
	fmt.Println("  4. As creator (User 1), generate teams")
	fmt.Println("  5. Select a match option and start draft")

	// Output JSON for programmatic use
	output := map[string]interface{}{
		"lobby": map[string]string{
			"id":        lobby.ID,
			"shortCode": lobby.ShortCode,
			"url":       fmt.Sprintf("http://localhost:3000/lobby/%s", lobby.ShortCode),
		},
		"users": users,
	}

	fmt.Println("\n" + "============================================================")
	fmt.Println("JSON OUTPUT (for scripts):")
	fmt.Println("============================================================")
	jsonOutput, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(jsonOutput))
}
