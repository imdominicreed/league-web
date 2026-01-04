import { test, expect } from '@playwright/test';
import {
  HomePage,
  CreateLobbyPage,
  LobbyRoomPage,
  generateTestUsername,
  registerUserViaApi,
  setAuthToken,
  createLobbyViaApi,
} from '../fixtures/pages';

test.describe('Lobby UI - Single User', () => {
  test('user can create a lobby through the UI', async ({ page }) => {
    const homePage = new HomePage(page);
    const createLobbyPage = new CreateLobbyPage(page);

    // Authenticate
    const username = generateTestUsername('lobby');
    const { token } = await registerUserViaApi(username, 'testpassword123');

    await homePage.goto();
    await setAuthToken(page, token);

    // Navigate to create lobby
    await homePage.clickCreateLobby();

    // Create lobby with default settings
    await createLobbyPage.createLobby('pro_play', 30);

    // Should be redirected to lobby room
    await page.waitForURL(/\/lobby\//);
    await expect(page.locator('text=10-Man Lobby')).toBeVisible();
  });

  test('lobby displays short code', async ({ page }) => {
    // Authenticate
    const username = generateTestUsername('lobby');
    const { token } = await registerUserViaApi(username, 'testpassword123');

    // Create lobby via API
    const lobby = await createLobbyViaApi(token);

    // Navigate to lobby
    await page.goto('/');
    await setAuthToken(page, token);

    const lobbyPage = new LobbyRoomPage(page);
    await lobbyPage.goto(lobby.id);

    // Should show short code
    await lobbyPage.expectLobbyCode(lobby.shortCode);
  });

  test('user can toggle ready status', async ({ page }) => {
    // Authenticate
    const username = generateTestUsername('lobby');
    const { token } = await registerUserViaApi(username, 'testpassword123');

    // Create lobby via API
    const lobby = await createLobbyViaApi(token);

    // Navigate to lobby
    await page.goto('/');
    await setAuthToken(page, token);

    const lobbyPage = new LobbyRoomPage(page);
    await lobbyPage.goto(lobby.id);

    // Should see Ready Up button
    await lobbyPage.expectReadyButton();

    // Click Ready Up
    await lobbyPage.clickReadyUp();

    // Should now see Cancel Ready button
    await lobbyPage.expectCancelReadyButton();

    // Click Cancel Ready
    await lobbyPage.clickCancelReady();

    // Should be back to Ready Up
    await lobbyPage.expectReadyButton();
  });

  test('shows player count', async ({ page }) => {
    // Authenticate
    const username = generateTestUsername('lobby');
    const { token } = await registerUserViaApi(username, 'testpassword123');

    // Create lobby via API
    const lobby = await createLobbyViaApi(token);

    // Navigate to lobby
    await page.goto('/');
    await setAuthToken(page, token);

    const lobbyPage = new LobbyRoomPage(page);
    await lobbyPage.goto(lobby.id);

    // Should show 1/10 players (just the creator)
    await lobbyPage.expectPlayerCount(1, 10);
  });

  test('user can leave lobby', async ({ page }) => {
    // Authenticate
    const username = generateTestUsername('lobby');
    const { token } = await registerUserViaApi(username, 'testpassword123');

    // Create lobby via API
    const lobby = await createLobbyViaApi(token);

    // Navigate to lobby
    await page.goto('/');
    await setAuthToken(page, token);

    const lobbyPage = new LobbyRoomPage(page);
    await lobbyPage.goto(lobby.id);

    // Leave the lobby
    await lobbyPage.leave();

    // Should be back at home
    expect(page.url()).toMatch(/\/$/);
  });

  test('shows user name in player list', async ({ page }) => {
    // Authenticate
    const username = generateTestUsername('lobby');
    const { token } = await registerUserViaApi(username, 'testpassword123');

    // Create lobby via API
    const lobby = await createLobbyViaApi(token);

    // Navigate to lobby
    await page.goto('/');
    await setAuthToken(page, token);

    const lobbyPage = new LobbyRoomPage(page);
    await lobbyPage.goto(lobby.id);

    // Should show the user's name in the player grid
    await expect(page.locator(`text=${username}`)).toBeVisible();

    // Should show "(You)" indicator for current user
    await expect(page.locator('text=(You)')).toBeVisible();
  });

  test('create lobby page has back link', async ({ page }) => {
    // Authenticate
    const username = generateTestUsername('lobby');
    const { token } = await registerUserViaApi(username, 'testpassword123');

    await page.goto('/');
    await setAuthToken(page, token);

    const createLobbyPage = new CreateLobbyPage(page);
    await createLobbyPage.goto();

    // Click back link
    await page.click('a:has-text("Back to Home")');

    // Should be at home
    await page.waitForURL('/');
  });
});
