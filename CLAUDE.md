# League Draft Website

Real-time League of Legends Pro Play Pick/Ban drafting simulator with Go backend and React frontend.

## Quick Reference

```bash
# Start database
make db

# Start backend (port 9999)
make dev-backend

# Start frontend (port 3000)
cd frontend && npm run dev

# Sync champion data from Riot
make sync-champions
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
| `internal/service/matchmaking_service.go` | Team balancing algorithm |
| `internal/service/lobby_service.go` | Lobby lifecycle management |
| `frontend/src/hooks/useWebSocket.ts` | WS connection, message dispatch to Redux |
| `frontend/src/components/draft/DraftBoard.tsx` | Main draft UI composition |
| `frontend/src/components/draft/TeamPanel.tsx` | Team display (1v1 or 5v5 mode) |
| `frontend/src/pages/LobbyRoom.tsx` | 10-man lobby UI with matchmaking |

## Draft Phase Sequence

Pro play uses 20 phases: 6 bans → 4 picks → 4 bans → 6 picks

```
Phases 0-5:   Ban (B-R-B-R-B-R)
Phases 6-9:   Pick (B-R-R-B)
Phases 10-13: Ban (R-B-R-B)
Phases 14-19: Pick (R-B-B-R-B-R)
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
4. Matchmaking algorithm creates 5 balanced team compositions
5. Creator selects preferred option
6. "Start Draft" creates a Room with team assignments
7. Players auto-join correct side, captains handle picks/bans

### Key Domain Entities

| Entity | Purpose |
|--------|---------|
| `Lobby` | 10-player lobby with status, settings, short code |
| `LobbyPlayer` | Player in lobby with ready status |
| `UserRoleProfile` | Per-role rank (Iron IV - Challenger), MMR, comfort rating (1-5) |
| `MatchOption` | Generated team composition with balance score |
| `MatchOptionAssignment` | Player's team and role in an option |
| `RoomPlayer` | Player assigned to draft room with team/role/captain status |

### Matchmaking Algorithm

Located in `internal/service/matchmaking_service.go`:

1. Load all 10 players' role profiles (MMR + comfort per role)
2. Generate C(10,5) = 252 team combinations
3. For each split, find optimal role assignments via permutation
4. Calculate balance score: `100 - (mmrDiff/100) - (comfortPenalty * 1.5)`
5. Return top 5 options sorted by balance score

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

### Backend (`.env` in project root)

Required:
- `JWT_SECRET` - Secret key for JWT signing
- `DATABASE_URL` - PostgreSQL connection string

Optional:
- `PORT` - Server port (default: 9999)
- `ENVIRONMENT` - Environment name (default: development)
- `JWT_EXPIRATION_HOURS` - Token expiration (default: 24)
- `DEFAULT_TIMER_SECONDS` - Draft timer (default: 30)
- `DDRAGON_VERSION` - Lock to specific patch version (auto-fetches latest if empty)

### Frontend (`frontend/.env`)

- `VITE_API_URL` - Backend API URL for dev proxy (default: http://localhost:9999)

## Testing

### Running Tests

```bash
# Backend (requires Docker for testcontainers)
go test ./...                          # All tests
go test ./... -v                       # Verbose
go test ./... -race                    # Race detection
go test ./... -short                   # Skip slow tests

# Frontend E2E (requires backend running)
cd frontend && npx playwright test     # All E2E tests
cd frontend && npx playwright test --ui # Interactive mode
```

### Backend Test Framework (`internal/testutil/`)

Uses **testcontainers-go** for real PostgreSQL instances. Docker must be running.

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

### Frontend E2E Framework (`frontend/e2e/`)

Uses **Playwright** with page objects and multi-user fixtures.

| Fixture | Purpose |
|---------|---------|
| `pages.ts` | Page objects: `HomePage`, `LoginPage`, `RegisterPage`, `CreateLobbyPage`, `LobbyRoomPage` |
| `multi-user.ts` | `createUsers(n)` creates N authenticated browser contexts, `lobbyWithUsers(n)` creates lobby with N users |

**API helpers**: `registerUserViaApi()`, `createLobbyViaApi()`, `joinLobbyViaApi()`, `setReadyViaApi()`, `generateTeams()`, `selectMatchOption()`

## Recent Improvements

- **Request/Response Logging**: API client logs all requests with full URLs and response status
- **WebSocket Error Logging**: Connection errors and message handling logged for debugging
- **Auth Error Logging**: Authentication middleware logs all auth failures
- **WSL Compatibility**: Backend port updated to 9999 for WSL environment

## Notes

- Champion data synced from Riot Data Dragon CDN
- Schema supports Fearless mode (series_id, fearless_bans table) but UI not yet implemented
- Frontend uses path alias `@/` for `src/`
- Authentication uses JWT tokens with configurable expiration
- WebSocket auto-reconnects after 3 seconds on disconnect
