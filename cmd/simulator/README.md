# Lobby Simulator

A CLI tool for testing 10-man lobbies with matchmaking and voting.

## Quick Start

```bash
# From project root
go run ./cmd/simulator/... <command> [options]

# Or build first
go build -o simulator ./cmd/simulator/...
./simulator <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| `full` | Create a lobby with fake users, ready all, generate teams |
| `populate` | Add fake users to an existing lobby |
| `ready` | Set all players in a lobby to ready |
| `vote` | Have fake users vote for a match option |
| `end-voting` | End voting and select option (captain can force) |
| `help` | Show help message |

## Voting Flow

### 1. Create a Lobby with Voting Enabled

```bash
# Create lobby with majority voting (>50% wins)
go run ./cmd/simulator/... full --voting

# Create lobby with unanimous voting (100% required)
go run ./cmd/simulator/... full --voting --voting-mode=unanimous

# Create lobby with captain override (voting + captain can force)
go run ./cmd/simulator/... full --voting --voting-mode=captain_override
```

**Voting Modes:**

| Mode | Description |
|------|-------------|
| `majority` | Option with >50% of votes wins |
| `unanimous` | All players must vote for same option |
| `captain_override` | Players vote, but captain can force-select any option |

### 2. Create Lobby Without Auto-Completing

Use `--skip-ready` to create a lobby that doesn't auto-complete the flow:

```bash
go run ./cmd/simulator/... full --count=10 --voting --voting-mode=captain_override --skip-ready
```

This creates 10 players but doesn't ready them or generate teams, so you can test the full UI flow.

### 3. Cast Votes

Once teams are generated (lobby in `matchmaking` status), players can vote:

```bash
# All fake users vote for option 2
go run ./cmd/simulator/... vote --lobby=ABC123 --option=2

# Each fake user votes randomly
go run ./cmd/simulator/... vote --lobby=ABC123 --random
```

### 4. End Voting (Captain Override)

The captain can end voting and either accept the winner or force a specific option:

```bash
# Accept the current winning option
go run ./cmd/simulator/... end-voting --lobby=ABC123 --user=LobbyAdmin_12345

# Force-select option 3 (captain_override mode only)
go run ./cmd/simulator/... end-voting --lobby=ABC123 --user=LobbyAdmin_12345 --force=3
```

## Example: Full Voting Flow

```bash
# 1. Create a lobby with captain_override voting
go run ./cmd/simulator/... full --count=10 --voting --voting-mode=captain_override --skip-ready

# Output:
#   Lobby created: ... (code: ABC123, voting: captain_override)
#   Captain: LobbyAdmin_12345

# 2. Open the lobby in browser
#    $FRONTEND_URL/lobby/ABC123 (e.g., http://lobby-voting.dev.local:3000/lobby/ABC123)

# 3. Login as captain (LobbyAdmin_12345 / asdf)
#    - Click "Ready Up" for all players (or do it via UI)
#    - Click "Generate Teams"

# 4. Have players vote
go run ./cmd/simulator/... vote --lobby=ABC123 --option=2

# 5. Captain force-selects a different option
go run ./cmd/simulator/... end-voting --lobby=ABC123 --user=LobbyAdmin_12345 --force=3

# 6. Start the draft in the UI
```

## Other Commands

### Create Lobby with Open Slots

```bash
# 9 fake players, 1 slot for you to join
go run ./cmd/simulator/... full

# 5 fake players, 5 slots open
go run ./cmd/simulator/... full --count=5
```

### Add Players to Existing Lobby

```bash
go run ./cmd/simulator/... populate --lobby=ABC123 --count=5
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `API_URL` | `http://localhost:9999` | Backend API URL |

## Default Credentials

All fake users are created with password: `asdf`
