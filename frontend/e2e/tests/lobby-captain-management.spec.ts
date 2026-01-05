import { LobbyRoomPage } from '../fixtures/pages';
import {
  test,
  expect,
  getLobby,
  setupLobbyWithPlayers,
  takeCaptain,
  promoteCaptain,
  kickPlayer,
  proposeSwap,
  getPendingAction,
  approvePendingAction,
  cancelPendingAction,
  UserSession,
  LobbyPlayer,
} from '../fixtures/multi-user';

/**
 * Lobby Captain Management E2E Tests
 *
 * These tests cover the captain management flows in a 10-man lobby:
 * - Take Captain: Player takes captain status from current captain
 * - Promote Captain: Captain promotes a teammate to become captain
 * - Kick Player: Captain kicks a teammate from the lobby
 * - Swap Players: Captain proposes swapping players between teams
 * - Swap Roles: Captain proposes swapping roles within a team
 */

// ========== Helper Functions ==========

function findCaptainOnTeam(players: LobbyPlayer[], team: string): LobbyPlayer | undefined {
  return players.find((p) => p.team === team && p.isCaptain);
}

function findNonCaptainOnTeam(players: LobbyPlayer[], team: string): LobbyPlayer | undefined {
  return players.find((p) => p.team === team && !p.isCaptain);
}

function findUserSession(users: UserSession[], userId: string): UserSession | undefined {
  return users.find((u) => u.user.id === userId);
}

// ========== Take Captain Tests ==========

