import { test, expect } from '@playwright/test';
import { createTestUser, setAuthToken, joinLobby, initializeRoleProfiles } from '../helpers/test-utils';

const API_BASE = 'http://localhost:9999/api/v1';

test.describe('BUG-009: Captain Indicator Shows for All Players in Lobby UI', () => {
  test('should only show captain badge for actual captains', async ({ page }) => {
    // Create test users
    const user1 = await createTestUser(page, 'bug009_captain1');
    const user2 = await createTestUser(page, 'bug009_player2');
    const user3 = await createTestUser(page, 'bug009_player3');
    const user4 = await createTestUser(page, 'bug009_player4');

    // Initialize role profiles for all users
    await initializeRoleProfiles(page, user1.token);
    await initializeRoleProfiles(page, user2.token);
    await initializeRoleProfiles(page, user3.token);
    await initializeRoleProfiles(page, user4.token);

    // User1 creates a lobby (becomes blue captain)
    const createRes = await page.request.post(`${API_BASE}/lobbies`, {
      headers: { Authorization: `Bearer ${user1.token}` },
      data: {
        draftMode: 'pro_play',
        timerDurationSeconds: 30,
      },
    });
    expect(createRes.ok()).toBeTruthy();
    const lobby = await createRes.json();

    // Other users join the lobby
    await joinLobby(page, user2.token, lobby.id);
    await joinLobby(page, user3.token, lobby.id);
    await joinLobby(page, user4.token, lobby.id);

    // Get lobby state via API to verify captain flags
    const getLobbyRes = await page.request.get(`${API_BASE}/lobbies/${lobby.id}`, {
      headers: { Authorization: `Bearer ${user1.token}` },
    });
    expect(getLobbyRes.ok()).toBeTruthy();
    const lobbyData = await getLobbyRes.json();

    // Verify API returns correct captain data
    // user1 (blue side) should be captain
    // The first player on red side should also be captain
    const bluePlayers = lobbyData.players.filter((p: any) => p.team === 'blue');
    const redPlayers = lobbyData.players.filter((p: any) => p.team === 'red');

    // Count actual captains from API
    const captainCount = lobbyData.players.filter((p: any) => p.isCaptain).length;
    console.log('API captain count:', captainCount);
    console.log('Players:', lobbyData.players.map((p: any) => ({
      displayName: p.displayName,
      team: p.team,
      isCaptain: p.isCaptain
    })));

    // There should be at most 2 captains (one per team)
    expect(captainCount).toBeLessThanOrEqual(2);

    // Set auth token for user1 and navigate to lobby
    await setAuthToken(page, user1);
    await page.goto(`/lobby/${lobby.id}`);

    // Wait for lobby to load
    await expect(page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Count captain badges in the UI
    // The "C" badge has class text-lol-gold and title "Captain"
    const captainBadges = page.locator('span[title="Captain"]');
    const badgeCount = await captainBadges.count();

    console.log('UI captain badge count:', badgeCount);

    // Should match the number of actual captains from API (at most 2)
    expect(badgeCount).toBeLessThanOrEqual(2);
    expect(badgeCount).toBe(captainCount);
  });

  test('should show captain badge only for blue and red captains', async ({ page }) => {
    // Create 6 users (3 per team minimum for this test)
    const users = await Promise.all([
      createTestUser(page, 'bug009_6p_u1'),
      createTestUser(page, 'bug009_6p_u2'),
      createTestUser(page, 'bug009_6p_u3'),
      createTestUser(page, 'bug009_6p_u4'),
      createTestUser(page, 'bug009_6p_u5'),
      createTestUser(page, 'bug009_6p_u6'),
    ]);

    // Initialize role profiles for all
    for (const user of users) {
      await initializeRoleProfiles(page, user.token);
    }

    // First user creates lobby
    const createRes = await page.request.post(`${API_BASE}/lobbies`, {
      headers: { Authorization: `Bearer ${users[0].token}` },
      data: {
        draftMode: 'pro_play',
        timerDurationSeconds: 30,
      },
    });
    expect(createRes.ok()).toBeTruthy();
    const lobby = await createRes.json();

    // Other 5 users join
    for (let i = 1; i < users.length; i++) {
      await joinLobby(page, users[i].token, lobby.id);
    }

    // Get final lobby state
    const getLobbyRes = await page.request.get(`${API_BASE}/lobbies/${lobby.id}`, {
      headers: { Authorization: `Bearer ${users[0].token}` },
    });
    expect(getLobbyRes.ok()).toBeTruthy();
    const lobbyData = await getLobbyRes.json();

    // Count captains
    const captains = lobbyData.players.filter((p: any) => p.isCaptain);
    console.log('API captains:', captains.map((c: any) => ({
      displayName: c.displayName,
      team: c.team,
      isCaptain: c.isCaptain
    })));

    // Verify: should have exactly 2 captains (one blue, one red)
    expect(captains.length).toBe(2);

    // Navigate to lobby as first user
    await setAuthToken(page, users[0]);
    await page.goto(`/lobby/${lobby.id}`);

    // Wait for lobby to load
    await expect(page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Count captain badges
    const captainBadges = page.locator('span[title="Captain"]');
    const badgeCount = await captainBadges.count();

    console.log('UI captain badge count:', badgeCount);

    // Should show exactly 2 captain badges
    expect(badgeCount).toBe(2);
  });
});
