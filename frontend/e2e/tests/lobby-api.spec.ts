import {
  test,
  expect,
  setUserReady,
  generateTeams,
  selectMatchOption,
  getLobby,
} from '../fixtures/multi-user';

// API-only integration tests - these test backend functionality without UI interaction
test.describe('Lobby API Integration', () => {
  test('10 users can join a lobby and complete team generation', async ({ lobbyWithUsers }) => {
    // Step 1: Create 10 users and a lobby
    const { lobby, users } = await lobbyWithUsers(10);

    expect(lobby.players).toHaveLength(10);
    expect(lobby.status).toBe('waiting_for_players');

    // Step 2: All users ready up
    const readyPromises = users.map((user) => setUserReady(user, lobby.id, true));
    await Promise.all(readyPromises);

    // Verify all users are ready
    const lobbyAfterReady = await getLobby(users[0], lobby.id);
    const allReady = lobbyAfterReady.players.every((p) => p.isReady);
    expect(allReady).toBe(true);

    // Step 3: Creator generates teams
    const matchOptions = await generateTeams(users[0], lobby.id);
    expect(matchOptions.length).toBeGreaterThan(0);

    // Verify lobby status changed to matchmaking
    const lobbyAfterGenerate = await getLobby(users[0], lobby.id);
    expect(lobbyAfterGenerate.status).toBe('matchmaking');

    // Step 4: Creator selects an option
    const selectedOptionNumber = matchOptions[0].optionNumber;
    const finalLobby = await selectMatchOption(users[0], lobby.id, selectedOptionNumber);

    // Step 5: Verify lobby state
    expect(finalLobby.status).toBe('team_selected');
    expect(finalLobby.selectedMatchOption).toBe(selectedOptionNumber);

    // Verify all players have team and role assignments
    for (const player of finalLobby.players) {
      expect(player.team).toBeTruthy();
      expect(player.assignedRole).toBeTruthy();
    }

    // Verify we have 5 players on each team
    const blueTeam = finalLobby.players.filter((p) => p.team === 'blue');
    const redTeam = finalLobby.players.filter((p) => p.team === 'red');
    expect(blueTeam).toHaveLength(5);
    expect(redTeam).toHaveLength(5);
  });

  test('users can join lobby with different user counts', async ({ lobbyWithUsers }) => {
    // Test with minimum viable player count (2)
    const { lobby, users } = await lobbyWithUsers(2);

    expect(lobby.players).toHaveLength(2);
    expect(users).toHaveLength(2);

    // Verify both users are in the lobby
    const playerUserIds = lobby.players.map((p) => p.userId);
    for (const user of users) {
      expect(playerUserIds).toContain(user.user.id);
    }
  });

  test('lobby creator is correctly identified', async ({ lobbyWithUsers }) => {
    const { lobby, users } = await lobbyWithUsers(5);

    // First user should be the creator
    expect(lobby.createdBy).toBe(users[0].user.id);
  });

  test('ready status can be toggled', async ({ lobbyWithUsers }) => {
    const { lobby, users } = await lobbyWithUsers(3);
    const testUser = users[1];

    // Ready up
    await setUserReady(testUser, lobby.id, true);
    let updatedLobby = await getLobby(users[0], lobby.id);
    let player = updatedLobby.players.find((p) => p.userId === testUser.user.id);
    expect(player?.isReady).toBe(true);

    // Un-ready
    await setUserReady(testUser, lobby.id, false);
    updatedLobby = await getLobby(users[0], lobby.id);
    player = updatedLobby.players.find((p) => p.userId === testUser.user.id);
    expect(player?.isReady).toBe(false);
  });
});

test.describe('Lobby API Error Handling', () => {
  test('cannot generate teams without enough players', async ({ createUsers }) => {
    const users = await createUsers(2);

    // Create lobby with just 2 users
    const response = await fetch('http://localhost:9999/api/v1/lobbies', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${users[0].token}`,
      },
      body: JSON.stringify({
        draftMode: 'pro_play',
        timerDurationSeconds: 30,
      }),
    });

    const lobby = await response.json();

    // Second user joins
    await fetch(`http://localhost:9999/api/v1/lobbies/${lobby.id}/join`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${users[1].token}`,
      },
    });

    // Both ready up
    await setUserReady(users[0], lobby.id, true);
    await setUserReady(users[1], lobby.id, true);

    // Try to generate teams - should fail with only 2 players
    const generateResponse = await fetch(`http://localhost:9999/api/v1/lobbies/${lobby.id}/generate-teams`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${users[0].token}`,
      },
    });

    // Expect failure since we need at least 10 players for team generation
    expect(generateResponse.status).not.toBe(200);
  });
});
