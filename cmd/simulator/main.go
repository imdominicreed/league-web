package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

var frontendURL = "http://localhost:3000"

func init() {
	// Load .env file from project root
	loadEnvFile("/workspaces/project/.env")

	// Override with environment variable if set
	if envURL := os.Getenv("FRONTEND_URL"); envURL != "" {
		frontendURL = envURL
	}
}

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return // .env file is optional
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Only set if not already set in environment
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
			// Set frontendURL directly
			if key == "FRONTEND_URL" {
				frontendURL = value
			}
		}
	}
}

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
	case "vote":
		voteCmd(apiURL, args)
	case "end-voting":
		endVotingCmd(apiURL, args)
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
  full        Create a lobby with 10 users, ready all, generate teams, and select option
  populate    Add fake users to an existing lobby
  ready       Set all players in a lobby to ready
  vote        Have all fake users vote for a match option
  end-voting  End voting and select option (captain can force in captain_override mode)
  help        Show this help message

ENVIRONMENT:
  API_URL   Backend API URL (default: http://localhost:9999)

EXAMPLES:
  # Create lobby with 9 fake players, leaving 1 slot for you to join
  simulator full

  # Create lobby with 5 fake players (5 slots open for real users)
  simulator full --count=5

  # Create fully automated 10-player lobby ready for "Start Draft"
  simulator full --count=10

  # Create lobby with voting enabled (majority vote)
  simulator full --voting

  # Create lobby with unanimous voting required
  simulator full --voting --voting-mode=unanimous

  # Add 5 more users to an existing lobby
  simulator populate --lobby=ABC123 --count=5

  # Ready all players in a lobby
  simulator ready --lobby=ABC123

  # Have all fake users in the lobby vote for option 2
  simulator vote --lobby=ABC123 --option=2

  # Have fake users vote randomly across options
  simulator vote --lobby=ABC123 --random

  # End voting and accept the winning option
  simulator end-voting --lobby=ABC123

  # Captain force-selects option 3 (captain_override mode only)
  simulator end-voting --lobby=ABC123 --force=3`)
}

// SimUser tracks a simulated user and their token for URL generation
type SimUser struct {
	DisplayName string
	Token       string
}

func fullCmd(apiURL string, args []string) {
	fs := flag.NewFlagSet("full", flag.ExitOnError)
	option := fs.Int("option", 1, "Match option to select (1-5)")
	count := fs.Int("count", 9, "Number of fake users to create (default 9, leaving 1 slot for you)")
	skipReady := fs.Bool("skip-ready", false, "Skip readying up players (useful when you want to join)")
	voting := fs.Bool("voting", false, "Enable voting for match option selection")
	votingMode := fs.String("voting-mode", "majority", "Voting mode: majority, unanimous, captain_override")
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

	lobby, err := client.CreateLobbyWithOptions(adminToken, CreateLobbyOptions{
		VotingEnabled: *voting,
		VotingMode:    *votingMode,
	})
	if err != nil {
		fmt.Printf("Failed to create lobby: %v\n", err)
		os.Exit(1)
	}
	votingStr := ""
	if *voting {
		votingStr = fmt.Sprintf(", voting: %s", *votingMode)
	}
	fmt.Printf("  Lobby created: %s (code: %s%s)\n", lobby.ID, lobby.ShortCode, votingStr)

	// Initialize profiles for admin
	if err := client.InitializeProfiles(adminToken); err != nil {
		fmt.Printf("Warning: Failed to initialize admin profiles: %v\n", err)
	}

	// Set varied MMR for admin
	if err := client.SetVariedProfiles(adminToken, 0); err != nil {
		fmt.Printf("Warning: Failed to set admin profiles: %v\n", err)
	}

	// Track all users with their tokens for URL generation
	users := []SimUser{{DisplayName: admin.DisplayName, Token: adminToken}}

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
		users = append(users, SimUser{DisplayName: user.DisplayName, Token: token})
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
		fmt.Printf("  Short Code: %s\n", lobby.ShortCode)
		fmt.Println()
		fmt.Println("  Click any link to auto-login as that user:")
		for _, u := range users {
			fmt.Printf("  - %s: %s/lobby/%s?token=%s\n", u.DisplayName, frontendURL, lobby.ShortCode, u.Token)
		}
		fmt.Println()
		fmt.Println("  Next steps:")
		fmt.Println("  1. Click a link above to auto-login and join")
		fmt.Println("  2. Click 'Ready Up'")
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
		fmt.Printf("  Short Code: %s\n", lobby.ShortCode)
		fmt.Println()
		fmt.Println("  Click any link to auto-login as that user:")
		for _, u := range users {
			fmt.Printf("  - %s: %s/lobby/%s?token=%s\n", u.DisplayName, frontendURL, lobby.ShortCode, u.Token)
		}
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
	fmt.Printf("  Short Code: %s\n", lobby.ShortCode)
	fmt.Printf("  Lobby ID:   %s\n", lobby.ID)
	fmt.Println()
	fmt.Println("  Click any link to auto-login as that user:")
	for _, u := range users {
		fmt.Printf("  - %s: %s/lobby/%s?token=%s\n", u.DisplayName, frontendURL, lobby.ShortCode, u.Token)
	}
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

	var users []SimUser
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

		users = append(users, SimUser{DisplayName: user.DisplayName, Token: token})
		fmt.Printf("  [%d/%d] %s joined\n", i+1, *count, user.DisplayName)
	}

	fmt.Println()
	fmt.Println("Done! Click any link to auto-login as that user:")
	for _, u := range users {
		fmt.Printf("  - %s: %s/lobby/%s?token=%s\n", u.DisplayName, frontendURL, *lobbyCode, u.Token)
	}
	fmt.Println()
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

func voteCmd(apiURL string, args []string) {
	fs := flag.NewFlagSet("vote", flag.ExitOnError)
	lobbyCode := fs.String("lobby", "", "Lobby ID or short code (required)")
	option := fs.Int("option", 1, "Match option to vote for (1-8)")
	random := fs.Bool("random", false, "Have each player vote randomly")
	password := fs.String("password", "asdf", "Password for the users")
	fs.Parse(args)

	if *lobbyCode == "" {
		fmt.Println("Error: --lobby is required")
		fmt.Println("\nUsage: simulator vote --lobby=ABC123 --option=2")
		fmt.Println("       simulator vote --lobby=ABC123 --random")
		os.Exit(1)
	}

	client := NewAPIClient(apiURL)

	fmt.Printf("=== Voting Simulator for lobby %s ===\n\n", *lobbyCode)

	// Get lobby info first
	lobby, err := client.GetLobby(*lobbyCode)
	if err != nil {
		fmt.Printf("Failed to get lobby: %v\n", err)
		os.Exit(1)
	}

	if !lobby.VotingEnabled {
		fmt.Println("Error: Voting is not enabled for this lobby")
		fmt.Println("Create a lobby with voting enabled to use this command")
		os.Exit(1)
	}

	fmt.Printf("Lobby status: %s\n", lobby.Status)
	fmt.Printf("Voting mode: %s\n", lobby.VotingMode)
	fmt.Printf("Current players: %d\n\n", len(lobby.Players))

	if lobby.Status != "matchmaking" {
		fmt.Printf("Lobby is in '%s' status, not 'matchmaking'\n", lobby.Status)
		fmt.Println("Teams must be generated before voting can happen")
		fmt.Println()
		fmt.Printf("Lobby URL: %s/lobby/%s\n", frontendURL, lobby.ShortCode)
		return
	}

	// Login as each existing player and cast votes
	fmt.Println("Logging in as existing players and casting votes...")

	var successCount int
	for i, player := range lobby.Players {
		// Login as this player
		token, err := client.Login(player.DisplayName, *password)
		if err != nil {
			fmt.Printf("  [%d/%d] %s - FAILED to login: %v\n", i+1, len(lobby.Players), player.DisplayName, err)
			continue
		}

		voteOption := *option
		if *random {
			// Distribute votes across options
			voteOption = (i % 8) + 1
		}

		status, err := client.Vote(token, lobby.ID, voteOption)
		if err != nil {
			fmt.Printf("  [%d/%d] %s - FAILED to vote: %v\n", i+1, len(lobby.Players), player.DisplayName, err)
			continue
		}

		successCount++
		fmt.Printf("  [%d/%d] %s voted for option %d (total: %d/%d)\n",
			i+1, len(lobby.Players), player.DisplayName, voteOption, status.VotesCast, status.TotalPlayers)
	}

	// Get final status
	fmt.Println()
	if successCount > 0 {
		// Login as first player to get status
		token, _ := client.Login(lobby.Players[0].DisplayName, *password)
		status, err := client.GetVotingStatus(token, lobby.ID)
		if err != nil {
			fmt.Printf("Failed to get final status: %v\n", err)
		} else {
			fmt.Println("=========================================")
			fmt.Println("  VOTING STATUS")
			fmt.Println("=========================================")
			fmt.Printf("  Total votes: %d/%d\n", status.VotesCast, status.TotalPlayers)
			fmt.Printf("  Can finalize: %v\n", status.CanFinalize)
			if status.WinningOption != nil {
				fmt.Printf("  Winning option: %d\n", *status.WinningOption)
			}
			fmt.Println()
			fmt.Println("Vote counts:")
			for opt, count := range status.VoteCounts {
				fmt.Printf("  Option %s: %d votes\n", opt, count)
			}
		}
	}

	fmt.Println()
	fmt.Printf("Lobby URL: %s/lobby/%s\n", frontendURL, lobby.ShortCode)
}

func endVotingCmd(apiURL string, args []string) {
	fs := flag.NewFlagSet("end-voting", flag.ExitOnError)
	lobbyCode := fs.String("lobby", "", "Lobby ID or short code (required)")
	force := fs.Int("force", 0, "Force select this option number (captain_override mode only)")
	user := fs.String("user", "", "Username to login as (must be captain)")
	password := fs.String("password", "asdf", "Password for user")
	fs.Parse(args)

	if *lobbyCode == "" {
		fmt.Println("Error: --lobby is required")
		fmt.Println("\nUsage: simulator end-voting --lobby=ABC123")
		fmt.Println("       simulator end-voting --lobby=ABC123 --force=3")
		os.Exit(1)
	}

	client := NewAPIClient(apiURL)

	fmt.Printf("=== End Voting for lobby %s ===\n\n", *lobbyCode)

	// Get lobby info
	lobby, err := client.GetLobby(*lobbyCode)
	if err != nil {
		fmt.Printf("Failed to get lobby: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Lobby status: %s\n", lobby.Status)
	fmt.Printf("Voting mode: %s\n", lobby.VotingMode)

	// Need to authenticate as captain
	var token string
	if *user != "" {
		// Login as specified user
		token, err = client.Login(*user, *password)
		if err != nil {
			fmt.Printf("Failed to login as %s: %v\n", *user, err)
			os.Exit(1)
		}
		fmt.Printf("Logged in as: %s\n", *user)
	} else {
		// Create a new user and try (this won't work as they won't be captain)
		fmt.Println("\nNote: You need to specify --user to login as the captain")
		fmt.Println("Example: simulator end-voting --lobby=ABC123 --user=LobbyAdmin_12345")
		os.Exit(1)
	}

	// Get voting status
	status, err := client.GetVotingStatus(token, lobby.ID)
	if err != nil {
		fmt.Printf("Failed to get voting status: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nCurrent votes: %d/%d\n", status.VotesCast, status.TotalPlayers)
	if status.WinningOption != nil {
		fmt.Printf("Current winner: Option %d\n", *status.WinningOption)
	}
	fmt.Printf("Can finalize: %v\n", status.CanFinalize)

	// End voting
	var forceOption *int
	if *force > 0 {
		forceOption = force
		fmt.Printf("\nForce-selecting option %d...\n", *force)
	} else {
		fmt.Println("\nEnding voting (selecting winner)...")
	}

	updatedLobby, err := client.EndVoting(token, lobby.ID, forceOption)
	if err != nil {
		fmt.Printf("Failed to end voting: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("OK!")
	fmt.Printf("\nLobby status: %s\n", updatedLobby.Status)
	if updatedLobby.SelectedMatchOption != nil {
		fmt.Printf("Selected option: %d\n", *updatedLobby.SelectedMatchOption)
	}
	fmt.Println()
	fmt.Printf("Lobby URL: %s/lobby/%s\n", frontendURL, updatedLobby.ShortCode)
}
