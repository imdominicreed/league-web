# Bug Fix Status

Track bug fix progress and verification results.

## Status Legend
- `PENDING` - Not yet addressed
- `IN_PROGRESS` - Currently being fixed
- `FIXED` - Implemented and verified via Playwright E2E test
- `BLOCKED` - Cannot fix (document reason)

---

## Bug Status

| Bug ID | Severity | Status | Verified | Notes |
|--------|----------|--------|----------|-------|
| BUG-001 | Medium | FIXED | Yes | No Logout Button on Home Page |
| BUG-002 | Low | PENDING | - | Login Page Accessible When Authenticated |
| BUG-004 | Medium | FIXED | Yes | Custom Timer Value Not Sent When Creating Lobby |
| BUG-005 | Medium | PENDING | - | Ready Button Logic Should Be Removed |
| BUG-006 | High | FIXED | Yes | Vote Button Click Does Not Trigger Vote Action |
| BUG-007 | Medium | PENDING | - | Lobby UI Does Not Update in Real-Time After Swap Approval |
| BUG-008 | High | FIXED | Yes | Kicked Player Receives No Notification or Redirect |
| BUG-009 | Medium | PENDING | - | Captain Indicator Shows for All Players in Lobby UI |
| BUG-010 | High | FIXED | Yes | Promote Captain Fails After Team Selection |
| BUG-011 | High | FIXED | Yes | Draft Timer Resets on Unpause Instead of Resuming |

---

## Verification Log

<!-- Append verification details as bugs are fixed -->

### BUG-006 - Vote Button Click Does Not Trigger Vote Action
**Fixed**: 2026-01-20
**Fix Location**: `frontend/src/components/lobby/MatchOptionCard.tsx` (lines 176-181)
**Root Cause**: The vote button had no `onClick` handler - the click handler was only on the parent div.
**Solution**: Added explicit `onClick` handler to the button with `e.stopPropagation()` to prevent double-firing.
**Verified By**: Playwright E2E test `frontend/e2e/bugs/bug-006.spec.ts` (3 tests pass)

### BUG-008 - Kicked Player Receives No Notification or Redirect
**Fixed**: 2026-01-20
**Fix Location**: `frontend/src/hooks/useLobbyWebSocket.ts` (lines 202-210) and `frontend/src/pages/LobbyRoom.tsx` (lines 76-86)
**Root Cause**: The `handlePlayerKicked` callback only removed the player from Redux state but didn't check if the kicked player was the current user. No notification or redirect logic existed.
**Solution**:
1. Added `kicked` state to `useLobbyWebSocket` hook that tracks when the current user is kicked
2. In `handlePlayerKicked`, check if `payload.userId === user.id` and set kicked state with kicker's name
3. In `LobbyRoom.tsx`, added useEffect that watches `kicked` state and shows alert then redirects to home
**Verified By**: Playwright E2E test `frontend/e2e/bugs/bug-008.spec.ts` (2 tests pass)

### BUG-010 - Promote Captain Fails After Team Selection
**Fixed**: 2026-01-20
**Fix Location**: `internal/service/lobby_service.go` (line 631-633)
**Root Cause**: The `PromoteCaptain` method had an overly restrictive lobby status check that only allowed promotion in `waiting_for_players` status, while `TakeCaptain` correctly allowed it in any state except `drafting` and `completed`.
**Solution**: Updated the status check in `PromoteCaptain` to use the same logic as `TakeCaptain`:
```go
// Before (too restrictive):
if lobby.Status != domain.LobbyStatusWaitingForPlayers {
    return ErrInvalidLobbyState
}

// After (matches TakeCaptain behavior):
if lobby.Status == domain.LobbyStatusDrafting || lobby.Status == domain.LobbyStatusCompleted {
    return ErrInvalidLobbyState
}
```
**Verified By**: Playwright E2E test `frontend/e2e/bugs/bug-010.spec.ts` (3 tests pass)

### BUG-011 - Draft Timer Resets on Unpause Instead of Resuming
**Fixed**: 2026-01-20
**Fix Location**: `internal/websocket/draft_state.go` (line 288)
**Root Cause**: After resuming from pause, the `durationMs` field in `TimerManager` was set to the remaining time from the pause. When `advancePhase()` was called for subsequent phases, it called `timerMgr.Start()` without resetting the duration to the original timer value. This caused all phases after the first resume to use the wrong (reduced) timer duration.
**Solution**: Added `dm.room.timerMgr.SetDuration(dm.timerDuration)` call before `timerMgr.Start()` in `advancePhase()` to ensure each new phase always starts with the full timer duration:
```go
// Reset timer duration to full duration and start timer for next phase
// This is necessary because SetDuration() may have been called with a
// partial duration when resuming from pause
dm.room.timerMgr.SetDuration(dm.timerDuration)
dm.room.timerMgr.Start()
```
**Verified By**: Playwright E2E test `frontend/e2e/bugs/bug-011.spec.ts` (3 tests pass)

### BUG-001 - No Logout Button on Home Page
**Fixed**: 2026-01-20
**Fix Location**: `frontend/src/pages/Home.tsx` (lines 1-15, 75-81)
**Root Cause**: The Home page displayed a welcome message for authenticated users but provided no logout mechanism. The logout Redux action and API endpoint existed but were not connected to any UI element.
**Solution**:
1. Added `useDispatch` hook and imported `logout` action from authSlice
2. Created `handleLogout` function that dispatches the logout action
3. Added a red "Logout" button at the bottom of the authenticated user's navigation links
4. Button has `data-testid="home-logout-button"` for E2E testing
**Verified By**: Playwright E2E test `frontend/e2e/bugs/bug-001.spec.ts` (3 tests pass)

### BUG-004 - Custom Timer Value Not Sent When Creating Lobby
**Fixed**: 2026-01-20
**Status**: The reported bug could not be reproduced. Testing confirms that custom timer values are correctly sent to the API and stored in the database.
**Verification Details**:
1. E2E test `bug-004.spec.ts` test 1: Captures the API request when creating a lobby via UI with timer=60 - confirms `timerDurationSeconds: 60` is sent
2. E2E test `bug-004.spec.ts` test 2: Creates lobby via API with timer=45 - confirms lobby is created with correct value
3. E2E test `bug-004.spec.ts` test 3: Creates lobby via UI with timer=90 - confirms successful lobby creation
**Technical Analysis**:
- `CreateLobby.tsx` line 14: `useState(30)` initializes timer state
- `CreateLobby.tsx` line 58: `onChange={(e) => setTimerDuration(Number(e.target.value))}` correctly updates state
- `CreateLobby.tsx` line 21: `timerDurationSeconds: timerDuration` passes state to Redux action
- `lobbySlice.ts` line 62-63: Passes data to `lobbyApi.create(data)`
- `lobby.ts` line 18-19: Sends POST request to `/lobbies` with JSON body
- Backend correctly receives and stores the value (validated in test 2)
**Conclusion**: Either the bug was fixed in a previous commit or was an intermittent issue. The functionality now works as expected.
**Verified By**: Playwright E2E test `frontend/e2e/bugs/bug-004.spec.ts` (3 tests pass)

