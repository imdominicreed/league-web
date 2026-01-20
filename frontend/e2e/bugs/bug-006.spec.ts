import { test, expect } from '@playwright/test';
import {
  createTestUser,
  setAuthToken,
  createLobbyWithVoting,
  joinLobby,
  initializeRoleProfiles,
  generateTeams,
  getVotingStatus,
} from '../helpers/test-utils';

const API_BASE = 'http://localhost:9999/api/v1';

test.describe('BUG-006: Vote Button Click Does Not Trigger Vote Action', () => {
  test('clicking Vote button should register a vote', async ({ page }) => {
    // Create 10 test users
    const users = [];
    for (let i = 1; i <= 10; i++) {
      const user = await createTestUser(page, `bug006_${i}`);
      await initializeRoleProfiles(page, user.token);
      users.push(user);
    }

    // User 1 creates lobby with voting enabled
    const lobby = await createLobbyWithVoting(page, users[0].token);
    const lobbyId = lobby.id;

    // All 10 users join the lobby
    for (let i = 1; i < 10; i++) {
      await joinLobby(page, users[i].token, lobbyId);
    }

    // Generate teams (user 1 is the creator/captain)
    await generateTeams(page, users[0].token, lobbyId);

    // Set auth token for user 1 and navigate to lobby
    await setAuthToken(page, users[0]);
    await page.goto(`/lobby/${lobbyId}`);

    // Wait for match options to appear
    await page.waitForSelector('[data-testid="match-option-1"]', { timeout: 15000 });

    // Get initial voting status
    const initialStatus = await getVotingStatus(page, users[0].token, lobbyId);
    const initialVotes = initialStatus.userVotes || [];

    // Find the Vote button on Option 1 and click it directly
    const option1Card = page.locator('[data-testid="match-option-1"]');
    const voteButton = option1Card.locator('button:has-text("Vote for This")');

    // Verify the button exists and is visible
    await expect(voteButton).toBeVisible();

    // Click the button directly (this was the bug - button click didn't work)
    await voteButton.click();

    // Wait for the button text to change to "Voted" indicating vote was registered
    await expect(option1Card.locator('button:has-text("Voted")')).toBeVisible({ timeout: 5000 });

    // Verify via API that the vote was registered
    const afterStatus = await getVotingStatus(page, users[0].token, lobbyId);
    expect(afterStatus.userVotes).toContain(1);

    // Verify vote count increased
    expect(afterStatus.voteCounts[1]).toBeGreaterThan(0);
  });

  test('clicking Vote button on different options should toggle votes', async ({ page }) => {
    // Create 10 test users
    const users = [];
    for (let i = 1; i <= 10; i++) {
      const user = await createTestUser(page, `bug006b_${i}`);
      await initializeRoleProfiles(page, user.token);
      users.push(user);
    }

    // User 1 creates lobby with voting enabled
    const lobby = await createLobbyWithVoting(page, users[0].token);
    const lobbyId = lobby.id;

    // All 10 users join the lobby
    for (let i = 1; i < 10; i++) {
      await joinLobby(page, users[i].token, lobbyId);
    }

    // Generate teams
    await generateTeams(page, users[0].token, lobbyId);

    // Set auth token for user 1 and navigate to lobby
    await setAuthToken(page, users[0]);
    await page.goto(`/lobby/${lobbyId}`);

    // Wait for match options to appear
    await page.waitForSelector('[data-testid="match-option-1"]', { timeout: 15000 });
    await page.waitForSelector('[data-testid="match-option-2"]', { timeout: 5000 });

    // Vote for Option 1 by clicking the button
    const option1Card = page.locator('[data-testid="match-option-1"]');
    const voteButton1 = option1Card.locator('button:has-text("Vote for This")');
    await voteButton1.click();

    // Wait for vote to register
    await expect(option1Card.locator('button:has-text("Voted")')).toBeVisible({ timeout: 5000 });

    // Now click Vote button on Option 2
    const option2Card = page.locator('[data-testid="match-option-2"]');
    const voteButton2 = option2Card.locator('button:has-text("Vote for This")');
    await voteButton2.click();

    // Wait for vote to register on option 2
    await expect(option2Card.locator('button:has-text("Voted")')).toBeVisible({ timeout: 5000 });

    // Verify via API that both votes are registered (multi-vote is allowed)
    const status = await getVotingStatus(page, users[0].token, lobbyId);
    expect(status.userVotes).toContain(1);
    expect(status.userVotes).toContain(2);
  });

  test('clicking Voted button should remove the vote', async ({ page }) => {
    // Create 10 test users
    const users = [];
    for (let i = 1; i <= 10; i++) {
      const user = await createTestUser(page, `bug006c_${i}`);
      await initializeRoleProfiles(page, user.token);
      users.push(user);
    }

    // User 1 creates lobby with voting enabled
    const lobby = await createLobbyWithVoting(page, users[0].token);
    const lobbyId = lobby.id;

    // All 10 users join the lobby
    for (let i = 1; i < 10; i++) {
      await joinLobby(page, users[i].token, lobbyId);
    }

    // Generate teams
    await generateTeams(page, users[0].token, lobbyId);

    // Set auth token for user 1 and navigate to lobby
    await setAuthToken(page, users[0]);
    await page.goto(`/lobby/${lobbyId}`);

    // Wait for match options to appear
    await page.waitForSelector('[data-testid="match-option-1"]', { timeout: 15000 });

    // Vote for Option 1 by clicking the button
    const option1Card = page.locator('[data-testid="match-option-1"]');
    const voteButton = option1Card.locator('button:has-text("Vote for This")');
    await voteButton.click();

    // Wait for vote to register
    const votedButton = option1Card.locator('button:has-text("Voted")');
    await expect(votedButton).toBeVisible({ timeout: 5000 });

    // Click the "Voted (Click to Remove)" button to unvote
    await votedButton.click();

    // Wait for button to change back to "Vote for This"
    await expect(option1Card.locator('button:has-text("Vote for This")')).toBeVisible({ timeout: 5000 });

    // Verify via API that the vote was removed
    const status = await getVotingStatus(page, users[0].token, lobbyId);
    // userVotes may be undefined/null when no votes, or empty array, or array without option 1
    expect(status.userVotes || []).not.toContain(1);
  });
});
