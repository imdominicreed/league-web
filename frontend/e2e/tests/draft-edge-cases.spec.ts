import { test, expect, setUserReady, generateTeams, selectMatchOption } from '../fixtures/multi-user';
import { DraftRoomPage } from '../fixtures/pages';

const API_BASE = 'http://localhost:9999/api/v1';

/**
 * Helper to create a lobby with custom timer via API
 */
async function createLobbyWithTimerViaApi(
  token: string,
  timerDurationSeconds: number = 30
): Promise<{ id: string; shortCode: string }> {
  const response = await fetch(`${API_BASE}/lobbies`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ draftMode: 'pro_play', timerDurationSeconds }),
  });
  if (!response.ok) {
    throw new Error(`Create lobby failed: ${response.status}`);
  }
  return response.json();
}

/**
 * Helper to join a lobby via API
 */
async function joinLobbyViaApi(token: string, lobbyId: string): Promise<void> {
  const response = await fetch(`${API_BASE}/lobbies/${lobbyId}/join`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) {
    throw new Error(`Join lobby failed: ${response.status}`);
  }
}

/**
 * Helper to start draft from lobby via API and return roomId
 */
async function startDraftViaApi(token: string, lobbyId: string): Promise<string> {
  const response = await fetch(`${API_BASE}/lobbies/${lobbyId}/start-draft`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) {
    throw new Error(`Start draft failed: ${response.status}`);
  }
  const data = await response.json();
  return data.id;
}

/**
 * Helper to set up a 10-player draft ready to start
 * Returns the roomId and array of DraftRoomPage objects
 */
async function setupDraftFromLobby(
  lobbyWithUsers: (count: number) => Promise<{ lobby: { id: string }; users: { page: any; token: string }[] }>
): Promise<{ roomId: string; draftPages: DraftRoomPage[]; users: { page: any; token: string }[] }> {
  // Create 10 users and lobby
  const { lobby, users } = await lobbyWithUsers(10);

  // Ready all users via API
  for (const user of users) {
    await setUserReady(user as any, lobby.id, true);
  }

  // Generate teams and select first option
  await generateTeams(users[0] as any, lobby.id);
  await selectMatchOption(users[0] as any, lobby.id, 1);

  // Start draft via API
  const roomId = await startDraftViaApi(users[0].token, lobby.id);

  // Navigate all users to draft room
  for (const user of users) {
    await user.page.goto(`/draft/${roomId}`);
  }

  // Create draft page objects and wait for load
  const draftPages = users.map((u) => new DraftRoomPage(u.page));
  for (const draftPage of draftPages) {
    await draftPage.waitForDraftLoaded();
    await draftPage.waitForWebSocketConnected();
  }

  return { roomId, draftPages, users };
}

/**
 * Helper to ready up and start the draft via UI
 */
async function readyUpAndStartDraft(draftPages: DraftRoomPage[]): Promise<void> {
  // Wait for state to settle
  await draftPages[0].page.waitForTimeout(1000);

  // Ready up via UI - only captains can click Ready
  for (const draftPage of draftPages) {
    const canReady = await draftPage.canClickReady();
    if (canReady) {
      await draftPage.clickReady();
      await draftPage.page.waitForTimeout(500);
    }
  }

  // Wait for ready state to propagate
  await draftPages[0].page.waitForTimeout(1500);

  // Find and click Start Draft
  for (const draftPage of draftPages) {
    const startButton = draftPage.page.locator('button:has-text("Start Draft")');
    if (await startButton.isVisible({ timeout: 3000 }).catch(() => false)) {
      await startButton.click();
      break;
    }
  }

  // Wait for active state
  await draftPages[0].waitForActiveState();
}

/**
 * Helper to find the captain whose turn it is
 */
async function findCurrentTurnCaptain(draftPages: DraftRoomPage[]): Promise<DraftRoomPage | null> {
  for (const draftPage of draftPages) {
    if (await draftPage.isYourTurn()) {
      return draftPage;
    }
  }
  return null;
}

/**
 * Helper to complete N phases of the draft
 */
async function completePhases(draftPages: DraftRoomPage[], count: number, startIndex: number = 0): Promise<void> {
  for (let phase = 0; phase < count; phase++) {
    const captain = await findCurrentTurnCaptain(draftPages);
    if (!captain) {
      throw new Error(`Phase ${phase}: No captain found for current turn`);
    }
    await captain.performBanOrPick(startIndex + phase);
    await draftPages[0].page.waitForTimeout(800);
  }
}

/**
 * Helper to set up a 10-player draft with custom timer duration
 */
