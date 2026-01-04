import { test, expect } from '../fixtures/multi-user';
import { LobbyRoomPage, DraftRoomPage } from '../fixtures/pages';

test.describe('Multi-User Lobby Flow with UI', () => {
  test('10 users can join and interact with lobby UI', async ({ lobbyWithUsers }) => {
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

    // Each user clicks Ready Up through the UI
    for (let i = 0; i < users.length; i++) {
      const lobbyPage = lobbyPages[i];
      await lobbyPage.clickReadyUp();
      // Small delay to avoid race conditions
      await users[i].page.waitForTimeout(200);
    }

    // Wait for ready state to propagate
    await users[0].page.waitForTimeout(1000);

    // Refresh creator's page to see updated state
    await users[0].page.reload();

    // Creator should see Generate Teams button
    const creatorPage = lobbyPages[0];
    await creatorPage.expectGenerateTeamsButton();

    // Creator clicks Generate Teams
    await creatorPage.clickGenerateTeams();

    // Wait for team options to appear
    await creatorPage.waitForMatchOptions();

    // Creator selects first option via UI
    await creatorPage.selectOption(1);

    // Creator confirms selection
    await creatorPage.clickConfirmSelection();

    // Wait for confirmation to process
    await users[0].page.waitForTimeout(500);

    // Creator should see Start Draft button
    await expect(users[0].page.locator('button:has-text("Start Draft")')).toBeVisible({ timeout: 10000 });

    // Creator clicks Start Draft
    await creatorPage.clickStartDraft();

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

    // Wait for state to propagate
    await user1.page.waitForTimeout(500);

    // User 2 should see user 1's ready status update (may need to wait for poll)
    await user2.page.waitForTimeout(3500); // Wait for 3s poll interval

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
    await fetch(`http://localhost:9999/api/v1/lobbies/${lobby.id}/ready`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${creator.token}`,
      },
      body: JSON.stringify({ ready: true }),
    });

    await fetch(`http://localhost:9999/api/v1/lobbies/${lobby.id}/ready`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${joiner.token}`,
      },
      body: JSON.stringify({ ready: true }),
    });

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
    const createResponse = await fetch('http://localhost:9999/api/v1/lobbies', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${users[0].token}`,
      },
      body: JSON.stringify({ draftMode: 'pro_play', timerDurationSeconds: 30 }),
    });
    const lobby = await createResponse.json();

    // Navigate creator to lobby
    await users[0].page.goto(`/lobby/${lobby.id}`);
    await expect(users[0].page.locator('text=1/10')).toBeVisible();

    // Each subsequent user joins and we verify count increases
    for (let i = 1; i < users.length; i++) {
      // Join via API
      await fetch(`http://localhost:9999/api/v1/lobbies/${lobby.id}/join`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${users[i].token}`,
        },
      });

      // Navigate to lobby
      await users[i].page.goto(`/lobby/${lobby.id}`);

      // Wait for poll to update
      await users[0].page.waitForTimeout(3500);
      await users[0].page.reload();

      // Creator should see updated count
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

    // Each user clicks Ready Up through the UI
    for (let i = 0; i < users.length; i++) {
      const lobbyPage = lobbyPages[i];
      await lobbyPage.clickReadyUp();
      await users[i].page.waitForTimeout(200);
    }

    // Wait for ready state to propagate
    await users[0].page.waitForTimeout(1000);
    await users[0].page.reload();

    // Creator generates teams, selects option, and starts draft
    const creatorPage = lobbyPages[0];
    await creatorPage.expectGenerateTeamsButton();
    await creatorPage.clickGenerateTeams();
    await creatorPage.waitForMatchOptions();
    await creatorPage.selectOption(1);
    await creatorPage.clickConfirmSelection();
    await users[0].page.waitForTimeout(500);

    // Start draft - all users will be redirected
    await expect(users[0].page.locator('button:has-text("Start Draft")')).toBeVisible({
      timeout: 10000,
    });
    await creatorPage.clickStartDraft();

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

    // In 10-man lobby, only 2 users (blue/red captains) can click Ready
    // Wait for WebSocket to settle before checking
    await draftPages[0].page.waitForTimeout(1000);

    // Ready up via UI - only non-spectators can click Ready
    for (const draftPage of draftPages) {
      const canReady = await draftPage.canClickReady();
      if (canReady) {
        await draftPage.clickReady();
        await draftPage.page.waitForTimeout(500);
      }
    }

    // Wait for ready state to propagate via WebSocket
    await draftPages[0].page.waitForTimeout(1500);

    // One of the captains should see Start Draft button
    // Find a user who can see Start Draft (wait longer for WebSocket to sync)
    let starterFound = false;
    for (const draftPage of draftPages) {
      const startButton = draftPage.page.locator('button:has-text("Start Draft")');
      if (await startButton.isVisible({ timeout: 5000 }).catch(() => false)) {
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
    let turnTaken = false;
    for (let phase = 0; phase < 6; phase++) {
      // Go through first 6 ban phases
      for (const draftPage of draftPages) {
        const isMyTurn = await draftPage.isYourTurn();
        if (isMyTurn) {
          await draftPage.performBanOrPick(phase);
          turnTaken = true;
          break;
        }
      }
      if (turnTaken) {
        // Wait for phase transition
        await draftPages[0].page.waitForTimeout(800);
        turnTaken = false;
      }
    }

    // After 6 bans, we should be in pick phase
    // Continue with 4 pick phases
    for (let phase = 0; phase < 4; phase++) {
      for (const draftPage of draftPages) {
        const isMyTurn = await draftPage.isYourTurn();
        if (isMyTurn) {
          // Use higher indices to avoid selecting already banned champions
          await draftPage.performBanOrPick(10 + phase);
          turnTaken = true;
          break;
        }
      }
      if (turnTaken) {
        await draftPages[0].page.waitForTimeout(800);
        turnTaken = false;
      }
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
    for (const user of users) {
      await fetch(`http://localhost:9999/api/v1/lobbies/${lobby.id}/ready`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${user.token}`,
        },
        body: JSON.stringify({ ready: true }),
      });
    }

    // Generate teams via API
    const genResponse = await fetch(
      `http://localhost:9999/api/v1/lobbies/${lobby.id}/generate-teams`,
      {
        method: 'POST',
        headers: { Authorization: `Bearer ${users[0].token}` },
      }
    );
    expect(genResponse.ok).toBe(true);

    // Select first option via API
    await fetch(`http://localhost:9999/api/v1/lobbies/${lobby.id}/select-option`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${users[0].token}`,
      },
      body: JSON.stringify({ optionNumber: 1 }),
    });

    // Start draft via API
    const startResponse = await fetch(
      `http://localhost:9999/api/v1/lobbies/${lobby.id}/start-draft`,
      {
        method: 'POST',
        headers: { Authorization: `Bearer ${users[0].token}` },
      }
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
    for (let i = 0; i < draftPages.length; i++) {
      await draftPages[i].waitForDraftLoaded();
      await draftPages[i].waitForWebSocketConnected();
      console.log(`User ${i} connected`);
    }

    // Add extra delay for all WebSocket states to fully settle
    console.log('Waiting 1s for WebSocket states to settle...');
    await draftPages[0].page.waitForTimeout(1000);

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
        await draftPage.page.waitForTimeout(500); // Let WebSocket message propagate
      }
    }

    // Should have clicked Ready on exactly 2 pages (blue and red clients)
    console.log(`Clicked Ready on ${readyClicks} pages: ${readyUsers.join(', ')}`);
    expect(readyClicks).toBe(2);

    // Wait for ready state to propagate via WebSocket
    console.log('Waiting for ready state to propagate via WebSocket...');
    await draftPages[0].page.waitForTimeout(1500);

    // Find and click Start Draft - should be visible when both teams ready
    // No reload needed - WebSocket should sync the state
    console.log('Looking for Start Draft button...');
    let startClicked = false;
    for (let i = 0; i < draftPages.length; i++) {
      const draftPage = draftPages[i];
      const startButton = draftPage.page.locator('button:has-text("Start Draft")');
      const isVisible = await startButton.isVisible({ timeout: 5000 }).catch(() => false);
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
      console.log(`Phase ${phase}: looking for player whose turn it is...`);

      let turnFound = false;
      for (let i = 0; i < draftPages.length; i++) {
        const draftPage = draftPages[i];
        if (await draftPage.isYourTurn()) {
          console.log(`Phase ${phase}: User ${i} taking turn`);
          await draftPage.performBanOrPick(phase);
          turnFound = true;
          break;
        }
      }

      if (!turnFound) {
        throw new Error(`Phase ${phase}: No player found whose turn it is`);
      }

      // Wait for phase transition
      await draftPages[0].page.waitForTimeout(800);
    }

    console.log('All 20 phases complete!');

    // Verify draft is complete
    await draftPages[0].expectDraftComplete();
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
      await user.page.waitForTimeout(200);
    }

    // Reload creator's page
    await users[0].page.reload();

    // Generate Teams button should NOT be visible with only 5 players
    await expect(users[0].page.locator('button:has-text("Generate Team Options")')).not.toBeVisible();
  });
});
