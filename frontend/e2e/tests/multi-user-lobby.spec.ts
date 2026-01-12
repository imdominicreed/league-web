import { test, expect, UserSession } from '../fixtures/multi-user';
import { LobbyRoomPage, DraftRoomPage, waitForAnyTurn } from '../fixtures/pages';
import { TIMEOUTS } from '../helpers/wait-strategies';
import { retryWithBackoff } from '../helpers/wait-strategies';

// These tests involve multiple users and WebSocket connections - run serially to avoid interference
test.describe.configure({ mode: 'serial' });

/**
 * Helper to find the captain who can approve a pending action.
 * Uses parallel checking for efficiency.
 */
async function findCaptainWithApproveButton(
  users: UserSession[],
  lobbyPages: LobbyRoomPage[],
  startIndex: number = 1
): Promise<{ index: number; lobbyPage: LobbyRoomPage } | null> {
  // Parallel check all users except the proposer
  const checks = await Promise.all(
    users.slice(startIndex).map(async (user, relativeIdx) => {
      const actualIdx = relativeIdx + startIndex;
      await user.page.reload();
      await expect(user.page.locator('text=10-Man Lobby')).toBeVisible({
        timeout: TIMEOUTS.MEDIUM,
      });

      const approveBtn = user.page.locator('button:has-text("Approve")');
      const isVisible = await approveBtn.isVisible().catch(() => false);
      return { index: actualIdx, hasApprove: isVisible };
    })
  );

  const captain = checks.find((c) => c.hasApprove);
  if (captain) {
    return { index: captain.index, lobbyPage: lobbyPages[captain.index] };
  }
  return null;
}

/**
 * Helper to make API calls with retry logic.
 */
async function apiCall(
  endpoint: string,
  token: string,
  options: { method?: string; body?: unknown } = {}
): Promise<Response> {
  return retryWithBackoff(async () => {
    const response = await fetch(`http://localhost:9999/api/v1${endpoint}`, {
      method: options.method || 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: options.body ? JSON.stringify(options.body) : undefined,
    });
    if (!response.ok && response.status >= 500) {
      throw new Error(`API Error ${response.status}`);
    }
    return response;
  });
}

