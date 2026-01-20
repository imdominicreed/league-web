import { test, expect, Page } from '@playwright/test';
import { createTestUser, setAuthToken, joinLobby, initializeRoleProfiles, generateTeams, selectMatchOption, TestUser } from '../helpers/test-utils';

const API_BASE = 'http://localhost:9999/api/v1';

/**
 * Propose a swap between two players
 */
async function proposeSwap(page: Page, token: string, lobbyId: string, player1Id: string, player2Id: string, swapType: 'players' | 'roles') {
  const res = await page.request.post(`${API_BASE}/lobbies/${lobbyId}/swap`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { player1Id, player2Id, swapType },
  });
  expect(res.ok()).toBeTruthy();
  return await res.json();
}

/**
 * Get pending action for a lobby
 */
async function getPendingAction(page: Page, token: string, lobbyId: string) {
  const res = await page.request.get(`${API_BASE}/lobbies/${lobbyId}/pending-action`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  return res.ok() ? await res.json() : null;
}

/**
 * Approve a pending action
 */
async function approvePendingAction(page: Page, token: string, lobbyId: string, actionId: string) {
  const res = await page.request.post(`${API_BASE}/lobbies/${lobbyId}/pending-action/${actionId}/approve`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(res.ok()).toBeTruthy();
  return await res.json();
}

/**
 * Get lobby data via API
 */
async function getLobby(page: Page, token: string, lobbyId: string) {
  const res = await page.request.get(`${API_BASE}/lobbies/${lobbyId}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(res.ok()).toBeTruthy();
  return await res.json();
}

test.describe('BUG-007: Lobby UI Does Not Update in Real-Time After Swap Approval', () => {
  test('should update player positions after cross-team swap is approved', async ({ page, context }) => {
    // Create 6 users (3 per team)
    const users: TestUser[] = [];
    for (let i = 1; i <= 6; i++) {
      const user = await createTestUser(page, `bug007_user${i}`);
      await initializeRoleProfiles(page, user.token);
      users.push(user);
    }

    // User1 creates lobby (blue captain)
    const createRes = await page.request.post(`${API_BASE}/lobbies`, {
      headers: { Authorization: `Bearer ${users[0].token}` },
      data: {
        draftMode: 'pro_play',
        timerDurationSeconds: 30,
      },
    });
    expect(createRes.ok()).toBeTruthy();
    const lobby = await createRes.json();

    // Other users join
    for (let i = 1; i < users.length; i++) {
      await joinLobby(page, users[i].token, lobby.id);
    }

    // Get lobby state to identify players on each team
    let lobbyData = await getLobby(page, users[0].token, lobby.id);
    const bluePlayers = lobbyData.players.filter((p: any) => p.team === 'blue');
    const redPlayers = lobbyData.players.filter((p: any) => p.team === 'red');

    // Identify captains (first player on each team)
    const blueCaptain = bluePlayers.find((p: any) => p.isCaptain);
    const redCaptain = redPlayers.find((p: any) => p.isCaptain);

    // Get a non-captain player from each team
    const blueNonCaptain = bluePlayers.find((p: any) => !p.isCaptain);
    const redNonCaptain = redPlayers.find((p: any) => !p.isCaptain);

    if (!blueCaptain || !redCaptain || !blueNonCaptain || !redNonCaptain) {
      console.log('Blue players:', bluePlayers);
      console.log('Red players:', redPlayers);
      throw new Error('Could not identify captains and non-captains on each team');
    }

    // Get tokens for the captains
    const blueCaptainUser = users.find(u => u.id === blueCaptain.userId)!;
    const redCaptainUser = users.find(u => u.id === redCaptain.userId)!;

    // Open lobby page for blue captain (who will propose the swap)
    await setAuthToken(page, blueCaptainUser);
    await page.goto(`/lobby/${lobby.id}`);
    await expect(page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Record the initial positions
    const getPlayerTeam = async (userId: string) => {
      const playerElement = page.locator(`[data-testid="lobby-player-${userId}"]`);
      const blueColumn = page.locator('[data-testid="team-column-blue"]');
      const redColumn = page.locator('[data-testid="team-column-red"]');

      const inBlue = await blueColumn.locator(`[data-testid="lobby-player-${userId}"]`).count() > 0;
      const inRed = await redColumn.locator(`[data-testid="lobby-player-${userId}"]`).count() > 0;

      if (inBlue) return 'blue';
      if (inRed) return 'red';
      return null;
    };

    // Verify initial positions
    const blueNonCaptainInitialTeam = await getPlayerTeam(blueNonCaptain.userId);
    const redNonCaptainInitialTeam = await getPlayerTeam(redNonCaptain.userId);

    console.log(`Initial: BlueNonCaptain(${blueNonCaptain.displayName}) on ${blueNonCaptainInitialTeam}, RedNonCaptain(${redNonCaptain.displayName}) on ${redNonCaptainInitialTeam}`);

    expect(blueNonCaptainInitialTeam).toBe('blue');
    expect(redNonCaptainInitialTeam).toBe('red');

    // Blue captain proposes a swap between blueNonCaptain and redNonCaptain
    const pendingAction = await proposeSwap(
      page,
      blueCaptainUser.token,
      lobby.id,
      blueNonCaptain.userId,
      redNonCaptain.userId,
      'players'
    );

    // Wait for pending action banner to appear (via WebSocket update)
    await page.waitForTimeout(2000); // Give WebSocket time to propagate
    await expect(page.locator('[data-testid="pending-action-banner"]')).toBeVisible({ timeout: 10000 });

    // Red captain approves the swap (via API from another session)
    await approvePendingAction(page, redCaptainUser.token, lobby.id, pendingAction.id);

    // Wait for UI to update (pending action banner should disappear)
    await expect(page.locator('[data-testid="pending-action-banner"]')).not.toBeVisible({ timeout: 10000 });

    // Wait a moment for WebSocket update to propagate
    await page.waitForTimeout(2000);

    // Verify players are now in swapped positions WITHOUT page refresh
    const blueNonCaptainFinalTeam = await getPlayerTeam(blueNonCaptain.userId);
    const redNonCaptainFinalTeam = await getPlayerTeam(redNonCaptain.userId);

    console.log(`After swap: BlueNonCaptain(${blueNonCaptain.displayName}) on ${blueNonCaptainFinalTeam}, RedNonCaptain(${redNonCaptain.displayName}) on ${redNonCaptainFinalTeam}`);

    // The key assertion: players should have swapped teams WITHOUT page refresh
    expect(blueNonCaptainFinalTeam).toBe('red');
    expect(redNonCaptainFinalTeam).toBe('blue');
  });

  test('should update UI via WebSocket state sync after swap', async ({ page }) => {
    // Create 4 users (2 per team minimum)
    const users: TestUser[] = [];
    for (let i = 1; i <= 4; i++) {
      const user = await createTestUser(page, `bug007_ws_user${i}`);
      await initializeRoleProfiles(page, user.token);
      users.push(user);
    }

    // Create lobby
    const createRes = await page.request.post(`${API_BASE}/lobbies`, {
      headers: { Authorization: `Bearer ${users[0].token}` },
      data: {
        draftMode: 'pro_play',
        timerDurationSeconds: 30,
      },
    });
    const lobby = await createRes.json();

    // Join users
    for (let i = 1; i < users.length; i++) {
      await joinLobby(page, users[i].token, lobby.id);
    }

    // Get player info
    let lobbyData = await getLobby(page, users[0].token, lobby.id);
    const blueCaptain = lobbyData.players.find((p: any) => p.team === 'blue' && p.isCaptain);
    const redCaptain = lobbyData.players.find((p: any) => p.team === 'red' && p.isCaptain);

    if (!blueCaptain || !redCaptain) {
      throw new Error('Could not identify captains');
    }

    const blueCaptainUser = users.find(u => u.id === blueCaptain.userId)!;
    const redCaptainUser = users.find(u => u.id === redCaptain.userId)!;

    // Navigate to lobby as blue captain
    await setAuthToken(page, blueCaptainUser);
    await page.goto(`/lobby/${lobby.id}`);
    await expect(page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Propose captain swap (swap the captains themselves)
    const pendingAction = await proposeSwap(
      page,
      blueCaptainUser.token,
      lobby.id,
      blueCaptain.userId,
      redCaptain.userId,
      'players'
    );

    // Wait for pending action to appear (via WebSocket update)
    await page.waitForTimeout(2000); // Give WebSocket time to propagate
    await expect(page.locator('[data-testid="pending-action-banner"]')).toBeVisible({ timeout: 10000 });

    // Approve via red captain
    await approvePendingAction(page, redCaptainUser.token, lobby.id, pendingAction.id);

    // Wait for pending action banner to disappear
    await expect(page.locator('[data-testid="pending-action-banner"]')).not.toBeVisible({ timeout: 10000 });

    // Give WebSocket time to update
    await page.waitForTimeout(2000);

    // Verify via API that swap happened
    const finalLobbyData = await getLobby(page, blueCaptainUser.token, lobby.id);
    const blueCaptainAfter = finalLobbyData.players.find((p: any) => p.userId === blueCaptain.userId);
    const redCaptainAfter = finalLobbyData.players.find((p: any) => p.userId === redCaptain.userId);

    console.log('After swap - original blue captain team:', blueCaptainAfter?.team);
    console.log('After swap - original red captain team:', redCaptainAfter?.team);

    // Verify API shows correct swap
    expect(blueCaptainAfter?.team).toBe('red');
    expect(redCaptainAfter?.team).toBe('blue');

    // Now check UI reflects this
    const blueColumn = page.locator('[data-testid="team-column-blue"]');
    const redColumn = page.locator('[data-testid="team-column-red"]');

    // Original blue captain should now be in red column
    const blueCaptainInRed = await redColumn.locator(`[data-testid="lobby-player-${blueCaptain.userId}"]`).count() > 0;
    // Original red captain should now be in blue column
    const redCaptainInBlue = await blueColumn.locator(`[data-testid="lobby-player-${redCaptain.userId}"]`).count() > 0;

    expect(blueCaptainInRed).toBe(true);
    expect(redCaptainInBlue).toBe(true);
  });
});
