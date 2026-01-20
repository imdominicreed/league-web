import { test, expect } from '@playwright/test';
import {
  createTestUser,
  createLobby,
  joinLobby,
  initializeRoleProfiles,
  getLobby,
  generateTeams,
  selectMatchOption,
  promoteCaptain,
  promoteCaptainRaw,
} from '../helpers/test-utils';

test.describe('BUG-010: Promote Captain Fails After Team Selection', () => {
  test('captain should be able to promote teammate after team selection', async ({ page }) => {
    // Create 10 users for a full lobby
    const users = [];
    for (let i = 0; i < 10; i++) {
      const user = await createTestUser(page, `bug010_${i}`);
      await initializeRoleProfiles(page, user.token);
      users.push(user);
    }

    // First user creates the lobby - they become blue team captain
    const creator = users[0];
    const lobby = await createLobby(page, creator.token);
    const lobbyId = lobby.id;

    // All other users join the lobby
    for (let i = 1; i < 10; i++) {
      await joinLobby(page, users[i].token, lobbyId);
    }

    // Generate teams (this puts lobby in 'matchmaking' status)
    // Creator is the blue team captain, so use their token
    const generatedOptions = await generateTeams(page, creator.token, lobbyId);
    console.log('Generated options count:', generatedOptions.length);
    console.log('First option number:', generatedOptions[0]?.optionNumber);

    // Select a match option (this puts lobby in 'team_selected' status)
    // Only a captain can select - creator is still a captain
    // Use the first option's actual number (might not be 0)
    const firstOption = generatedOptions[0];
    expect(firstOption).toBeDefined();
    await selectMatchOption(page, creator.token, lobbyId, firstOption.optionNumber);

    // Get lobby data to find the captain and a teammate
    const lobbyData = await getLobby(page, creator.token, lobbyId);

    // Verify lobby is in team_selected status
    expect(lobbyData.status).toBe('team_selected');

    // After team selection, captains are reset based on join order
    // Find a captain (the first captain we find)
    const captainPlayer = lobbyData.players.find(
      (p: { isCaptain: boolean }) => p.isCaptain
    );
    expect(captainPlayer).toBeDefined();

    // Find the captain's user object
    const captain = users.find(u => u.id === captainPlayer.userId)!;
    expect(captain).toBeDefined();

    // Find a non-captain teammate on the same team
    const teammatePlayer = lobbyData.players.find(
      (p: { userId: string; team: string; isCaptain: boolean }) =>
        p.team === captainPlayer.team &&
        !p.isCaptain &&
        p.userId !== captain.id
    );
    expect(teammatePlayer).toBeDefined();

    // BUG-010: This should now succeed (previously failed with HTTP 500)
    const result = await promoteCaptain(page, captain.token, lobbyId, teammatePlayer.userId);

    // Verify the promotion succeeded
    expect(result).toBeDefined();

    // Verify the new captain status
    const updatedLobby = await getLobby(page, captain.token, lobbyId);
    const newCaptain = updatedLobby.players.find(
      (p: { userId: string }) => p.userId === teammatePlayer.userId
    );
    const formerCaptain = updatedLobby.players.find(
      (p: { userId: string }) => p.userId === captain.id
    );

    expect(newCaptain.isCaptain).toBe(true);
    expect(formerCaptain.isCaptain).toBe(false);
  });

  test('captain should still be able to promote in waiting_for_players status', async ({ page }) => {
    // Create 6 users (enough to have teammates)
    const users = [];
    for (let i = 0; i < 6; i++) {
      const user = await createTestUser(page, `bug010b_${i}`);
      await initializeRoleProfiles(page, user.token);
      users.push(user);
    }

    // First user creates the lobby - they become blue team captain
    const creator = users[0];
    const lobby = await createLobby(page, creator.token);
    const lobbyId = lobby.id;

    // Other users join the lobby
    for (let i = 1; i < 6; i++) {
      await joinLobby(page, users[i].token, lobbyId);
    }

    // Get lobby data
    const lobbyData = await getLobby(page, creator.token, lobbyId);

    // Verify lobby is still in waiting_for_players status
    expect(lobbyData.status).toBe('waiting_for_players');

    // Creator is the blue team captain
    const captainPlayer = lobbyData.players.find(
      (p: { userId: string }) => p.userId === creator.id
    );
    expect(captainPlayer).toBeDefined();
    expect(captainPlayer.isCaptain).toBe(true);

    // Find a non-captain teammate on the same team (blue) as creator
    const teammatePlayer = lobbyData.players.find(
      (p: { userId: string; team: string; isCaptain: boolean }) =>
        p.team === captainPlayer.team &&
        !p.isCaptain &&
        p.userId !== creator.id
    );
    expect(teammatePlayer).toBeDefined();

    // Promote should succeed
    const result = await promoteCaptain(page, creator.token, lobbyId, teammatePlayer.userId);
    expect(result).toBeDefined();

    // Verify the promotion
    const updatedLobby = await getLobby(page, creator.token, lobbyId);
    const newCaptain = updatedLobby.players.find(
      (p: { userId: string }) => p.userId === teammatePlayer.userId
    );
    expect(newCaptain.isCaptain).toBe(true);
  });

  test('captain should be able to promote in matchmaking status', async ({ page }) => {
    // Create 10 users for a full lobby
    const users = [];
    for (let i = 0; i < 10; i++) {
      const user = await createTestUser(page, `bug010c_${i}`);
      await initializeRoleProfiles(page, user.token);
      users.push(user);
    }

    // First user creates the lobby - they become blue team captain
    const creator = users[0];
    const lobby = await createLobby(page, creator.token);
    const lobbyId = lobby.id;

    // All other users join the lobby
    for (let i = 1; i < 10; i++) {
      await joinLobby(page, users[i].token, lobbyId);
    }

    // Generate teams (this puts lobby in 'matchmaking' status)
    // Creator is the blue team captain, so use their token
    await generateTeams(page, creator.token, lobbyId);

    // Get lobby data
    const lobbyData = await getLobby(page, creator.token, lobbyId);

    // Verify lobby is in matchmaking status
    expect(lobbyData.status).toBe('matchmaking');

    // Creator is still blue team captain after generate teams
    const captainPlayer = lobbyData.players.find(
      (p: { userId: string }) => p.userId === creator.id
    );
    expect(captainPlayer).toBeDefined();
    expect(captainPlayer.isCaptain).toBe(true);

    // Find a non-captain teammate on the same team as creator (blue)
    const teammatePlayer = lobbyData.players.find(
      (p: { userId: string; team: string; isCaptain: boolean }) =>
        p.team === captainPlayer.team &&
        !p.isCaptain &&
        p.userId !== creator.id
    );
    expect(teammatePlayer).toBeDefined();

    // Promote should succeed in matchmaking status
    const result = await promoteCaptain(page, creator.token, lobbyId, teammatePlayer.userId);
    expect(result).toBeDefined();

    // Verify the promotion
    const updatedLobby = await getLobby(page, creator.token, lobbyId);
    const newCaptain = updatedLobby.players.find(
      (p: { userId: string }) => p.userId === teammatePlayer.userId
    );
    expect(newCaptain.isCaptain).toBe(true);
  });
});
