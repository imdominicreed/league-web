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
  api/handlers/                 # REST handlers (auth, room, champion)
  api/middleware/               # Auth JWT, CORS
  config/                       # Environment config
  domain/                       # Entities: User, Room, DraftState, Champion
  repository/postgres/          # GORM repositories
  service/                      # Business logic
  websocket/                    # Hub (room manager), Client, Room (draft state machine)

frontend/src/
  api/                          # REST client
  components/draft/             # DraftBoard, ChampionGrid, TeamPanel, Timer
  hooks/useWebSocket.ts         # WebSocket connection + message handling
  store/slices/                 # Redux: authSlice, draftSlice, championsSlice, roomSlice
  pages/                        # Home, Login, Register, CreateDraft, JoinDraft, DraftRoom
```

## Key Files

| File | Purpose |
|------|---------|
| `internal/websocket/hub.go` | Manages WebSocket rooms, client registration |
| `internal/websocket/room.go` | Draft state machine, timer, phase transitions |
| `internal/domain/draft.go` | Phase sequence (20 phases for pro play) |
| `frontend/src/hooks/useWebSocket.ts` | WS connection, message dispatch to Redux |
| `frontend/src/components/draft/DraftBoard.tsx` | Main draft UI composition |

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
# Run all tests (requires Docker for testcontainers)
go test ./...

# Run with verbose output
go test ./... -v

# Run specific package
go test ./internal/api/handlers/...
go test ./internal/service/...
go test ./internal/repository/postgres/...
go test ./internal/websocket/...

# Run with race detection
go test ./... -race

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Skip slow tests (full draft flow)
go test ./... -short

# Frontend linting
cd frontend && npm run lint
```

### Test Infrastructure

Tests use **testcontainers-go** for real PostgreSQL instances. Docker must be running.

**Key test utilities** in `internal/testutil/`:

| File | Purpose |
|------|---------|
| `testutil.go` | `TestDB` (PostgreSQL container), `TestServer` (full HTTP server) |
| `fixtures.go` | `UserBuilder`, `RoomBuilder`, `ChampionBuilder` for test data |
| `assertions.go` | Custom assertions like `AssertStatusCode`, `AssertJSONResponse` |
| `ws_client.go` | WebSocket test client for draft flow testing |

### Writing New Tests

**1. Repository Tests** (`internal/repository/postgres/*_test.go`):
```go
func TestMyRepo_Method(t *testing.T) {
    testDB := testutil.NewTestDB(t)  // Creates PostgreSQL container
    repo := postgres.NewMyRepository(testDB.DB)
    ctx := context.Background()

    // Create test data using builders
    user, _ := testutil.NewUserBuilder().
        WithDisplayName("testuser").
        Build(t, testDB.DB)

    // Test the method
    result, err := repo.SomeMethod(ctx, user.ID)
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

**2. Service Tests** (`internal/service/*_test.go`):
```go
func TestMyService_Method(t *testing.T) {
    testDB := testutil.NewTestDB(t)
    repos := postgres.NewRepositories(testDB.DB)
    cfg := testutil.TestConfig()
    svc := service.NewMyService(repos.X, cfg)

    tests := []struct {
        name    string
        input   SomeInput
        wantErr error
    }{
        {name: "success", input: validInput},
        {name: "error case", input: badInput, wantErr: service.ErrSomething},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := svc.Method(ctx, tt.input)
            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
                return
            }
            require.NoError(t, err)
            // assertions...
        })
    }
}
```

**3. Handler Tests** (`internal/api/handlers/*_test.go`):
```go
func TestMyHandler_Endpoint(t *testing.T) {
    ts := testutil.NewTestServer(t)  // Full HTTP server with all dependencies

    // Authenticate a user
    _, token := testutil.NewUserBuilder().
        WithDisplayName("testuser").
        BuildAndAuthenticate(t, ts)

    // Make authenticated request
    req := testutil.CreateAuthenticatedRequest(t, "POST", ts.APIURL("/path"), body, token)
    client := &http.Client{}
    resp, err := client.Do(req)
    require.NoError(t, err)

    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

**4. WebSocket Tests** (`internal/websocket/*_test.go`):
```go
func TestDraft_Scenario(t *testing.T) {
    ts := testutil.NewTestServer(t)

    _, blueToken := testutil.NewUserBuilder().BuildAndAuthenticate(t, ts)
    _, redToken := testutil.NewUserBuilder().BuildAndAuthenticate(t, ts)

    room := testutil.NewRoomBuilder().BuildWithHub(t, ts)
    testutil.SeedRealChampions(t, ts.DB.DB)

    blueClient := testutil.NewWSClient(t, ts.WebSocketURL(blueToken))
    redClient := testutil.NewWSClient(t, ts.WebSocketURL(redToken))

    blueClient.JoinRoom(room.ID.String(), "blue")
    blueClient.ExpectStateSync(5 * time.Second)

    redClient.JoinRoom(room.ID.String(), "red")
    redClient.ExpectStateSync(5 * time.Second)

    // Ready up and start draft
    blueClient.Ready(true)
    redClient.Ready(true)
    time.Sleep(100 * time.Millisecond)
    blueClient.DrainMessages()
    redClient.DrainMessages()

    blueClient.StartDraft()
    // Test draft actions...
}
```

### Test Coverage Goals

| Package | Focus Areas |
|---------|-------------|
| `repository/postgres` | CRUD operations, edge cases |
| `service` | Business logic, validation, error cases |
| `api/handlers` | HTTP status codes, auth, request validation |
| `websocket` | Draft state machine, phase transitions, error handling |

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
