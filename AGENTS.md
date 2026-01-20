## Build & Run

### Backend (Go)
```bash
make dev-backend          # Start backend on port 9999
# OR
go run cmd/server/main.go # Direct run
```

### Frontend (React)
```bash
cd frontend && npm run dev  # Start frontend on port 3000
```

### Both (tmux)
```bash
make dev  # Starts both in tmux split
```

## Validation

Run these after implementing fixes:

- **Backend Tests**: `go test ./...`
- **Backend Tests (verbose)**: `go test ./... -v`
- **Frontend Typecheck**: `cd frontend && npm run typecheck`
- **Frontend Lint**: `cd frontend && npm run lint`

## Health Check

- Backend health: `curl http://localhost:9999/health`

## E2E Testing (Playwright CLI)

Bug fix verification uses Playwright E2E tests:

```bash
# Run all E2E tests
cd frontend && npm run test:e2e

# Run specific bug test
cd frontend && npm run test:e2e -- --grep "BUG-001"

# Run tests in headed mode (see browser)
cd frontend && npm run test:e2e:headed

# Run tests with UI debugger
cd frontend && npm run test:e2e:ui
```

### Test File Structure
```
frontend/e2e/
  bugs/                    # Bug verification tests
    bug-001.spec.ts
    bug-002.spec.ts
  helpers/
    test-utils.ts          # Shared utilities (createTestUser, etc.)
```

### Writing Bug Tests
```typescript
import { test, expect } from '@playwright/test';
import { createTestUser, setAuthToken } from '../helpers/test-utils';

test.describe('BUG-XXX: Description', () => {
  test('should verify the fix works', async ({ page }) => {
    // Setup
    const user = await createTestUser(page, 'bugtest');
    await setAuthToken(page, user);

    // Reproduce steps from bug report
    await page.goto('/some-page');

    // Verify expected behavior
    await expect(page.locator('...')).toBeVisible();
  });
});
```

## Codebase Patterns

### Backend (Go)
- Handlers: `internal/api/handlers/`
- Services: `internal/service/`
- Domain: `internal/domain/`
- WebSocket: `internal/websocket/`

### Frontend (React + TypeScript)
- Pages: `frontend/src/pages/`
- Components: `frontend/src/components/`
- Redux slices: `frontend/src/store/slices/`
- API clients: `frontend/src/api/`
- Hooks: `frontend/src/hooks/`
- Path alias: `@/` maps to `frontend/src/`

### API
- Base URL: `http://localhost:9999/api/v1`
- Auth: JWT in Authorization header
- WebSocket: `ws://localhost:9999/ws`

## Database

PostgreSQL running at `db:5432`:
```
postgres://postgres:postgres@db:5432/league_draft
```

Migrations run automatically on backend startup.
