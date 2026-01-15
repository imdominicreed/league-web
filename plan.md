# Match History Feature

Add a match history feature that shows completed drafts - users can see what champions were picked/banned and who was on which team after a draft completes.

## Current State Analysis

### Existing Infrastructure
- **Room entity** (`internal/domain/room.go`): Has `Status`, `CompletedAt`, `StartedAt` fields - supports tracking completion
- **DraftState entity** (`internal/domain/draft.go`): Stores `BlueBans`, `RedBans`, `BluePicks`, `RedPicks` as JSON, has `IsComplete` flag
- **DraftAction entity** (`internal/domain/draft.go`): Exists for audit trail but NOT currently being populated
- **RoomPlayer entity** (`internal/domain/matchmaking.go`): Stores team/role assignments for team drafts
- **RoomRepository**: Already has `GetByUserID()` method that retrieves user's rooms

### Critical Gaps to Fix
1. **Draft completion doesn't persist `Room.CompletedAt`** - WebSocket marks `DraftState.IsComplete` but doesn't update Room
2. **DraftAction records not created** - `RecordAction()` exists but never called during draft
3. **No API endpoint for completed matches** - Need filtered endpoint for match history
4. **No frontend UI for viewing history** - Need new page and components

### Patterns to Follow
- Repository: Context-aware GORM with Preload for relations
- Handlers: Chi router params, middleware auth, JSON responses
- Frontend: Redux slices, API clients in `src/api/`, pages with loading states

## Goals

1. Persist draft completion data properly (Room.CompletedAt, DraftActions)
2. Create API endpoint to fetch user's completed matches with full draft data
3. Build frontend page showing match history list
4. Build match detail view showing picks/bans timeline and team compositions

## Implementation Plan

### Phase 1: Backend - Fix Draft Completion Persistence

- [ ] Update `internal/websocket/room.go` to persist `Room.Status = "completed"` and `Room.CompletedAt` when draft finishes
- [ ] Update `internal/websocket/room.go` to record `DraftAction` entries when champions are locked in
- [ ] Add `DraftActionRepository` to repository interfaces if not exists
- [ ] Wire up draft action recording in the WebSocket room

### Phase 2: Backend - Match History API

- [ ] Add `GetCompletedByUserID(ctx, userID, limit, offset)` method to `RoomRepository` interface
- [ ] Implement the method in `internal/repository/postgres/room_repo.go` - filter by status="completed" and include DraftState
- [ ] Create `internal/api/handlers/match_history.go` with:
  - `GET /api/v1/match-history` - list user's completed matches
  - `GET /api/v1/match-history/:roomId` - get single match detail with full draft data
- [ ] Add routes to `cmd/server/main.go`

### Phase 3: Frontend - API Client and Types

- [ ] Add match history types to `frontend/src/types/index.ts`:
  - `MatchHistoryItem` (summary for list view)
  - `MatchDetail` (full draft data for detail view)
- [ ] Create `frontend/src/api/matchHistory.ts` with API client methods
- [ ] Add `matchHistorySlice` to Redux store (optional - could use local state)

### Phase 4: Frontend - Match History Page

- [ ] Create `frontend/src/pages/MatchHistory.tsx` - list of completed matches
- [ ] Create `frontend/src/components/match-history/MatchHistoryCard.tsx` - summary card for each match
- [ ] Add route to `frontend/src/App.tsx`
- [ ] Add navigation link to match history (header/home page)

### Phase 5: Frontend - Match Detail View

- [ ] Create `frontend/src/pages/MatchDetail.tsx` - full draft breakdown
- [ ] Create `frontend/src/components/match-history/DraftTimeline.tsx` - shows pick/ban order
- [ ] Create `frontend/src/components/match-history/TeamComposition.tsx` - shows final team with champions
- [ ] Reuse existing champion image components from draft UI

### Phase 6: Testing and Polish

- [ ] Add backend integration tests for match history endpoints
- [ ] Test draft completion persistence with existing draft tests
- [ ] Manual E2E testing of full flow
- [ ] Add empty state for users with no match history

## Files to Modify

| File | Changes |
|------|---------|
| `internal/websocket/room.go` | Persist Room.CompletedAt and Room.Status on draft completion |
| `internal/websocket/room.go` | Record DraftAction on lock-in |
| `internal/repository/interfaces.go` | Add GetCompletedByUserID to RoomRepository |
| `internal/repository/postgres/room_repo.go` | Implement GetCompletedByUserID with DraftState preload |
| `cmd/server/main.go` | Add match history routes |
| `frontend/src/App.tsx` | Add match history route |

## Files to Create

| File | Purpose |
|------|---------|
| `internal/api/handlers/match_history.go` | Match history REST endpoints |
| `frontend/src/api/matchHistory.ts` | API client for match history |
| `frontend/src/pages/MatchHistory.tsx` | Match history list page |
| `frontend/src/pages/MatchDetail.tsx` | Single match detail page |
| `frontend/src/components/match-history/MatchHistoryCard.tsx` | Summary card component |
| `frontend/src/components/match-history/DraftTimeline.tsx` | Pick/ban timeline component |
| `frontend/src/components/match-history/TeamComposition.tsx` | Team display component |

## Success Criteria

1. When a draft completes, `Room.CompletedAt` is set and `Room.Status` is "completed"
2. `GET /api/v1/match-history` returns list of user's completed drafts
3. `GET /api/v1/match-history/:roomId` returns full draft data including picks/bans
4. Match history page displays list of completed matches with:
   - Date/time of match
   - Draft mode (pro_play/fearless)
   - Which side user was on
   - Final team compositions (champion icons)
5. Match detail page shows:
   - Full pick/ban timeline in order
   - Both team compositions with player names (for team drafts)
   - Champion images for all picks/bans
6. Navigation to match history is accessible from main UI