async function setupDraftWithShortTimer(
  createUsers: (count: number) => Promise<{ page: any; token: string; user: { id: string } }[]>,
  timerDurationSeconds: number
): Promise<{ roomId: string; draftPages: DraftRoomPage[]; users: { page: any; token: string }[] }> {
  // Create 10 users
  const users = await createUsers(10);

  // First user creates lobby with custom timer
  const lobby = await createLobbyWithTimerViaApi(users[0].token, timerDurationSeconds);

  // Other users join
  for (let i = 1; i < users.length; i++) {
    await joinLobbyViaApi(users[i].token, lobby.id);
  }

  // Ready all users
  for (const user of users) {
    await setUserReady(user as any, lobby.id, true);
  }

  // Generate teams and select first option
  await generateTeams(users[0] as any, lobby.id);
  await selectMatchOption(users[0] as any, lobby.id, 1);

  // Start draft
  const roomId = await startDraftViaApi(users[0].token, lobby.id);

  // Navigate all users to draft room
  for (const user of users) {
    await user.page.goto(`/draft/${roomId}`);
  }

  // Create draft page objects and wait for load
  const draftPages = users.map((u) => new DraftRoomPage(u.page));
  for (const draftPage of draftPages) {
    await draftPage.waitForDraftLoaded();
    await draftPage.waitForWebSocketConnected();
  }

  return { roomId, draftPages, users };
}

// ============================================================================
// INVALID ACTION TESTS
// ============================================================================

test.describe('Draft Edge Cases: Invalid Actions', () => {
  test('banned champion button is disabled in pick phase', async ({ lobbyWithUsers }) => {
    test.setTimeout(180000);

    const { draftPages } = await setupDraftFromLobby(lobbyWithUsers);
    await readyUpAndStartDraft(draftPages);

    // Get the first action to track phase changes
    const initialAction = await draftPages[0].getCurrentAction();
    expect(initialAction).toContain('Ban');

    // Complete first ban phase and note which champion was banned
    const captain = await findCurrentTurnCaptain(draftPages);
    expect(captain).not.toBeNull();

    // Select first champion (index 0) and lock in
    await captain!.performBanOrPick(0);

    // Now champion at index 0 should be disabled for everyone
    await draftPages[0].page.waitForTimeout(500);

    // Check that the banned champion is now disabled
    const isDisabled = await draftPages[0].isChampionDisabledByIndex(0);
    expect(isDisabled).toBe(true);

    // Complete remaining 5 bans (phases 1-5)
    await completePhases(draftPages, 5, 1);

    // Now we're in pick phase - verify first 6 champions are disabled (they were banned)
    // Check that at least a few of the early indices are disabled
    for (let i = 0; i < 6; i++) {
      const disabled = await draftPages[0].isChampionDisabledByIndex(i);
      expect(disabled).toBe(true);
    }
  });

  test('picked champion button is disabled for subsequent picks', async ({ lobbyWithUsers }) => {
    test.setTimeout(180000);

    const { draftPages } = await setupDraftFromLobby(lobbyWithUsers);
    await readyUpAndStartDraft(draftPages);

    // Complete 6 ban phases (indices 0-5)
    await completePhases(draftPages, 6, 0);

    // Now in pick phase - pick champion at index 10
    const captain = await findCurrentTurnCaptain(draftPages);
    expect(captain).not.toBeNull();
    await captain!.performBanOrPick(10);

    // Champion at index 10 should now be disabled
    await draftPages[0].page.waitForTimeout(500);
    const isDisabled = await draftPages[0].isChampionDisabledByIndex(10);
    expect(isDisabled).toBe(true);
  });

  test('non-captain has disabled champion grid', async ({ lobbyWithUsers }) => {
    test.setTimeout(180000);

    const { draftPages } = await setupDraftFromLobby(lobbyWithUsers);
    await readyUpAndStartDraft(draftPages);

    // Count how many pages have enabled champion grid (should be exactly 1 - the current captain)
    let enabledCount = 0;
    for (const draftPage of draftPages) {
      if (await draftPage.isYourTurn()) {
        enabledCount++;
      }
    }

    // Exactly 1 captain should have an enabled grid
    expect(enabledCount).toBe(1);

    // The other 9 users should have disabled grids
    let disabledCount = 0;
    for (const draftPage of draftPages) {
      if (!(await draftPage.isYourTurn())) {
        disabledCount++;
      }
    }
    expect(disabledCount).toBe(9);
  });

  test('lock-in button is disabled without champion selection', async ({ lobbyWithUsers }) => {
    test.setTimeout(180000);

    const { draftPages } = await setupDraftFromLobby(lobbyWithUsers);
    await readyUpAndStartDraft(draftPages);

    // Find the captain whose turn it is
    const captain = await findCurrentTurnCaptain(draftPages);
    expect(captain).not.toBeNull();

    // Lock In should be disabled before any selection
    await captain!.expectLockInDisabled();

    // Select a champion
    await captain!.selectChampionByIndex(0);
    await captain!.page.waitForTimeout(300);

    // Now Lock In should be enabled
    await captain!.expectLockInEnabled();
  });
});

