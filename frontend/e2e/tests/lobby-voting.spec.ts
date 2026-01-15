import { test, expect, UserSession } from '../fixtures/multi-user';
import { LobbyRoomPage } from '../fixtures/pages';
import { TIMEOUTS } from '../helpers/wait-strategies';
import { retryWithBackoff } from '../helpers/wait-strategies';

// These tests involve multiple users and WebSocket connections - run serially to avoid interference
test.describe.configure({ mode: 'serial' });

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

/**
 * Helper to create a lobby with voting enabled via API
 */
async function createVotingLobby(
  token: string,
  votingMode: 'majority' | 'unanimous' | 'captain_override' = 'majority'
): Promise<{ id: string; shortCode: string }> {
  const response = await apiCall('/lobbies', token, {
    body: {
      draftMode: 'pro_play',
      timerDurationSeconds: 90,
      votingEnabled: true,
      votingMode,
    },
  });
  return response.json();
}

/**
 * Helper to set up a lobby with voting and 10 ready players
 */
async function setupVotingLobbyWith10Players(
  users: UserSession[],
  votingMode: 'majority' | 'unanimous' | 'captain_override' = 'majority'
) {
  const lobby = await createVotingLobby(users[0].token, votingMode);

  // All users except creator join
  await Promise.all(
    users.slice(1).map((user) => apiCall(`/lobbies/${lobby.id}/join`, user.token))
  );

  // Ready up all users
  await Promise.all(
    users.map((user) =>
      apiCall(`/lobbies/${lobby.id}/ready`, user.token, { body: { ready: true } })
    )
  );

  // Generate teams
  await apiCall(`/lobbies/${lobby.id}/generate-teams`, users[0].token);

  return lobby;
}