test.describe('Take Captain Tests', () => {
  test.describe.configure({ mode: 'serial' });

  test('player can take captain from current captain', async ({ createUsers }) => {
    test.setTimeout(120000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    // Find a non-captain player on blue team
    const blueCaptain = findCaptainOnTeam(lobby.players, 'blue');
    const blueNonCaptain = findNonCaptainOnTeam(lobby.players, 'blue');
    expect(blueCaptain).toBeDefined();
    expect(blueNonCaptain).toBeDefined();

    // Get the user session for the non-captain
    const nonCaptainUser = findUserSession(users, blueNonCaptain!.userId);
    expect(nonCaptainUser).toBeDefined();

    // Navigate to lobby
    await nonCaptainUser!.page.goto(`/lobby/${lobby.id}`);
    const lobbyPage = new LobbyRoomPage(nonCaptainUser!.page);

    // Wait for page to load
    await expect(nonCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Should see Take Captain button (not captain yet)
    await lobbyPage.expectTakeCaptainButton();

    // Click Take Captain
    await lobbyPage.clickTakeCaptain();

    // Wait for state update and reload
    await nonCaptainUser!.page.waitForTimeout(1000);
    await nonCaptainUser!.page.reload();
    await expect(nonCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Verify: Should now see Captain Controls
    await lobbyPage.expectCaptainControls();

    // Verify via API
    const updatedLobby = await getLobby(nonCaptainUser!, lobby.id);
    const newCaptain = updatedLobby.players.find((p) => p.userId === blueNonCaptain!.userId);
    expect(newCaptain?.isCaptain).toBe(true);
  });

  test('UI updates correctly after taking captain', async ({ createUsers }) => {
    test.setTimeout(120000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    const blueNonCaptain = findNonCaptainOnTeam(lobby.players, 'blue');
    expect(blueNonCaptain).toBeDefined();

    const nonCaptainUser = findUserSession(users, blueNonCaptain!.userId);
    expect(nonCaptainUser).toBeDefined();

    await nonCaptainUser!.page.goto(`/lobby/${lobby.id}`);
    const lobbyPage = new LobbyRoomPage(nonCaptainUser!.page);

    await expect(nonCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Before: Should see "Player Actions"
    await lobbyPage.expectPlayerActions();

    // Take captain
    await lobbyPage.clickTakeCaptain();

    // Wait for state update and reload
    await nonCaptainUser!.page.waitForTimeout(1000);
    await nonCaptainUser!.page.reload();
    await expect(nonCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // After: Should see "Captain Controls"
    await lobbyPage.expectCaptainControls();

    // Should see Captain badge
    await expect(nonCaptainUser!.page.locator('text=Captain').first()).toBeVisible();
  });
});

// ========== Promote Captain Tests ==========

test.describe('Promote Captain Tests', () => {
  test.describe.configure({ mode: 'serial' });

  test('captain can promote a teammate to captain', async ({ createUsers }) => {
    test.setTimeout(120000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    // Find captain on blue team
    const blueCaptain = findCaptainOnTeam(lobby.players, 'blue');
    expect(blueCaptain).toBeDefined();

    const blueCaptainUser = findUserSession(users, blueCaptain!.userId);
    expect(blueCaptainUser).toBeDefined();

    // Find a teammate to promote
    const teammate = findNonCaptainOnTeam(lobby.players, 'blue');
    expect(teammate).toBeDefined();

    await blueCaptainUser!.page.goto(`/lobby/${lobby.id}`);
    const lobbyPage = new LobbyRoomPage(blueCaptainUser!.page);

    await expect(blueCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Open promote modal
    await lobbyPage.clickPromoteCaptain();

    // Should see modal with teammates
    await expect(blueCaptainUser!.page.locator('text=Promote Captain').nth(1)).toBeVisible();
    await expect(
      blueCaptainUser!.page.locator(`button:has-text("${teammate!.displayName}")`)
    ).toBeVisible();

    // Select teammate
    await lobbyPage.selectPlayerInModal(teammate!.displayName);

    // Wait for state update and reload to see changes
    await blueCaptainUser!.page.waitForTimeout(1000);
    await blueCaptainUser!.page.reload();
    await expect(blueCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Verify: Original captain no longer has captain controls
    await lobbyPage.expectPlayerActions();

    // Verify via API
    const updatedLobby = await getLobby(blueCaptainUser!, lobby.id);
    const newCaptain = updatedLobby.players.find((p) => p.userId === teammate!.userId);
    expect(newCaptain?.isCaptain).toBe(true);
  });

  test('only teammates shown in promote modal', async ({ createUsers }) => {
    test.setTimeout(120000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    const blueCaptain = findCaptainOnTeam(lobby.players, 'blue');
    expect(blueCaptain).toBeDefined();

    const blueCaptainUser = findUserSession(users, blueCaptain!.userId);
    expect(blueCaptainUser).toBeDefined();

    const redPlayers = lobby.players.filter((p) => p.team === 'red');

    await blueCaptainUser!.page.goto(`/lobby/${lobby.id}`);
    const lobbyPage = new LobbyRoomPage(blueCaptainUser!.page);

    await expect(blueCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    await lobbyPage.clickPromoteCaptain();

    // Wait for modal to open
    await expect(blueCaptainUser!.page.locator('text=Promote Captain').nth(1)).toBeVisible();

    // Red team players should NOT be in the modal
    for (const redPlayer of redPlayers) {
      await expect(
        blueCaptainUser!.page.locator(`button:has-text("${redPlayer.displayName}")`)
      ).not.toBeVisible();
    }

    // Self should NOT be in the modal
    await expect(
      blueCaptainUser!.page.locator(`button:has-text("${blueCaptain!.displayName}")`)
    ).not.toBeVisible();

    // Close modal
    await lobbyPage.cancelModal();
  });

  test('UI updates for both old and new captain', async ({ createUsers }) => {
    test.setTimeout(120000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    const blueCaptain = findCaptainOnTeam(lobby.players, 'blue');
    const teammate = findNonCaptainOnTeam(lobby.players, 'blue');
    expect(blueCaptain).toBeDefined();
    expect(teammate).toBeDefined();

    const blueCaptainUser = findUserSession(users, blueCaptain!.userId);
    const teammateUser = findUserSession(users, teammate!.userId);
    expect(blueCaptainUser).toBeDefined();
    expect(teammateUser).toBeDefined();

    // Navigate both users
    await blueCaptainUser!.page.goto(`/lobby/${lobby.id}`);
    await teammateUser!.page.goto(`/lobby/${lobby.id}`);

    await expect(blueCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });
    await expect(teammateUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    const captainPage = new LobbyRoomPage(blueCaptainUser!.page);
    const teammatePage = new LobbyRoomPage(teammateUser!.page);

    // Teammate should see Player Actions before promotion
    await teammatePage.expectPlayerActions();

    // Promote teammate
    await captainPage.clickPromoteCaptain();
    await captainPage.selectPlayerInModal(teammate!.displayName);

    // Wait for state update
    await blueCaptainUser!.page.waitForTimeout(1000);

    // Reload promoted user's page to see updated state
    await teammateUser!.page.reload();

    await expect(teammateUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // New captain should see Captain Controls
    await teammatePage.expectCaptainControls();
  });
});

// ========== Kick Player Tests ==========

test.describe('Kick Player Tests', () => {
  test.describe.configure({ mode: 'serial' });

  test('captain can kick a teammate', async ({ createUsers }) => {
    test.setTimeout(120000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    const blueCaptain = findCaptainOnTeam(lobby.players, 'blue');
    expect(blueCaptain).toBeDefined();

    const blueCaptainUser = findUserSession(users, blueCaptain!.userId);
    expect(blueCaptainUser).toBeDefined();

    const teammate = findNonCaptainOnTeam(lobby.players, 'blue');
    expect(teammate).toBeDefined();

    // Kick teammate via API for reliability
    await kickPlayer(blueCaptainUser!, lobby.id, teammate!.userId);

    // Navigate and verify the UI shows the update
    await blueCaptainUser!.page.goto(`/lobby/${lobby.id}`);
    const lobbyPage = new LobbyRoomPage(blueCaptainUser!.page);

    await expect(blueCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Verify: Teammate is no longer in lobby (only check team columns)
    await lobbyPage.expectPlayerNotInLobby(teammate!.displayName);

    // Verify via API
    const updatedLobby = await getLobby(blueCaptainUser!, lobby.id);
    expect(updatedLobby.players.find((p) => p.userId === teammate!.userId)).toBeUndefined();
  });

  test('only teammates shown in kick modal', async ({ createUsers }) => {
    test.setTimeout(120000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    const blueCaptain = findCaptainOnTeam(lobby.players, 'blue');
    expect(blueCaptain).toBeDefined();

    const blueCaptainUser = findUserSession(users, blueCaptain!.userId);
    expect(blueCaptainUser).toBeDefined();

    const redPlayers = lobby.players.filter((p) => p.team === 'red');

    await blueCaptainUser!.page.goto(`/lobby/${lobby.id}`);
    const lobbyPage = new LobbyRoomPage(blueCaptainUser!.page);

    await expect(blueCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    await lobbyPage.clickKickPlayer();

    // Wait for modal
    await expect(blueCaptainUser!.page.locator('text=Kick Player').nth(1)).toBeVisible();

    // Red team players should NOT be in the modal
    for (const redPlayer of redPlayers) {
      await expect(
        blueCaptainUser!.page.locator(`button:has-text("${redPlayer.displayName}")`)
      ).not.toBeVisible();
    }

    // Self should NOT be in the modal
    await expect(
      blueCaptainUser!.page.locator(`button:has-text("${blueCaptain!.displayName}")`)
    ).not.toBeVisible();

    // Close modal
    await lobbyPage.cancelModal();
  });

  test('kicked player is removed from lobby', async ({ createUsers }) => {
    test.setTimeout(120000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    const blueCaptain = findCaptainOnTeam(lobby.players, 'blue');
    const teammate = findNonCaptainOnTeam(lobby.players, 'blue');
    expect(blueCaptain).toBeDefined();
    expect(teammate).toBeDefined();

    const blueCaptainUser = findUserSession(users, blueCaptain!.userId);
    const teammateUser = findUserSession(users, teammate!.userId);
    expect(blueCaptainUser).toBeDefined();
    expect(teammateUser).toBeDefined();

    // Navigate both
    await blueCaptainUser!.page.goto(`/lobby/${lobby.id}`);
    await teammateUser!.page.goto(`/lobby/${lobby.id}`);

    await expect(blueCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });
    await expect(teammateUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    const captainPage = new LobbyRoomPage(blueCaptainUser!.page);

    // Kick teammate via API for reliability
    await kickPlayer(blueCaptainUser!, lobby.id, teammate!.userId);

    // Reload captain's page
    await blueCaptainUser!.page.reload();
    await expect(blueCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Kicked player should not be visible
    await captainPage.expectPlayerNotInLobby(teammate!.displayName);

    // Verify the kicked user session shows they're no longer in lobby
    await teammateUser!.page.reload();

    // They should either see an error or the controls should be different
    // Check that they no longer see Captain Controls
    await expect(teammateUser!.page.locator('text=Captain Controls')).not.toBeVisible();
  });
});

// ========== Swap Player Tests (Between Teams) ==========

test.describe('Swap Player Tests (Between Teams)', () => {
  test.describe.configure({ mode: 'serial' });

  test('captain can propose player swap between teams', async ({ createUsers }) => {
    test.setTimeout(180000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    const blueCaptain = findCaptainOnTeam(lobby.players, 'blue');
    expect(blueCaptain).toBeDefined();

    const blueCaptainUser = findUserSession(users, blueCaptain!.userId);
    expect(blueCaptainUser).toBeDefined();

    const blueNonCaptain = findNonCaptainOnTeam(lobby.players, 'blue');
    const redNonCaptain = findNonCaptainOnTeam(lobby.players, 'red');
    expect(blueNonCaptain).toBeDefined();
    expect(redNonCaptain).toBeDefined();

    // Propose swap via API (more reliable than UI)
    // Note: Backend expects userId, not LobbyPlayer.id
    await proposeSwap(
      blueCaptainUser!,
      lobby.id,
      blueNonCaptain!.userId,
      redNonCaptain!.userId,
      'players'
    );

    // Navigate to lobby and verify UI shows pending action
    await blueCaptainUser!.page.goto(`/lobby/${lobby.id}`);
    const lobbyPage = new LobbyRoomPage(blueCaptainUser!.page);

    await expect(blueCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Pending action banner should appear
    await lobbyPage.expectPendingActionBanner();
    const actionType = await lobbyPage.getPendingActionType();
    expect(actionType).toContain('Swap Players');
  });

  test('other team captain can approve swap', async ({ createUsers }) => {
    test.setTimeout(180000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    const blueCaptain = findCaptainOnTeam(lobby.players, 'blue');
    const redCaptain = findCaptainOnTeam(lobby.players, 'red');
    expect(blueCaptain).toBeDefined();
    expect(redCaptain).toBeDefined();

    const blueCaptainUser = findUserSession(users, blueCaptain!.userId);
    const redCaptainUser = findUserSession(users, redCaptain!.userId);
    expect(blueCaptainUser).toBeDefined();
    expect(redCaptainUser).toBeDefined();

    const blueNonCaptain = findNonCaptainOnTeam(lobby.players, 'blue');
    const redNonCaptain = findNonCaptainOnTeam(lobby.players, 'red');
    expect(blueNonCaptain).toBeDefined();
    expect(redNonCaptain).toBeDefined();

    // Propose swap via API for speed
    await proposeSwap(
      blueCaptainUser!,
      lobby.id,
      blueNonCaptain!.id,
      redNonCaptain!.id,
      'players'
    );

    // Navigate red captain
    await redCaptainUser!.page.goto(`/lobby/${lobby.id}`);
    const redLobbyPage = new LobbyRoomPage(redCaptainUser!.page);

    await expect(redCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Red captain should see pending action banner
    await redLobbyPage.expectPendingActionBanner();

    // Red captain should see Approve button
    await redLobbyPage.expectApproveButton();

    // Red captain approves
    await redLobbyPage.clickApprovePendingAction();

    // Wait for state update
    await redCaptainUser!.page.waitForTimeout(1000);

    // Reload to see updated state
    await redCaptainUser!.page.reload();

    // Verify swap occurred via API
    const updatedLobby = await getLobby(redCaptainUser!, lobby.id);
    const swappedBlue = updatedLobby.players.find((p) => p.userId === blueNonCaptain!.userId);
    const swappedRed = updatedLobby.players.find((p) => p.userId === redNonCaptain!.userId);

    expect(swappedBlue?.team).toBe('red');
    expect(swappedRed?.team).toBe('blue');
  });

  test('swap executes after both captains approve', async ({ createUsers }) => {
    test.setTimeout(180000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    const blueCaptain = findCaptainOnTeam(lobby.players, 'blue');
    const redCaptain = findCaptainOnTeam(lobby.players, 'red');
    const blueCaptainUser = findUserSession(users, blueCaptain!.userId);
    const redCaptainUser = findUserSession(users, redCaptain!.userId);

    const blueNonCaptain = findNonCaptainOnTeam(lobby.players, 'blue');
    const redNonCaptain = findNonCaptainOnTeam(lobby.players, 'red');

    // Propose swap via API
    const pendingAction = await proposeSwap(
      blueCaptainUser!,
      lobby.id,
      blueNonCaptain!.id,
      redNonCaptain!.id,
      'players'
    );

    // Blue captain already auto-approved (as proposer)
    expect(pendingAction.approvedByBlue).toBe(true);
    expect(pendingAction.approvedByRed).toBe(false);

    // Red captain approves via API
    await approvePendingAction(redCaptainUser!, lobby.id, pendingAction.id);

    // Verify swap executed
    const updatedLobby = await getLobby(blueCaptainUser!, lobby.id);
    const swappedBlue = updatedLobby.players.find((p) => p.userId === blueNonCaptain!.userId);
    const swappedRed = updatedLobby.players.find((p) => p.userId === redNonCaptain!.userId);

    expect(swappedBlue?.team).toBe('red');
    expect(swappedRed?.team).toBe('blue');

    // Pending action should be gone
    const currentAction = await getPendingAction(blueCaptainUser!, lobby.id);
    expect(currentAction).toBeNull();
  });

  test('proposer can cancel swap', async ({ createUsers }) => {
    test.setTimeout(180000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    const blueCaptain = findCaptainOnTeam(lobby.players, 'blue');
    const blueCaptainUser = findUserSession(users, blueCaptain!.userId);

    const blueNonCaptain = findNonCaptainOnTeam(lobby.players, 'blue');
    const redNonCaptain = findNonCaptainOnTeam(lobby.players, 'red');

    // Propose swap via API
    await proposeSwap(
      blueCaptainUser!,
      lobby.id,
      blueNonCaptain!.id,
      redNonCaptain!.id,
      'players'
    );

    // Navigate to lobby
    await blueCaptainUser!.page.goto(`/lobby/${lobby.id}`);
    const lobbyPage = new LobbyRoomPage(blueCaptainUser!.page);

    await expect(blueCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    await lobbyPage.expectPendingActionBanner();

    // Cancel the swap via UI
    await lobbyPage.clickCancelPendingAction();

    // Wait for state update
    await blueCaptainUser!.page.waitForTimeout(1000);

    // Banner should disappear
    await lobbyPage.expectNoPendingActionBanner();

    // Verify via API
    const action = await getPendingAction(blueCaptainUser!, lobby.id);
    expect(action).toBeNull();
  });

  test('pending action banner shows correct info', async ({ createUsers }) => {
    test.setTimeout(180000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    const blueCaptain = findCaptainOnTeam(lobby.players, 'blue');
    const blueCaptainUser = findUserSession(users, blueCaptain!.userId);

    const blueNonCaptain = findNonCaptainOnTeam(lobby.players, 'blue');
    const redNonCaptain = findNonCaptainOnTeam(lobby.players, 'red');

    // Propose swap via API
    await proposeSwap(
      blueCaptainUser!,
      lobby.id,
      blueNonCaptain!.id,
      redNonCaptain!.id,
      'players'
    );

    await blueCaptainUser!.page.goto(`/lobby/${lobby.id}`);
    const lobbyPage = new LobbyRoomPage(blueCaptainUser!.page);

    await expect(blueCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Check banner shows correct info
    await lobbyPage.expectPendingActionBanner();

    const actionType = await lobbyPage.getPendingActionType();
    expect(actionType).toContain('Swap Players');

    // Should show player names in description
    await expect(
      blueCaptainUser!.page.locator(`text=${blueNonCaptain!.displayName}`)
    ).toBeVisible();
    await expect(
      blueCaptainUser!.page.locator(`text=${redNonCaptain!.displayName}`)
    ).toBeVisible();

    // Blue captain should see Approved badge (as proposer)
    await lobbyPage.expectApprovedBadge();
  });
});

// ========== Swap Role Tests (Within Team) ==========

test.describe('Swap Role Tests (Within Team)', () => {
  test.describe.configure({ mode: 'serial' });

  test('captain can propose role swap within team', async ({ createUsers }) => {
    test.setTimeout(180000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    const blueCaptain = findCaptainOnTeam(lobby.players, 'blue');
    expect(blueCaptain).toBeDefined();

    const blueCaptainUser = findUserSession(users, blueCaptain!.userId);
    expect(blueCaptainUser).toBeDefined();

    // Find two blue team players (not captain)
    const blueTeammates = lobby.players.filter(
      (p) => p.team === 'blue' && !p.isCaptain
    );
    expect(blueTeammates.length).toBeGreaterThanOrEqual(2);

    // Propose role swap via API
    await proposeSwap(
      blueCaptainUser!,
      lobby.id,
      blueTeammates[0].id,
      blueTeammates[1].id,
      'roles'
    );

    // Navigate to lobby and verify UI shows pending action
    await blueCaptainUser!.page.goto(`/lobby/${lobby.id}`);
    const lobbyPage = new LobbyRoomPage(blueCaptainUser!.page);

    await expect(blueCaptainUser!.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 10000 });

    // Pending action banner should appear
    await lobbyPage.expectPendingActionBanner();
    const actionType = await lobbyPage.getPendingActionType();
    expect(actionType).toContain('Swap Roles');
  });

  test('role swap executes after both captains approve', async ({ createUsers }) => {
    test.setTimeout(180000);

    const { lobby, users } = await setupLobbyWithPlayers(createUsers);

    const blueCaptain = findCaptainOnTeam(lobby.players, 'blue');
    const redCaptain = findCaptainOnTeam(lobby.players, 'red');
    const blueCaptainUser = findUserSession(users, blueCaptain!.userId);
    const redCaptainUser = findUserSession(users, redCaptain!.userId);

    const blueTeammates = lobby.players.filter(
      (p) => p.team === 'blue' && !p.isCaptain
    );

    const player1Role = blueTeammates[0].assignedRole;
    const player2Role = blueTeammates[1].assignedRole;

    // Propose role swap via API
    const pendingAction = await proposeSwap(
      blueCaptainUser!,
      lobby.id,
      blueTeammates[0].id,
      blueTeammates[1].id,
      'roles'
    );

    // Red captain approves via API
    await approvePendingAction(redCaptainUser!, lobby.id, pendingAction.id);

    // Verify roles were swapped
    const updatedLobby = await getLobby(blueCaptainUser!, lobby.id);
    const updatedPlayer1 = updatedLobby.players.find(
      (p) => p.userId === blueTeammates[0].userId
    );
    const updatedPlayer2 = updatedLobby.players.find(
      (p) => p.userId === blueTeammates[1].userId
    );

    expect(updatedPlayer1?.assignedRole).toBe(player2Role);
    expect(updatedPlayer2?.assignedRole).toBe(player1Role);
  });
});