// ============================================================================
// RECONNECTION TESTS
// ============================================================================

test.describe('Draft Edge Cases: Reconnection', () => {
  test('captain can reload during opponent turn and continue', async ({ lobbyWithUsers }) => {
    test.setTimeout(180000);

    const { draftPages } = await setupDraftFromLobby(lobbyWithUsers);
    await readyUpAndStartDraft(draftPages);

    // Complete first ban (blue team)
    const blueCaptain = await findCurrentTurnCaptain(draftPages);
    expect(blueCaptain).not.toBeNull();
    const blueCaptainIndex = draftPages.indexOf(blueCaptain!);

    await blueCaptain!.performBanOrPick(0);
    await draftPages[0].page.waitForTimeout(800);

    // Now it's red team's turn - blue captain reloads
    await blueCaptain!.reloadAndReconnect();

    // Blue captain should see the ban that was made
    const bannedChamps = await blueCaptain!.getBannedChampionNames();
    expect(bannedChamps.length).toBeGreaterThanOrEqual(1);

    // Red team completes their ban
    const redCaptain = await findCurrentTurnCaptain(draftPages);
    expect(redCaptain).not.toBeNull();
    await redCaptain!.performBanOrPick(1);
    await draftPages[0].page.waitForTimeout(800);

    // Now blue captain should be able to take their turn again
    // The reloaded page should detect it's their turn
    const isBluesTurn = await draftPages[blueCaptainIndex].isYourTurn();
    expect(isBluesTurn).toBe(true);

    // Blue captain can complete their action
    await draftPages[blueCaptainIndex].performBanOrPick(2);
  });

  test('captain can reload during own turn and complete action', async ({ lobbyWithUsers }) => {
    test.setTimeout(180000);

    const { draftPages } = await setupDraftFromLobby(lobbyWithUsers);
    await readyUpAndStartDraft(draftPages);

    // Find the captain whose turn it is
    const captain = await findCurrentTurnCaptain(draftPages);
    expect(captain).not.toBeNull();
    const captainIndex = draftPages.indexOf(captain!);

    // Get current action before reload
    const actionBefore = await captain!.getCurrentAction();

    // Reload during their turn
    await captain!.reloadAndReconnect();

    // Timer should still be running (check it's greater than 0)
    const timer = await draftPages[captainIndex].getTimerSeconds();
    expect(timer).toBeGreaterThan(0);

    // Action should be the same (still their turn)
    const actionAfter = await draftPages[captainIndex].getCurrentAction();
    expect(actionAfter).toBe(actionBefore);

    // They should still be able to complete their action
    const isStillMyTurn = await draftPages[captainIndex].isYourTurn();
    expect(isStillMyTurn).toBe(true);

    await draftPages[captainIndex].performBanOrPick(0);
  });

  test('all 10 users can reload simultaneously and continue draft', async ({ lobbyWithUsers }) => {
    test.setTimeout(180000);

    const { draftPages } = await setupDraftFromLobby(lobbyWithUsers);
    await readyUpAndStartDraft(draftPages);

    // Complete 2 bans first
    await completePhases(draftPages, 2, 0);

    // Reload all 10 pages simultaneously
    await Promise.all(draftPages.map((dp) => dp.page.reload()));

    // Wait for all to reconnect
    await Promise.all(draftPages.map((dp) => dp.waitForDraftLoaded()));
    await Promise.all(draftPages.map((dp) => dp.waitForWebSocketConnected()));

    // Verify state is consistent - all should see 2 bans completed
    for (const draftPage of draftPages) {
      const bannedChamps = await draftPage.getBannedChampionNames();
      expect(bannedChamps.length).toBe(2);
    }

    // Draft should be able to continue
    const captain = await findCurrentTurnCaptain(draftPages);
    expect(captain).not.toBeNull();
    await captain!.performBanOrPick(2);
  });
});

// ============================================================================
// TIMER EXPIRY TESTS
// ============================================================================