test.describe('Lobby Voting Feature', () => {
  test('lobby with voting enabled shows vote UI instead of select UI', async ({ lobbyWithUsers }) => {
    test.setTimeout(120000);

    // Create 10 users
    const { users } = await lobbyWithUsers(10);

    // Create lobby with voting enabled via API
    const lobby = await createVotingLobby(users[0].token, 'majority');

    // All users except creator join
    await Promise.all(
      users.slice(1).map((user) => apiCall(`/lobbies/${lobby.id}/join`, user.token))
    );

    // Ready up all users
    await Promise.all(
      users.map((user) =>
        apiCall(`/lobbies/${lobby.id}/ready`, user.token, { body: { ready: true } })
      )
    );

    // Generate teams via API
    await apiCall(`/lobbies/${lobby.id}/generate-teams`, users[0].token);

    // Navigate all users to the lobby page
    for (const user of users) {
      await user.page.goto(`/lobby/${lobby.id}`);
    }

    // Wait for page to load
    for (const user of users) {
      await expect(user.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    }

    // All users should see "Vote for Team Composition" heading (not "Select Team Composition")
    for (const user of users) {
      await expect(user.page.locator('text=Vote for Team Composition')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    }

    // All users should see "Vote for This" buttons (not "Select This Option")
    for (const user of users) {
      await expect(user.page.locator('button:has-text("Vote for This")').first()).toBeVisible();
    }
  });

  test('users can cast votes and see vote counts update', async ({ lobbyWithUsers }) => {
    test.setTimeout(120000);

    // Create 10 users
    const { users } = await lobbyWithUsers(10);
    const lobby = await setupVotingLobbyWith10Players(users, 'majority');

    // Navigate all users to the lobby page
    const lobbyPages: LobbyRoomPage[] = [];
    for (const user of users) {
      await user.page.goto(`/lobby/${lobby.id}`);
      lobbyPages.push(new LobbyRoomPage(user.page));
    }

    // Wait for page to load
    for (const user of users) {
      await expect(user.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    }

    // Wait for match options to load
    await lobbyPages[0].waitForMatchOptions();

    // First user votes for option 1
    await lobbyPages[0].voteForOption(1);

    // Verify their vote is recorded
    await lobbyPages[0].expectVotedOnOption(1);

    // Reload another user's page to see vote count
    await users[1].page.reload();
    await expect(users[1].page.locator('text=Vote for Team Composition')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Should see 1 vote on option 1
    await expect(users[1].page.locator('[data-testid="match-option-1"]').locator('text=1 vote')).toBeVisible();
  });

  test('user can change their vote', async ({ lobbyWithUsers }) => {
    test.setTimeout(120000);

    // Create 10 users
    const { users } = await lobbyWithUsers(10);
    const lobby = await setupVotingLobbyWith10Players(users, 'majority');

    // Navigate first user to the lobby page
    await users[0].page.goto(`/lobby/${lobby.id}`);
    const lobbyPage = new LobbyRoomPage(users[0].page);

    // Wait for page to load
    await expect(users[0].page.locator('text=10-Man Lobby')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await lobbyPage.waitForMatchOptions();

    // Vote for option 1
    await lobbyPage.voteForOption(1);
    await lobbyPage.expectVotedOnOption(1);

    // Wait a moment for state to update
    await users[0].page.waitForTimeout(500);

    // Change vote to option 2
    await lobbyPage.voteForOption(2);
    await lobbyPage.expectVotedOnOption(2);

    // Option 1 should no longer show "Your Vote"
    const option1 = users[0].page.locator('[data-testid="match-option-1"]');
    await expect(option1.locator('text=Your Vote')).not.toBeVisible();
  });

  test('voting banner shows progress and captain can finalize', async ({ lobbyWithUsers }) => {
    test.setTimeout(180000);

    // Create 10 users
    const { users } = await lobbyWithUsers(10);
    const lobby = await setupVotingLobbyWith10Players(users, 'majority');

    // Navigate all users to the lobby page
    const lobbyPages: LobbyRoomPage[] = [];
    for (const user of users) {
      await user.page.goto(`/lobby/${lobby.id}`);
      lobbyPages.push(new LobbyRoomPage(user.page));
    }

    // Wait for page to load
    for (const user of users) {
      await expect(user.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    }

    // Wait for match options to load
    await lobbyPages[0].waitForMatchOptions();

    // Voting banner should be visible
    await lobbyPages[0].expectVotingBanner();

    // Have 6 users vote for option 1 (majority = 6/10 > 50%)
    for (let i = 0; i < 6; i++) {
      await lobbyPages[i].voteForOption(1);
      await lobbyPages[i].expectVotedOnOption(1);
    }

    // Reload first user's page to see updated voting status
    await users[0].page.reload();
    await expect(users[0].page.locator('text=10-Man Lobby')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Voting banner should show progress
    await lobbyPages[0].expectVotingBanner();

    // Captain (first user) should be able to finalize (majority reached)
    await expect(users[0].page.locator('button:has-text("Finalize Vote")')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  });

  test('captain override mode allows force selection', async ({ lobbyWithUsers }) => {
    test.setTimeout(180000);

    // Create 10 users
    const { users } = await lobbyWithUsers(10);
    const lobby = await setupVotingLobbyWith10Players(users, 'captain_override');

    // Navigate first user (captain) to the lobby page
    await users[0].page.goto(`/lobby/${lobby.id}`);
    const lobbyPage = new LobbyRoomPage(users[0].page);

    // Wait for page to load
    await expect(users[0].page.locator('text=10-Man Lobby')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await lobbyPage.waitForMatchOptions();

    // Have just 1 user vote (not majority)
    await lobbyPage.voteForOption(1);
    await lobbyPage.expectVotedOnOption(1);

    // Reload to get voting status
    await users[0].page.reload();
    await expect(users[0].page.locator('text=10-Man Lobby')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Captain should see Force Option button (captain_override mode)
    await expect(users[0].page.locator('button:has-text("Force Option")')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  });

  test('unanimous mode requires all votes', async ({ lobbyWithUsers }) => {
    test.setTimeout(180000);

    // Create 10 users
    const { users } = await lobbyWithUsers(10);
    const lobby = await setupVotingLobbyWith10Players(users, 'unanimous');

    // Navigate all users to the lobby page
    const lobbyPages: LobbyRoomPage[] = [];
    for (const user of users) {
      await user.page.goto(`/lobby/${lobby.id}`);
      lobbyPages.push(new LobbyRoomPage(user.page));
    }

    // Wait for page to load
    for (const user of users) {
      await expect(user.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    }

    // Wait for match options to load
    await lobbyPages[0].waitForMatchOptions();

    // Have 9 users vote for option 1 (not unanimous yet)
    for (let i = 0; i < 9; i++) {
      await lobbyPages[i].voteForOption(1);
      await lobbyPages[i].expectVotedOnOption(1);
    }

    // Reload first user's page to see updated voting status
    await users[0].page.reload();
    await expect(users[0].page.locator('text=10-Man Lobby')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Should NOT see Finalize button (9/10 is not unanimous)
    await expect(users[0].page.locator('button:has-text("Finalize Vote")')).not.toBeVisible();

    // Last user votes
    await users[9].page.reload();
    await expect(users[9].page.locator('text=Vote for Team Composition')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await lobbyPages[9].voteForOption(1);
    await lobbyPages[9].expectVotedOnOption(1);

    // Reload captain's page
    await users[0].page.reload();
    await expect(users[0].page.locator('text=10-Man Lobby')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Now should see Finalize button (10/10 unanimous)
    await expect(users[0].page.locator('button:has-text("Finalize Vote")')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  });
});

test.describe('Lobby Voting Voters Display', () => {
  test('users can see who voted for each option', async ({ lobbyWithUsers }) => {
    test.setTimeout(120000);

    // Create 10 users
    const { users } = await lobbyWithUsers(10);
    const lobby = await setupVotingLobbyWith10Players(users, 'majority');

    // Navigate all users to the lobby page
    const lobbyPages: LobbyRoomPage[] = [];
    for (const user of users) {
      await user.page.goto(`/lobby/${lobby.id}`);
      lobbyPages.push(new LobbyRoomPage(user.page));
    }

    // Wait for page to load
    for (const user of users) {
      await expect(user.page.locator('text=10-Man Lobby')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    }

    // Wait for match options to load
    await lobbyPages[0].waitForMatchOptions();

    // Have first 3 users vote for option 1
    for (let i = 0; i < 3; i++) {
      await lobbyPages[i].voteForOption(1);
      await lobbyPages[i].expectVotedOnOption(1);
    }

    // Have next 2 users vote for option 2
    for (let i = 3; i < 5; i++) {
      await lobbyPages[i].voteForOption(2);
      await lobbyPages[i].expectVotedOnOption(2);
    }

    // Reload a user's page to see all voters
    await users[5].page.reload();
    await expect(users[5].page.locator('text=Vote for Team Composition')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Option 1 should show first 3 users' names
    const option1Card = users[5].page.locator('[data-testid="match-option-1"]');
    for (let i = 0; i < 3; i++) {
      await expect(option1Card.locator(`text=${users[i].user.displayName}`)).toBeVisible();
    }

    // Option 2 should show users 3 and 4's names
    const option2Card = users[5].page.locator('[data-testid="match-option-2"]');
    for (let i = 3; i < 5; i++) {
      await expect(option2Card.locator(`text=${users[i].user.displayName}`)).toBeVisible();
    }
  });

  test('voter names update when votes change', async ({ lobbyWithUsers }) => {
    test.setTimeout(120000);

    // Create 10 users
    const { users } = await lobbyWithUsers(10);
    const lobby = await setupVotingLobbyWith10Players(users, 'majority');

    // Navigate first 2 users to the lobby page
    await users[0].page.goto(`/lobby/${lobby.id}`);
    await users[1].page.goto(`/lobby/${lobby.id}`);
    const lobbyPage0 = new LobbyRoomPage(users[0].page);
    const lobbyPage1 = new LobbyRoomPage(users[1].page);

    // Wait for pages to load
    await expect(users[0].page.locator('text=10-Man Lobby')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await expect(users[1].page.locator('text=10-Man Lobby')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Wait for match options to load
    await lobbyPage0.waitForMatchOptions();
    await lobbyPage1.waitForMatchOptions();

    // User 0 votes for option 1
    await lobbyPage0.voteForOption(1);

    // Reload user 1's page to see voter
    await users[1].page.reload();
    await expect(users[1].page.locator('text=Vote for Team Composition')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // User 0's name should be visible on option 1
    const option1Card = users[1].page.locator('[data-testid="match-option-1"]');
    await expect(option1Card.locator(`text=${users[0].user.displayName}`)).toBeVisible();

    // User 0 changes vote to option 2
    await users[0].page.reload();
    await expect(users[0].page.locator('text=Vote for Team Composition')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await lobbyPage0.voteForOption(2);

    // Reload user 1's page to see updated voters
    await users[1].page.reload();
    await expect(users[1].page.locator('text=Vote for Team Composition')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // User 0's name should now be on option 2, not option 1
    const option1CardAfter = users[1].page.locator('[data-testid="match-option-1"]');
    const option2Card = users[1].page.locator('[data-testid="match-option-2"]');

    await expect(option1CardAfter.locator(`text=${users[0].user.displayName}`)).not.toBeVisible();
    await expect(option2Card.locator(`text=${users[0].user.displayName}`)).toBeVisible();
  });
});

test.describe('Lobby Voting API Integration', () => {
  test('voting status is fetched correctly via API', async ({ createUsers }) => {
    test.setTimeout(60000);

    // Create 10 users via fixture
    const users = await createUsers(10);

    // Create lobby with voting enabled
    const lobby = await createVotingLobby(users[0].token, 'majority');

    // All users join
    await Promise.all(
      users.slice(1).map((user) => apiCall(`/lobbies/${lobby.id}/join`, user.token))
    );

    // Ready up and generate teams
    await Promise.all(
      users.map((user) =>
        apiCall(`/lobbies/${lobby.id}/ready`, user.token, { body: { ready: true } })
      )
    );
    await apiCall(`/lobbies/${lobby.id}/generate-teams`, users[0].token);

    // Cast some votes via API
    await apiCall(`/lobbies/${lobby.id}/vote`, users[0].token, { body: { optionNumber: 1 } });
    await apiCall(`/lobbies/${lobby.id}/vote`, users[1].token, { body: { optionNumber: 1 } });
    await apiCall(`/lobbies/${lobby.id}/vote`, users[2].token, { body: { optionNumber: 2 } });

    // Fetch voting status
    const statusResponse = await fetch(
      `http://localhost:9999/api/v1/lobbies/${lobby.id}/voting-status`,
      {
        method: 'GET',
        headers: {
          Authorization: `Bearer ${users[0].token}`,
        },
      }
    );

    expect(statusResponse.ok).toBe(true);
    const status = await statusResponse.json();

    expect(status.votesCast).toBe(3);
    expect(status.totalPlayers).toBe(10);
    expect(status.voteCounts['1']).toBe(2);
    expect(status.voteCounts['2']).toBe(1);
    expect(status.userVote).toBe(1); // User 0 voted for option 1

    // Verify voters are returned
    expect(status.voters).toBeDefined();
    expect(status.voters['1']).toHaveLength(2);
    expect(status.voters['2']).toHaveLength(1);

    // Verify voter info contains display names
    const option1Voters = status.voters['1'].map((v: { displayName: string }) => v.displayName);
    expect(option1Voters).toContain(users[0].user.displayName);
    expect(option1Voters).toContain(users[1].user.displayName);

    const option2Voters = status.voters['2'].map((v: { displayName: string }) => v.displayName);
    expect(option2Voters).toContain(users[2].user.displayName);
  });

  test('cannot vote when voting is not enabled', async ({ createUsers }) => {
    test.setTimeout(60000);

    // Create users
    const users = await createUsers(2);

    // Create lobby WITHOUT voting
    const createResponse = await apiCall('/lobbies', users[0].token, {
      body: {
        draftMode: 'pro_play',
        timerDurationSeconds: 90,
        votingEnabled: false,
      },
    });
    const lobby = await createResponse.json();

    // Try to vote - should fail
    const voteResponse = await fetch(
      `http://localhost:9999/api/v1/lobbies/${lobby.id}/vote`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${users[0].token}`,
        },
        body: JSON.stringify({ optionNumber: 1 }),
      }
    );

    expect(voteResponse.ok).toBe(false);
    expect(voteResponse.status).toBe(400);
  });
});