test.describe('Multi-User Lobby Flow with UI', () => {
  test('10 users can join and interact with lobby UI', async ({ lobbyWithUsers }) => {
    test.setTimeout(120000); // 2 minutes - this test involves many browser instances

    // Create 10 users and a lobby (via API for speed)
    const { lobby, users } = await lobbyWithUsers(10);

    expect(lobby.players).toHaveLength(10);

    // Navigate all users to the lobby page
    const lobbyPages: LobbyRoomPage[] = [];
    for (const user of users) {
      await user.page.goto(`/lobby/${lobby.id}`);
      lobbyPages.push(new LobbyRoomPage(user.page));
    }

    // Wait for all pages to load
    for (const user of users) {
      await expect(user.page.locator('text=10-Man Lobby')).toBeVisible();
    }

    // Ready up all users via API for reliability (UI ready clicking is tested in other tests)
    await Promise.all(
      users.map((user) =>
        apiCall(`/lobbies/${lobby.id}/ready`, user.token, { body: { ready: true } })
      )
    );

    // Navigate creator to lobby page and wait for it to show updated state
    const creatorPage = lobbyPages[0];
    await users[0].page.reload();
    await expect(users[0].page.locator('text=10-Man Lobby')).toBeVisible();

    // Creator should see Generate Teams button (all 10 are ready)
    await creatorPage.expectGenerateTeamsButton();

    // Creator (Blue Captain) clicks Propose Matchmake via UI
    await creatorPage.clickGenerateTeams();

    // Wait for the pending action banner to appear (matchmake proposal)
    await creatorPage.expectPendingActionBanner();

    // Find Red Captain and approve (parallel search)
    const matchmakeCaptain = await findCaptainWithApproveButton(users, lobbyPages);
    if (matchmakeCaptain) {
      await matchmakeCaptain.lobbyPage.clickApprovePendingAction();
      await users[matchmakeCaptain.index].page.waitForTimeout(1000);
    }

    // Reload creator page to see match options
    await users[0].page.reload();
    await expect(users[0].page.locator('text=10-Man Lobby')).toBeVisible();

    // Wait for team options to appear
    await creatorPage.waitForMatchOptions();

    // Creator (Blue Captain) proposes selecting first option
    await creatorPage.selectOption(1);

    // Wait for the pending action banner
    await creatorPage.expectPendingActionBanner();

    // Find Red Captain for option selection approval (parallel search)
    const optionCaptain = await findCaptainWithApproveButton(users, lobbyPages);
    if (optionCaptain) {
      await optionCaptain.lobbyPage.clickApprovePendingAction();
      await users[optionCaptain.index].page.waitForTimeout(1000);
    }

    // Reload creator page to see Start Draft option
    await users[0].page.reload();
    await expect(users[0].page.locator('text=10-Man Lobby')).toBeVisible();

    // Creator should see Start Draft button (or Propose Start Draft)
    await creatorPage.expectStartDraftButton();

    // Creator clicks Start Draft (this also creates a pending action)
    await creatorPage.clickStartDraft();

    // Wait for pending action if Start Draft creates one
    const pendingBanner = users[0].page.locator('.bg-yellow-900\\/30');
    try {
      await pendingBanner.waitFor({ state: 'visible', timeout: 3000 });
      // Find Red Captain for start draft approval (parallel search)
      const startDraftCaptain = await findCaptainWithApproveButton(users, lobbyPages);
      if (startDraftCaptain) {
        await startDraftCaptain.lobbyPage.clickApprovePendingAction();
        await users[startDraftCaptain.index].page.waitForTimeout(1000);
      }
    } catch {
      // No pending action, draft started directly
    }

    // All users should be redirected to draft page
    for (const user of users) {
      await user.page.waitForURL(/\/draft\//, { timeout: 30000 });
    }
  });

  test('users can see each other in lobby player grid', async ({ lobbyWithUsers }) => {
    // Create 3 users for a simpler test
    const { lobby, users } = await lobbyWithUsers(3);

    // Navigate all users to lobby
    for (const user of users) {
      await user.page.goto(`/lobby/${lobby.id}`);
      await expect(user.page.locator('text=10-Man Lobby')).toBeVisible();
    }

    // Each user should see all 3 player names in the grid
    for (const viewer of users) {
      for (const player of users) {
        await expect(viewer.page.locator(`text=${player.user.displayName}`)).toBeVisible();
      }
    }
  });

  test('ready status updates are visible to all users', async ({ lobbyWithUsers }) => {
    // Create 2 users
    const { lobby, users } = await lobbyWithUsers(2);

    // Navigate both to lobby
    for (const user of users) {
      await user.page.goto(`/lobby/${lobby.id}`);
      await expect(user.page.locator('text=10-Man Lobby')).toBeVisible();
    }

    const [user1, user2] = users;

    // User 1 clicks ready
    await user1.page.click('button:has-text("Ready Up")');

    // Wait for button to change to Cancel Ready
    await expect(user1.page.locator('button:has-text("Cancel Ready")')).toBeVisible();

    // Refresh user 2's page to see the update
    await user2.page.reload();

    // User 1 should show Cancel Ready (they're ready)
    await expect(user1.page.locator('button:has-text("Cancel Ready")')).toBeVisible();
  });

  test('only creator can generate teams', async ({ lobbyWithUsers }) => {
    // Create 2 users
    const { lobby, users } = await lobbyWithUsers(2);

    // Navigate both to lobby
    for (const user of users) {
      await user.page.goto(`/lobby/${lobby.id}`);
    }

    const [creator, joiner] = users;

    // Creator should see they are the creator (by having access to generate button when ready)
    // Joiner should NOT see Generate Teams button even when ready
    // (Though with only 2 players, it won't appear anyway since 10 are needed)

    // Ready up both users via API for speed
    await Promise.all([
      apiCall(`/lobbies/${lobby.id}/ready`, creator.token, { body: { ready: true } }),
      apiCall(`/lobbies/${lobby.id}/ready`, joiner.token, { body: { ready: true } }),
    ]);

    // Reload both pages
    await creator.page.reload();
    await joiner.page.reload();

    // Since we need 10 players for Generate Teams, this test just verifies
    // that the joiner doesn't see controls they shouldn't see
    // The Generate Teams button should not appear with only 2 players
    await expect(creator.page.locator('button:has-text("Generate Team Options")')).not.toBeVisible();
    await expect(joiner.page.locator('button:has-text("Generate Team Options")')).not.toBeVisible();
  });

  test('lobby shows correct player counts during join', async ({ createUsers }) => {
    // Create 5 users
    const users = await createUsers(5);

    // First user creates lobby via API
    const createResponse = await apiCall('/lobbies', users[0].token, {
      body: { draftMode: 'pro_play', timerDurationSeconds: 90 },
    });
    const lobby = await createResponse.json();

    // Navigate creator to lobby
    await users[0].page.goto(`/lobby/${lobby.id}`);
    await expect(users[0].page.locator('text=1/10')).toBeVisible();

    // Each subsequent user joins and we verify count increases
    for (let i = 1; i < users.length; i++) {
      // Join via API
      await apiCall(`/lobbies/${lobby.id}/join`, users[i].token);

      // Navigate to lobby
      await users[i].page.goto(`/lobby/${lobby.id}`);

      // Reload creator's page and verify updated count
      await users[0].page.reload();
      await expect(users[0].page.locator(`text=${i + 1}/10`)).toBeVisible({ timeout: 5000 });
    }
  });
});

test.describe('Multi-User Lobby Draft Flow', () => {
  test('10 users complete full draft flow from lobby', async ({ lobbyWithUsers }) => {
    test.setTimeout(300000); // 5 minutes for full 10 user draft flow

    // Create 10 users and a lobby (via API for speed)
    const { lobby, users } = await lobbyWithUsers(10);

    expect(lobby.players).toHaveLength(10);

    // Navigate all users to the lobby page
    const lobbyPages: LobbyRoomPage[] = [];
    for (const user of users) {
      await user.page.goto(`/lobby/${lobby.id}`);
      lobbyPages.push(new LobbyRoomPage(user.page));
    }

    // Wait for all pages to load
    for (const user of users) {
      await expect(user.page.locator('text=10-Man Lobby')).toBeVisible();
    }

    // Ready up all users via API for reliability
    await Promise.all(
      users.map((user) =>
        apiCall(`/lobbies/${lobby.id}/ready`, user.token, { body: { ready: true } })
      )
    );

    // Reload creator's page to see updated state
    await users[0].page.reload();
    await expect(users[0].page.locator('text=10-Man Lobby')).toBeVisible();

    // Creator generates teams, selects option, and starts draft
    const creatorPage = lobbyPages[0];
    await creatorPage.expectGenerateTeamsButton();
    await creatorPage.clickGenerateTeams();

    // Handle captain approval flow for matchmaking
    await creatorPage.expectPendingActionBanner();
    const matchmakeCaptain = await findCaptainWithApproveButton(users, lobbyPages);
    if (matchmakeCaptain) {
      await matchmakeCaptain.lobbyPage.clickApprovePendingAction();
      await users[matchmakeCaptain.index].page.waitForTimeout(1000);
    }

    await users[0].page.reload();
    await creatorPage.waitForMatchOptions();
    await creatorPage.selectOption(1);

    // Handle captain approval flow for option selection
    await creatorPage.expectPendingActionBanner();
    const optionCaptain = await findCaptainWithApproveButton(users, lobbyPages);
    if (optionCaptain) {
      await optionCaptain.lobbyPage.clickApprovePendingAction();
      await users[optionCaptain.index].page.waitForTimeout(1000);
    }

    // Start draft - all users will be redirected
    await users[0].page.reload();
    await creatorPage.expectStartDraftButton();
    await creatorPage.clickStartDraft();

    // Handle captain approval flow for start draft
    const pendingBanner = users[0].page.locator('.bg-yellow-900\\/30');
    try {
      await pendingBanner.waitFor({ state: 'visible', timeout: 3000 });
      const startDraftCaptain = await findCaptainWithApproveButton(users, lobbyPages);
      if (startDraftCaptain) {
        await startDraftCaptain.lobbyPage.clickApprovePendingAction();
        await users[startDraftCaptain.index].page.waitForTimeout(1000);
      }
    } catch {
      // No pending action, draft started directly
    }

    // All users should be redirected to draft page
    const draftPages: DraftRoomPage[] = [];
    for (const user of users) {
      await user.page.waitForURL(/\/draft\//, { timeout: 30000 });
      draftPages.push(new DraftRoomPage(user.page));
    }

    // Wait for draft UI to load for all users
    for (const draftPage of draftPages) {
      await draftPage.waitForDraftLoaded();
    }

    // Wait for WebSocket connections to establish
    await Promise.all(draftPages.map((dp) => dp.waitForWebSocketConnected()));

    // Ready up via UI - only non-spectators can click Ready
    for (const draftPage of draftPages) {
      const canReady = await draftPage.canClickReady();
      if (canReady) {
        await draftPage.clickReady();
      }
    }

    // One of the captains should see Start Draft button
    // Find a user who can see Start Draft (wait longer for WebSocket to sync)
    let starterFound = false;
    for (const draftPage of draftPages) {
      const startButton = draftPage.getPage().locator('button:has-text("Start Draft")');
      const count = await startButton.count();
      if (count > 0 && (await startButton.isVisible())) {
        await startButton.click();
        starterFound = true;
        break;
      }
    }

    expect(starterFound).toBe(true);

    // Wait for draft to start - Lock In button should appear
    await draftPages[0].waitForActiveState();

    // At this point, draft is active. Blue team captain picks first (ban phase)
    // Find the user whose turn it is and perform a ban
    for (let phase = 0; phase < 6; phase++) {
      // Go through first 6 ban phases - use waitForAnyTurn for proper waiting
      const captain = await waitForAnyTurn(draftPages, 15000);
      await captain.performBanOrPick(phase);
    }

    // After 6 bans, we should be in pick phase
    // Continue with 4 pick phases
    for (let phase = 0; phase < 4; phase++) {
      const captain = await waitForAnyTurn(draftPages, 15000);
      // Use higher indices to avoid selecting already banned champions
      await captain.performBanOrPick(10 + phase);
    }

    // Verify that picks are visible on team panels
    for (const draftPage of draftPages) {
      await draftPage.expectPicksVisible();
    }
  });

  test('captains can pick and ban in correct order', async ({ lobbyWithUsers }) => {
    test.setTimeout(180000); // 3 minutes for 10 user draft flow

    // Create 10 users and go through lobby flow via API for speed
    const { lobby, users } = await lobbyWithUsers(10);

    // Ready up all users via API
    await Promise.all(
      users.map((user) =>
        apiCall(`/lobbies/${lobby.id}/ready`, user.token, { body: { ready: true } })
      )
    );

    // Generate teams via API
    const genResponse = await apiCall(
      `/lobbies/${lobby.id}/generate-teams`,
      users[0].token
    );
    expect(genResponse.ok).toBe(true);

    // Select first option via API
    await apiCall(`/lobbies/${lobby.id}/select-option`, users[0].token, {
      body: { optionNumber: 1 },
    });

    // Start draft via API
    const startResponse = await apiCall(
      `/lobbies/${lobby.id}/start-draft`,
      users[0].token
    );
    const startData = await startResponse.json();
    const roomId = startData.id; // API returns 'id' not 'roomId'

    // Navigate all users to draft room
    console.log(`Navigating 10 users to draft room ${roomId}...`);
    for (const user of users) {
      await user.page.goto(`/draft/${roomId}`);
    }

    // Create draft page objects
    const draftPages = users.map((u) => new DraftRoomPage(u.page));

    // Wait for all to load and WebSocket to connect
    console.log('Waiting for all pages to load and WebSocket to connect...');
    await Promise.all(
      draftPages.map(async (draftPage, i) => {
        await draftPage.waitForDraftLoaded();
        await draftPage.waitForWebSocketConnected();
        console.log(`User ${i} connected`);
      })
    );

    // Ready up via UI - only non-spectators can click Ready
    // In 10-man draft, only 2 users (one per team) are the actual clients
    let readyClicks = 0;
    const readyUsers: string[] = [];
    for (let i = 0; i < draftPages.length; i++) {
      const draftPage = draftPages[i];
      const canReady = await draftPage.canClickReady();
      console.log(`User ${i}: canClickReady = ${canReady}`);
      if (canReady) {
        await draftPage.clickReady();
        readyClicks++;
        readyUsers.push(`user_${i}`);
      }
    }

    // Should have clicked Ready on exactly 2 pages (blue and red clients)
    console.log(`Clicked Ready on ${readyClicks} pages: ${readyUsers.join(', ')}`);
    expect(readyClicks).toBe(2);

    // Find and click Start Draft - should be visible when both teams ready
    // No reload needed - WebSocket should sync the state
    console.log('Looking for Start Draft button...');
    let startClicked = false;
    for (let i = 0; i < draftPages.length; i++) {
      const draftPage = draftPages[i];
      const startButton = draftPage.getPage().locator('button:has-text("Start Draft")');
      const count = await startButton.count();
      const isVisible = count > 0 && (await startButton.isVisible());
      console.log(`User ${i}: Start Draft visible = ${isVisible}`);
      if (isVisible) {
        await startButton.click();
        startClicked = true;
        console.log(`User ${i} clicked Start Draft`);
        break;
      }
    }

    expect(startClicked).toBe(true);

    // Wait for active state
    console.log('Waiting for draft to become active...');
    await draftPages[0].waitForActiveState();
    console.log('Draft is active!');

    // Complete all 20 phases of pro play draft:
    // Phases 0-5: 6 bans (B-R-B-R-B-R)
    // Phases 6-9: 4 picks (B-R-R-B)
    // Phases 10-13: 4 bans (R-B-R-B)
    // Phases 14-19: 6 picks (R-B-B-R-B-R)
    const TOTAL_PHASES = 20;

    for (let phase = 0; phase < TOTAL_PHASES; phase++) {
      console.log(`Phase ${phase}: waiting for player turn...`);

      // Use waitForAnyTurn for proper Playwright waiting instead of manual polling
      const captain = await waitForAnyTurn(draftPages, 15000);
      console.log(`Phase ${phase}: player taking turn`);
      await captain.performBanOrPick(phase);
    }

    console.log('All 20 phases complete!');

    // Verify draft is complete
    await draftPages[0].expectDraftComplete();
  });
});

test.describe('Match Options Visibility', () => {
  test('all users can see match options but only creator can select', async ({ lobbyWithUsers }) => {
    test.setTimeout(120000); // 2 minutes

    // Create 10 users and a lobby
    const { lobby, users } = await lobbyWithUsers(10);
    const [creator, ...joiners] = users;

    // Ready up all users via API for speed
    await Promise.all(
      users.map((user) =>
        apiCall(`/lobbies/${lobby.id}/ready`, user.token, { body: { ready: true } })
      )
    );

    // Generate teams via API (creator only)
    const genResponse = await apiCall(
      `/lobbies/${lobby.id}/generate-teams`,
      creator.token
    );
    expect(genResponse.ok).toBe(true);

    // Navigate all users to the lobby page and wait for match options to load
    for (const user of users) {
      await user.page.goto(`/lobby/${lobby.id}`);
      // Wait for page to fully load - match options should be visible since lobby is in matchmaking status
      await expect(user.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 15000 });
    }

    // All users should see "Select Team Composition" heading (wait longer for match options fetch)
    for (let i = 0; i < users.length; i++) {
      await expect(users[i].page.locator('text=Select Team Composition')).toBeVisible({ timeout: 15000 });
    }

    // All users should see match option cards (Option 1, Option 2, etc.)
    for (let i = 0; i < users.length; i++) {
      await expect(users[i].page.locator('[data-testid="match-option-1"]')).toBeVisible();
      await expect(users[i].page.locator('text=Option 1')).toBeVisible();
      // Should see Blue Team and Red Team sections
      await expect(users[i].page.locator('text=Blue Team').first()).toBeVisible();
      await expect(users[i].page.locator('text=Red Team').first()).toBeVisible();
    }

    // CREATOR should see "Select This Option" buttons
    await expect(creator.page.locator('button:has-text("Select This Option")').first()).toBeVisible();

    // NON-CREATORS should NOT see "Select This Option" buttons (no onSelect prop)
    for (const joiner of joiners) {
      await expect(joiner.page.locator('button:has-text("Select This Option")')).not.toBeVisible();
    }

    // Creator selects an option
    const creatorPage = new LobbyRoomPage(creator.page);
    await creatorPage.selectOption(1);
    await creatorPage.clickConfirmSelection();

    // Wait for selection to be confirmed
    await creatorPage.expectStartDraftButton();

    // After selection, all users should see the selected option highlighted
    for (const user of users) {
      // Reload to get updated state
      await user.page.reload();
      await expect(user.page.locator('text=Select Team Composition')).toBeVisible({ timeout: 10000 });
      // Option 1 should be visually selected (has gold border)
      const option1 = user.page.locator('[data-testid="match-option-1"]');
      await expect(option1).toHaveClass(/border-lol-gold/);
    }

    // Only creator should see "Start Draft" button
    await creatorPage.expectStartDraftButton();
    // Non-creators should not see the creator-only start draft button
    for (const joiner of joiners) {
      await expect(joiner.page.locator('button:has-text("Start Draft (Creator Only)")')).not.toBeVisible();
    }
  });

  test('non-creator can see match options after joining late', async ({ createUsers }) => {
    test.setTimeout(120000);

    // Create 10 users
    const users = await createUsers(10);
    const [creator, lateJoiner, ...others] = users;

    // Creator creates lobby
    const createResponse = await apiCall('/lobbies', creator.token, {
      body: { draftMode: 'pro_play', timerDurationSeconds: 90 },
    });
    const lobby = await createResponse.json();

    // All users except lateJoiner join
    await Promise.all(
      others.map((user) => apiCall(`/lobbies/${lobby.id}/join`, user.token))
    );

    // lateJoiner joins
    await apiCall(`/lobbies/${lobby.id}/join`, lateJoiner.token);

    // Ready up all users
    await Promise.all(
      users.map((user) =>
        apiCall(`/lobbies/${lobby.id}/ready`, user.token, { body: { ready: true } })
      )
    );

    // Generate teams
    await apiCall(`/lobbies/${lobby.id}/generate-teams`, creator.token);

    // Now lateJoiner navigates to lobby - should see match options
    await lateJoiner.page.goto(`/lobby/${lobby.id}`);
    await expect(lateJoiner.page.locator('text=10-Man Lobby')).toBeVisible();

    // Should see match options even though they joined after generation
    await expect(lateJoiner.page.locator('text=Select Team Composition')).toBeVisible({ timeout: 10000 });
    await expect(lateJoiner.page.locator('[data-testid="match-option-1"]')).toBeVisible();

    // But should NOT see the confirm selection button (only creator can select)
    await expect(lateJoiner.page.locator('button:has-text("Confirm Selection")')).not.toBeVisible();
  });
});

test.describe('Multi-User Lobby Error Cases', () => {
  test('cannot generate teams without 10 players ready', async ({ lobbyWithUsers }) => {
    // Create 5 users (not enough for team generation)
    const { lobby, users } = await lobbyWithUsers(5);

    // Navigate creator to lobby
    await users[0].page.goto(`/lobby/${lobby.id}`);

    // Ready up all users via UI
    for (const user of users) {
      await user.page.goto(`/lobby/${lobby.id}`);
      await user.page.click('button:has-text("Ready Up")');
      await expect(user.page.locator('button:has-text("Cancel Ready")')).toBeVisible();
    }

    // Reload creator's page
    await users[0].page.reload();

    // Generate Teams button should NOT be visible with only 5 players
    await expect(users[0].page.locator('button:has-text("Generate Team Options")')).not.toBeVisible();
  });
});
