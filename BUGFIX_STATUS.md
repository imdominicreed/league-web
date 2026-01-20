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
| BUG-001 | Medium | PENDING | - | No Logout Button on Home Page |
| BUG-002 | Low | PENDING | - | Login Page Accessible When Authenticated |
| BUG-004 | Medium | PENDING | - | Custom Timer Value Not Sent When Creating Lobby |
| BUG-005 | Medium | PENDING | - | Ready Button Logic Should Be Removed |
| BUG-006 | High | FIXED | Yes | Vote Button Click Does Not Trigger Vote Action |
| BUG-007 | Medium | PENDING | - | Lobby UI Does Not Update in Real-Time After Swap Approval |
| BUG-008 | High | PENDING | - | Kicked Player Receives No Notification or Redirect |
| BUG-009 | Medium | PENDING | - | Captain Indicator Shows for All Players in Lobby UI |
| BUG-010 | High | PENDING | - | Promote Captain Fails After Team Selection |
| BUG-011 | High | PENDING | - | Draft Timer Resets on Unpause Instead of Resuming |

---

## Verification Log

<!-- Append verification details as bugs are fixed -->

### BUG-006 - Vote Button Click Does Not Trigger Vote Action
**Fixed**: 2026-01-20
**Fix Location**: `frontend/src/components/lobby/MatchOptionCard.tsx` (lines 176-181)
**Root Cause**: The vote button had no `onClick` handler - the click handler was only on the parent div.
**Solution**: Added explicit `onClick` handler to the button with `e.stopPropagation()` to prevent double-firing.
**Verified By**: Playwright E2E test `frontend/e2e/bugs/bug-006.spec.ts` (3 tests pass)

