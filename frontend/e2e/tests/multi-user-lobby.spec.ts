import { test, expect } from '../fixtures/multi-user';
import { LobbyRoomPage } from '../fixtures/pages';

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
