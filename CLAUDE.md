# League Draft Website

Real-time League of Legends Pro Play Pick/Ban drafting simulator with Go backend and React frontend.

## Devcontainer Environment

You are running inside a devcontainer with everything pre-configured.

### Pre-Running Services

| Service | Internal URL | External URL |
|---------|--------------|--------------|
| PostgreSQL | `db:5432` | N/A (container only) |
| Backend | `localhost:9999` | `http://<project>.dev.local:9999` |
| Frontend | `localhost:3000` | `http://<project>.dev.local:3000` |

**Database is already running** - no need to start it manually. Connection string:
```
postgres://postgres:postgres@db:5432/league_draft
```

### Available Tools

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.23+ | `go build`, `go test`, `go run` |
| Node.js | 20+ | Via fnm, includes npm |
| Docker | Host access | For testcontainers, via mounted socket |
| Git | Latest | Full access to host repo |
| Make | Latest | Project automation |
| tmux | Latest | Session management |
| Fish shell | Default | With starship prompt |

### Working Directory

```
/workspaces/project/
```

## Quick Reference

```bash
# Start backend (port 9999) - connects to already-running db
make dev-backend

# Start frontend (port 3000)
cd frontend && npm run dev

# Start both in tmux split
make dev

# Sync champion data from Riot
make sync-champions

# Run all backend tests (Docker available for testcontainers)
go test ./...
```

## Architecture

- **Backend**: Go with Chi router, Gorilla WebSocket, GORM + PostgreSQL
- **Frontend**: React + TypeScript + Vite, Redux Toolkit, Tailwind CSS
- **Real-time**: WebSocket hub pattern for live draft state sync

## Project Structure

```
cmd/server/main.go              # Entry point
internal/
  api/handlers/                 # REST handlers (auth, room, champion, profile, lobby)
  api/middleware/               # Auth JWT, CORS
  config/                       # Environment config
  domain/                       # Entities: User, Room, DraftState, Champion, Lobby, RoomPlayer
  repository/postgres/          # GORM repositories
  service/                      # Business logic (auth, room, lobby, matchmaking, profile)
  websocket/                    # Hub (room manager), Client, Room (draft state machine)

frontend/src/
  api/                          # REST client (auth, profile, lobby)
  components/draft/             # DraftBoard, ChampionGrid, TeamPanel, Timer
  components/lobby/             # LobbyPlayerGrid, MatchOptionCard
  components/profile/           # RoleProfileEditor
  hooks/useWebSocket.ts         # WebSocket connection + message handling
  store/slices/                 # Redux: authSlice, draftSlice, championsSlice, roomSlice, lobbySlice, profileSlice
  pages/                        # Home, Login, Register, CreateDraft, JoinDraft, DraftRoom, Profile, CreateLobby, LobbyRoom
```

## Key Files

| File | Purpose |
|------|---------|
| `internal/websocket/hub.go` | Manages WebSocket rooms, client registration, team draft detection |
| `internal/websocket/room.go` | Draft state machine, timer, phase transitions, captain-only picking |
| `internal/domain/draft.go` | Phase sequence (20 phases for pro play) |
| `internal/domain/lobby.go` | Lobby and LobbyPlayer entities |
| `internal/domain/matchmaking.go` | MatchOption, MatchOptionAssignment, RoomPlayer entities |
| `internal/service/matchmaking_service.go` | Multi-algorithm team balancing (MMR, comfort, hybrid, lane) |
| `internal/service/lobby_service.go` | Lobby lifecycle management |
| `frontend/src/hooks/useWebSocket.ts` | WS connection, message dispatch to Redux |
| `frontend/src/components/draft/DraftBoard.tsx` | Main draft UI composition |
| `frontend/src/components/draft/TeamPanel.tsx` | Team display (1v1 or 5v5 mode) |
| `frontend/src/pages/LobbyRoom.tsx` | 10-man lobby UI with matchmaking |

## Draft Phase Sequence