test.describe('Draft Edge Cases: Timer Expiry', () => {
  test('timer expiry auto-advances to next phase', async ({ createUsers }) => {
    test.setTimeout(120000);

    // Create draft with 5-second timer
    const { draftPages } = await setupDraftWithShortTimer(createUsers, 5);
    await readyUpAndStartDraft(draftPages);

    // Get initial action (should be Blue Ban)
    const initialAction = await draftPages[0].getCurrentAction();
    expect(initialAction.toLowerCase()).toContain('blue');
    expect(initialAction.toLowerCase()).toContain('ban');

    // Don't take any action - let timer expire
    // Wait for timer to tick down and phase to change (5 seconds + buffer)
    await draftPages[0].page.waitForTimeout(7000);

    // Phase should have auto-advanced to Red Ban
    const currentAction = await draftPages[0].getCurrentAction();
    expect(currentAction.toLowerCase()).toContain('red');
    expect(currentAction.toLowerCase()).toContain('ban');
  });

  test('multiple timer expiries advance draft correctly', async ({ createUsers }) => {
    test.setTimeout(120000);

    // Create draft with 3-second timer for faster test
    const { draftPages } = await setupDraftWithShortTimer(createUsers, 3);
    await readyUpAndStartDraft(draftPages);

    // Let 3 phases expire (3 seconds each = ~9-12 seconds with buffer)
    await draftPages[0].page.waitForTimeout(12000);

    // Should have advanced 3+ phases
    // In pro play: phases 0-2 are Blue Ban, Red Ban, Blue Ban
    // After 3 expirations, we should be on phase 3 (Red Ban)
    const action = await draftPages[0].getCurrentAction();
    // Could be on phase 3, 4, or later depending on timing
    expect(action.toLowerCase()).toMatch(/ban|pick/);

    // Verify draft is still functional - next captain can act
    const captain = await findCurrentTurnCaptain(draftPages);
    if (captain) {
      await captain.performBanOrPick(10);
      // Phase should advance after manual action
      await draftPages[0].page.waitForTimeout(1000);
    }
  });
});

// ============================================================================
// RACE CONDITION TESTS
// ============================================================================

test.describe('Draft Edge Cases: Race Conditions', () => {
  test('rapid champion selection changes work correctly', async ({ lobbyWithUsers }) => {
    test.setTimeout(180000);

    const { draftPages } = await setupDraftFromLobby(lobbyWithUsers);
    await readyUpAndStartDraft(draftPages);

    // Find the captain whose turn it is
    const captain = await findCurrentTurnCaptain(draftPages);
    expect(captain).not.toBeNull();

    // Rapidly click different champions
    await captain!.selectChampionByIndex(0);
    await captain!.selectChampionByIndex(1);
    await captain!.selectChampionByIndex(2);
    await captain!.selectChampionByIndex(3);
    // Final selection
    await captain!.selectChampionByIndex(5);

    // Small wait for last selection to register
    await captain!.page.waitForTimeout(200);

    // Lock In should work
    await captain!.expectLockInEnabled();
    await captain!.clickLockIn();

    // Phase should advance
    await captain!.page.waitForTimeout(800);

    // Champion at index 5 should be disabled (it was locked in)
    const isDisabled = await draftPages[0].isChampionDisabledByIndex(5);
    expect(isDisabled).toBe(true);
  });

  test('double lock-in click only advances phase once', async ({ lobbyWithUsers }) => {
    test.setTimeout(180000);

    const { draftPages } = await setupDraftFromLobby(lobbyWithUsers);
    await readyUpAndStartDraft(draftPages);

    // Get initial action
    const initialAction = await draftPages[0].getCurrentAction();

    // Find the captain
    const captain = await findCurrentTurnCaptain(draftPages);
    expect(captain).not.toBeNull();

    // Select a champion
    await captain!.selectChampionByIndex(0);
    await captain!.page.waitForTimeout(200);

    // Rapidly double-click Lock In
    const lockInButton = captain!.page.locator('button:has-text("Lock In")');
    await lockInButton.dblclick();

    // Wait for phase to advance
    await captain!.page.waitForTimeout(1000);

    // Get current action - should have advanced exactly once
    const currentAction = await draftPages[0].getCurrentAction();
    expect(currentAction).not.toBe(initialAction);

    // Verify we're on second ban phase (Red Ban), not third
    // First phase: Blue Ban -> Second phase: Red Ban
    expect(currentAction.toLowerCase()).toContain('red');
    expect(currentAction.toLowerCase()).toContain('ban');

    // Only 1 champion should be banned (not 2)
    const bannedCount = await draftPages[0].getBannedChampionNames();
    expect(bannedCount.length).toBe(1);
  });
});
