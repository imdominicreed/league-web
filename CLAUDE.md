# League Draft Website

Real-time League of Legends Pro Play Pick/Ban drafting simulator with Go backend and React frontend.

## Quick Reference

```bash
# Start database
make db

# Start backend (port 9090)
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

Required:
- `JWT_SECRET` - Secret key for JWT signing
- `DATABASE_URL` - PostgreSQL connection string

Optional:
- `PORT` - Server port (default: 9090)
- `DEFAULT_TIMER_SECONDS` - Draft timer (default: 30)
- `DDRAGON_VERSION` - Lock to specific patch version

## Testing

```bash
go test ./...                   # Backend tests
cd frontend && npm run lint     # Frontend linting
```

## Notes

- Champion data synced from Riot Data Dragon CDN
- Schema supports Fearless mode (series_id, fearless_bans table) but UI not yet implemented
- Frontend uses path alias `@/` for `src/`