Pro play uses 20 phases: 6 bans → 6 picks → 4 bans → 4 picks

```
Phases 0-5:   Ban (B-R-B-R-B-R)
Phases 6-11:  Pick (B-R-R-B-B-R)
Phases 12-15: Ban (R-B-R-B)
Phases 16-19: Pick (R-B-B-R)
```

## WebSocket Protocol

Client → Server: `JOIN_ROOM`, `SELECT_CHAMPION`, `LOCK_IN`, `HOVER_CHAMPION`, `READY`, `START_DRAFT`

Server → Client: `STATE_SYNC`, `PLAYER_UPDATE`, `PHASE_CHANGED`, `CHAMPION_SELECTED`, `TIMER_TICK`, `DRAFT_COMPLETED`

## 10-Man Lobby System

The lobby system enables 10-player team drafts with role-based matchmaking.

### Lobby Flow

1. Creator creates lobby with draft mode (pro_play/fearless) and timer settings
2. Players join via lobby code, set ready status
3. When all 10 players ready, creator generates team options
4. Matchmaking algorithm creates up to 8 balanced team compositions (using 4 different algorithms)
5. Creator selects preferred option (each shows algorithm type badge and key metrics)
6. "Start Draft" creates a Room with team assignments
7. Players auto-join correct side, captains handle picks/bans

### Key Domain Entities

| Entity | Purpose |
|--------|---------|
| `Lobby` | 10-player lobby with status, settings, short code |
| `LobbyPlayer` | Player in lobby with ready status |
| `UserRoleProfile` | Per-role rank (Iron IV - Challenger), MMR, comfort rating (1-5) |
| `MatchOption` | Generated team composition with algorithm type, balance score, comfort averages, max lane diff |
| `MatchOptionAssignment` | Player's team and role in an option |
| `RoomPlayer` | Player assigned to draft room with team/role/captain status |

### Matchmaking Algorithm

Located in `internal/service/matchmaking_service.go`:

Uses a **multi-algorithm approach** to generate diverse, high-quality team options:

1. Load all 10 players' role profiles (MMR + comfort per role)
2. Generate C(10,5) = 252 team splits
3. For each split, try all 120 × 120 = 14,400 role permutations (both teams optimized)
4. Score each composition with 4 different algorithms
5. Return top 8 unique options (2 best from each algorithm, deduplicated)

**Scoring Algorithms** (`AlgorithmType`):

| Algorithm | Focus | Best For |
|-----------|-------|----------|
| `mmr_balanced` | Minimize team MMR difference | Competitive games with similar skill players |
| `role_comfort` | Maximize player comfort ratings | Games where players want their main roles |
| `hybrid` | Balance MMR and comfort equally | General-purpose matchmaking |
| `lane_balanced` | Minimize worst lane matchup | Wide skill range lobbies (prevents stomps) |

**Scoring Details**:
- `mmr_balanced`: Heavy penalty for MMR diff, light comfort penalty
- `role_comfort`: Heavy comfort penalty (exponential), light MMR penalty
- `hybrid`: Balanced penalties for both factors
- `lane_balanced`: Penalizes max single-lane MMR gap to prevent unfair matchups

When MMR range > 1000, `lane_balanced` is prioritized in final sorting.

### Lobby API Endpoints

| Endpoint | Purpose |
|----------|---------|
| `POST /api/v1/lobbies` | Create new lobby |
| `GET /api/v1/lobbies/:idOrCode` | Get lobby with players |
| `POST /api/v1/lobbies/:id/join` | Join lobby |
| `POST /api/v1/lobbies/:id/leave` | Leave lobby |
| `POST /api/v1/lobbies/:id/ready` | Set ready status |
| `POST /api/v1/lobbies/:id/generate-teams` | Generate matchmaking options |
| `GET /api/v1/lobbies/:id/match-options` | Get generated options |
| `POST /api/v1/lobbies/:id/select-option` | Select team composition |
| `POST /api/v1/lobbies/:id/start-draft` | Create room and start draft |

