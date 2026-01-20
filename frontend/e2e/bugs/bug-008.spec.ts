import { test, expect } from '@playwright/test';
import {
  createTestUser,
  setAuthToken,
  createLobby,
  joinLobby,
  initializeRoleProfiles,
  kickPlayer,
  getLobby,
} from '../helpers/test-utils';

test.describe('BUG-008: Kicked Player Receives No Notification or Redirect', () => {
  test('kicked player should see alert and be redirected to home', async ({ page }) => {
    // Create captain and several players, then find one on the same team to kick
    const captain = await createTestUser(page, 'bug008_captain');
    await initializeRoleProfiles(page, captain.token);

    // Create multiple players
    const players = [];
    for (let i = 0; i < 6; i++) {
      const p = await createTestUser(page, `bug008_p${i}`);
      await initializeRoleProfiles(page, p.token);
      players.push(p);
    }

    // Captain creates lobby (captain will be on blue team)
    const lobby = await createLobby(page, captain.token);
    const lobbyId = lobby.id;

    // All players join the lobby
    for (const p of players) {
      await joinLobby(page, p.token, lobbyId);
    }

    // Find a player on the same team as captain (blue team) to kick
    const lobbyData = await getLobby(page, captain.token, lobbyId);
    const captainData = lobbyData.players.find((p: { userId: string }) => p.userId === captain.id);
    const captainTeam = captainData.team;

    // Find a non-captain player on the same team
    const playerToKickData = lobbyData.players.find(
      (p: { userId: string; team: string; isCaptain: boolean }) =>
        p.team === captainTeam && p.userId !== captain.id && !p.isCaptain
    );

    expect(playerToKickData).toBeDefined();

    // Find the corresponding test user
    const playerToKick = players.find(p => p.id === playerToKickData.userId)!;
    expect(playerToKick).toBeDefined();

    // Set auth token for the player to be kicked and navigate to lobby
    await setAuthToken(page, playerToKick);
    await page.goto(`/lobby/${lobbyId}`);

    // Wait for lobby to load
    await page.waitForSelector('[data-testid="lobby-code-display"]', { timeout: 10000 });

    // Set up dialog handler to capture the alert message
    let alertMessage = '';
    page.on('dialog', async dialog => {
      alertMessage = dialog.message();
      await dialog.accept();
    });

    // Captain kicks the player via API
    await kickPlayer(page, captain.token, lobbyId, playerToKick.id);

    // Wait for the alert to appear and the redirect to happen
    await page.waitForURL('/', { timeout: 10000 });

    // Verify the alert was shown with appropriate message
    expect(alertMessage).toContain('kicked');
    expect(alertMessage).toContain(captain.displayName);

    // Verify we're now on the home page
    await expect(page).toHaveURL('/');
  });

  test('kicked player should not remain in ghost state on lobby page', async ({ page }) => {
    // Create captain and several players, then find one on same team to kick
    const captain = await createTestUser(page, 'bug008b_captain');
    await initializeRoleProfiles(page, captain.token);

    // Create multiple players
    const players = [];
    for (let i = 0; i < 6; i++) {
      const p = await createTestUser(page, `bug008b_p${i}`);
      await initializeRoleProfiles(page, p.token);
      players.push(p);
    }

    // Captain creates lobby
    const lobby = await createLobby(page, captain.token);
    const lobbyId = lobby.id;

    // All players join
    for (const p of players) {
      await joinLobby(page, p.token, lobbyId);
    }

    // Find a player on the same team as captain to kick
    const lobbyData = await getLobby(page, captain.token, lobbyId);
    const captainData = lobbyData.players.find((p: { userId: string }) => p.userId === captain.id);
    const captainTeam = captainData.team;

    const playerToKickData = lobbyData.players.find(
      (p: { userId: string; team: string; isCaptain: boolean }) =>
        p.team === captainTeam && p.userId !== captain.id && !p.isCaptain
    );

    expect(playerToKickData).toBeDefined();
    const playerToKick = players.find(p => p.id === playerToKickData.userId)!;

    // Set auth token for playerToKick and navigate to lobby
    await setAuthToken(page, playerToKick);
    await page.goto(`/lobby/${lobbyId}`);

    // Wait for lobby to load
    await page.waitForSelector('[data-testid="lobby-code-display"]', { timeout: 10000 });

    // Set up dialog handler
    page.on('dialog', async dialog => {
      await dialog.accept();
    });

    // Captain kicks playerToKick
    await kickPlayer(page, captain.token, lobbyId, playerToKick.id);

    // Verify player is redirected to home (not stuck in ghost state)
    await page.waitForURL('/', { timeout: 10000 });

    // Verify player can no longer see the lobby page
    await expect(page).toHaveURL('/');
  });
});
