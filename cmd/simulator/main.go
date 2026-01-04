package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Global flags
	apiURL := "http://localhost:9999"
	if envURL := os.Getenv("API_URL"); envURL != "" {
		apiURL = envURL
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "full":
		fullCmd(apiURL, args)
	case "populate":
		populateCmd(apiURL, args)
	case "ready":
		readyCmd(apiURL, args)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Lobby Simulator - Development tool for testing 10-man lobbies

USAGE:
  simulator <command> [options]

COMMANDS:
  full      Create a lobby with 10 users, ready all, generate teams, and select option
  populate  Add fake users to an existing lobby
  ready     Set all players in a lobby to ready
  help      Show this help message

ENVIRONMENT:
  API_URL   Backend API URL (default: http://localhost:9999)

EXAMPLES:
  # Create lobby with 9 fake players, leaving 1 slot for you to join
  simulator full

  # Create lobby with 5 fake players (5 slots open for real users)
  simulator full --count=5

  # Create fully automated 10-player lobby ready for "Start Draft"
  simulator full --count=10

  # Add 5 more users to an existing lobby
  simulator populate --lobby=ABC123 --count=5

  # Ready all players in a lobby
  simulator ready --lobby=ABC123`)
}

func fullCmd(apiURL string, args []string) {
	fs := flag.NewFlagSet("full", flag.ExitOnError)
	option := fs.Int("option", 1, "Match option to select (1-5)")
	count := fs.Int("count", 9, "Number of fake users to create (default 9, leaving 1 slot for you)")
	skipReady := fs.Bool("skip-ready", false, "Skip readying up players (useful when you want to join)")
	fs.Parse(args)

	if *count < 1 || *count > 10 {
		fmt.Println("Error: --count must be between 1 and 10")
		os.Exit(1)
	}

	client := NewAPIClient(apiURL)

	fmt.Println("=== Lobby Simulator: Full Flow ===")
	fmt.Println()

	// 1. Create lobby with admin user
	fmt.Print("Creating admin user and lobby... ")
	admin, adminToken, err := client.RegisterUser("LobbyAdmin")
	if err != nil {
		fmt.Printf("FAILED\n  Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK (user: %s)\n", admin.DisplayName)

	lobby, err := client.CreateLobby(adminToken)
	if err != nil {
		fmt.Printf("Failed to create lobby: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Lobby created: %s (code: %s)\n", lobby.ID, lobby.ShortCode)

	// Initialize profiles for admin
	if err := client.InitializeProfiles(adminToken); err != nil {
		fmt.Printf("Warning: Failed to initialize admin profiles: %v\n", err)
	}

	// Set varied MMR for admin
	if err := client.SetVariedProfiles(adminToken, 0); err != nil {
		fmt.Printf("Warning: Failed to set admin profiles: %v\n", err)
	}

	// 2. Create and join additional users
	tokens := []string{adminToken}
	fmt.Println()
	fmt.Printf("Adding %d players to lobby:\n", *count)

	for i := 1; i < *count; i++ {
		displayName := fmt.Sprintf("Player%d", i)
		user, token, err := client.RegisterUser(displayName)
		if err != nil {
			fmt.Printf("  [%d/%d] FAILED to create user: %v\n", i+1, *count, err)
			os.Exit(1)
		}

		if err := client.JoinLobby(token, lobby.ID); err != nil {
			fmt.Printf("  [%d/%d] FAILED to join lobby: %v\n", i+1, *count, err)
			os.Exit(1)
		}

		// Initialize and set varied profiles
		if err := client.InitializeProfiles(token); err != nil {
			fmt.Printf("Warning: Failed to initialize profiles for %s: %v\n", user.DisplayName, err)
		}
		if err := client.SetVariedProfiles(token, i); err != nil {
			fmt.Printf("Warning: Failed to set profiles for %s: %v\n", user.DisplayName, err)
		}

		tokens = append(tokens, token)
		fmt.Printf("  [%d/%d] %s joined\n", i+1, *count, user.DisplayName)
	}

	// If we have less than 10 players, show join instructions
	if *count < 10 {
		slotsOpen := 10 - *count
		fmt.Println()
		fmt.Println("=========================================")
		fmt.Printf("  LOBBY WAITING FOR %d MORE PLAYER(S)\n", slotsOpen)
		fmt.Println("=========================================")
		fmt.Println()
		fmt.Printf("  Lobby URL:  http://localhost:3000/lobby/%s\n", lobby.ShortCode)
		fmt.Printf("  Short Code: %s\n", lobby.ShortCode)
		fmt.Println()
		fmt.Println("  Next steps:")
		fmt.Println("  1. Open the lobby URL in your browser")
		fmt.Println("  2. Register/login if needed")
		fmt.Println("  3. Join the lobby")
		fmt.Println("  4. Click 'Ready Up'")
		if !*skipReady {
			fmt.Println()
			fmt.Println("  The fake players are already ready.")
			fmt.Println("  Once you ready up, click 'Generate Teams'!")
		}
		fmt.Println()

		// Ready up fake players if not skipped
		if !*skipReady {
			fmt.Print("Setting fake players ready... ")
			for _, token := range tokens {
				if err := client.SetReady(token, lobby.ID, true); err != nil {
					fmt.Printf("FAILED\n  Error: %v\n", err)
					os.Exit(1)
				}
			}
			fmt.Println("OK")
		}
		return
	}

	// Full 10-player flow
	if *skipReady {
		fmt.Println()
		fmt.Println("=========================================")
		fmt.Println("  LOBBY POPULATED (ready skipped)")
		fmt.Println("=========================================")
		fmt.Println()
		fmt.Printf("  Lobby URL:  http://localhost:3000/lobby/%s\n", lobby.ShortCode)
		fmt.Printf("  Short Code: %s\n", lobby.ShortCode)
		fmt.Println()
		return
	}

	// 3. All players ready up
	fmt.Println()
	fmt.Print("Setting all players ready... ")
	for _, token := range tokens {
		if err := client.SetReady(token, lobby.ID, true); err != nil {
			fmt.Printf("FAILED\n  Error: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println("OK")

	// 4. Generate teams
	fmt.Print("Generating team options... ")
	options, err := client.GenerateTeams(adminToken, lobby.ID)
	if err != nil {
		fmt.Printf("FAILED\n  Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("OK (%d options)\n", len(options))

	// 5. Select option
	fmt.Printf("Selecting option %d... ", *option)
	if err := client.SelectOption(adminToken, lobby.ID, *option); err != nil {
		fmt.Printf("FAILED\n  Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK")

	// Print summary
	fmt.Println()
	fmt.Println("=========================================")
	fmt.Println("  LOBBY READY FOR DRAFT")
	fmt.Println("=========================================")
	fmt.Println()
	fmt.Printf("  Lobby URL:  http://localhost:3000/lobby/%s\n", lobby.ShortCode)
	fmt.Printf("  Short Code: %s\n", lobby.ShortCode)
	fmt.Printf("  Lobby ID:   %s\n", lobby.ID)
	fmt.Println()
	fmt.Println("  Click 'Start Draft' in the UI to begin!")
	fmt.Println()
}

func populateCmd(apiURL string, args []string) {
	fs := flag.NewFlagSet("populate", flag.ExitOnError)
	lobbyCode := fs.String("lobby", "", "Lobby ID or short code (required)")
	count := fs.Int("count", 9, "Number of users to add")
	fs.Parse(args)

	if *lobbyCode == "" {
		fmt.Println("Error: --lobby is required")
		fmt.Println("\nUsage: simulator populate --lobby=ABC123 [--count=9]")
		os.Exit(1)
	}

	client := NewAPIClient(apiURL)

	fmt.Printf("Adding %d players to lobby %s...\n\n", *count, *lobbyCode)

	for i := 0; i < *count; i++ {
		displayName := fmt.Sprintf("Player%d", i+1)
		user, token, err := client.RegisterUser(displayName)
		if err != nil {
			fmt.Printf("  [%d/%d] FAILED to create user: %v\n", i+1, *count, err)
			continue
		}

		if err := client.JoinLobby(token, *lobbyCode); err != nil {
			fmt.Printf("  [%d/%d] FAILED to join: %v\n", i+1, *count, err)
			continue
		}

		// Initialize profiles
		if err := client.InitializeProfiles(token); err != nil {
			fmt.Printf("Warning: Failed to initialize profiles for %s\n", user.DisplayName)
		}
		if err := client.SetVariedProfiles(token, i); err != nil {
			fmt.Printf("Warning: Failed to set profiles for %s\n", user.DisplayName)
		}

		fmt.Printf("  [%d/%d] %s joined\n", i+1, *count, user.DisplayName)
	}

	fmt.Println()
	fmt.Printf("Done! View lobby at: http://localhost:3000/lobby/%s\n", *lobbyCode)
}

func readyCmd(apiURL string, args []string) {
	fs := flag.NewFlagSet("ready", flag.ExitOnError)
	lobbyCode := fs.String("lobby", "", "Lobby ID or short code (required)")
	fs.Parse(args)

	if *lobbyCode == "" {
		fmt.Println("Error: --lobby is required")
		fmt.Println("\nUsage: simulator ready --lobby=ABC123")
		os.Exit(1)
	}

	client := NewAPIClient(apiURL)

	fmt.Printf("Getting lobby %s...\n", *lobbyCode)
	lobby, err := client.GetLobby(*lobbyCode)
	if err != nil {
		fmt.Printf("Failed to get lobby: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d players, setting all to ready...\n", len(lobby.Players))

	// Note: We can't set ready for other users without their tokens
	// This command is mainly informational or for future WebSocket implementation
	fmt.Println()
	fmt.Println("Note: Each player must set their own ready status.")
	fmt.Println("Use 'simulator full' to create a lobby where all players are ready.")
}