### Profile API Endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /api/v1/profile` | Get user profile |
| `GET /api/v1/profile/roles` | Get all 5 role profiles |
| `PUT /api/v1/profile/roles/:role` | Update rank/MMR/comfort for a role |
| `POST /api/v1/profile/roles/initialize` | Create default profiles for all roles |

### Team Draft Mode

When a room is created from a lobby:
- `Room.IsTeamDraft = true` and `Room.LobbyID` is set
- `RoomPlayer` entries store team/role assignments
- WebSocket hub auto-assigns client sides based on `RoomPlayer.Team`
- Only captains (first player per team in role order) can pick/ban
- `TeamPanel` displays 5 players with role icons (TOP, JGL, MID, ADC, SUP)

## Environment Variables

Environment is pre-configured in the devcontainer. A `.env` file exists at project root.

### Backend Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `DATABASE_URL` | `postgres://postgres:postgres@db:5432/league_draft` | PostgreSQL connection |
| `JWT_SECRET` | (set in .env) | JWT signing key |
| `PORT` | `9999` | Server port |
| `ENVIRONMENT` | `development` | Environment name |
| `JWT_EXPIRATION_HOURS` | `24` | Token expiration |
| `DEFAULT_TIMER_SECONDS` | `30` | Draft timer |
| `DDRAGON_VERSION` | (empty = latest) | Lock to specific patch |

### Frontend Variables (`frontend/.env`)

| Variable | Default | Purpose |
|----------|---------|---------|
| `VITE_API_URL` | `http://localhost:9999` | Backend API URL |

## Testing

### Manual Testing

Use the Playwright MCP CLI for browser automation and manual testing.

### Backend Tests

Docker is available via mounted socket - testcontainers work out of the box.

```bash
go test ./...                          # All tests
go test ./... -v                       # Verbose
go test ./... -race                    # Race detection
go test ./... -short                   # Skip slow tests
go test ./internal/api/handlers/...   # Specific package
```

### Test Framework (`internal/testutil/`)

| Component | Purpose |
|-----------|---------|
| `NewTestDB(t)` | Creates PostgreSQL container with migrations |
| `NewTestServer(t)` | Full HTTP server with hub, services, repos |
| `TestConfig()` | Returns test-appropriate config (fast timers, etc.) |

**Builders** (fluent API for test data):

| Builder | Key Methods |
|---------|-------------|
| `NewUserBuilder()` | `WithDisplayName()`, `Build()`, `BuildAndAuthenticate()` |
| `NewRoomBuilder()` | `WithCreator()`, `WithDraftMode()`, `Build()`, `BuildWithHub()` |
| `NewChampionBuilder()` | `WithID()`, `WithName()`, `WithTags()`, `Build()` |
| `NewLobbyBuilder()` | `WithCreator()`, `WithStatus()`, `Build()`, `BuildWithPlayers()` |
| `NewUserRoleProfileBuilder()` | `WithUser()`, `WithRole()`, `WithRank()`, `Build()` |

**Seeding helpers**: `SeedChampions()`, `SeedRealChampions()`, `SeedLobbyWith10Players()`, `SeedLobbyWith10ReadyPlayers()`

**WebSocket client** (`WSClient`): `JoinRoom()`, `Ready()`, `StartDraft()`, `SelectChampion()`, `LockIn()`, `ExpectStateSync()`, `ExpectPhaseChanged()`, `ExpectError()`

**Assertions**: `AssertStatusCode()`, `AssertJSONResponse()`, `AssertErrorResponse()`, `AssertContainsChampion()`

## Notes

- Champion data synced from Riot Data Dragon CDN
- Schema supports Fearless mode (series_id, fearless_bans table) but UI not yet implemented
- Frontend uses path alias `@/` for `src/`
- Authentication uses JWT tokens with configurable expiration
- WebSocket auto-reconnects after 3 seconds on disconnect
- Database migrations run automatically on backend startup
