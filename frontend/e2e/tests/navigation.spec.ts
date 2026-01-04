import { test, expect } from '@playwright/test';
import {
  HomePage,
  generateTestUsername,
  registerUserViaApi,
  setAuthToken,
} from '../fixtures/pages';

test.describe('Navigation', () => {
  test('unauthenticated user sees login and register buttons', async ({ page }) => {
    const homePage = new HomePage(page);

    await homePage.goto();
    await homePage.expectUnauthenticated();

    // Should not see authenticated menu items
    await expect(page.locator('a:has-text("Create Draft Room")')).not.toBeVisible();
    await expect(page.locator('a:has-text("Create 10-Man Lobby")')).not.toBeVisible();
  });

  test('authenticated user sees main menu options', async ({ page }) => {
    const homePage = new HomePage(page);

    // Register and authenticate
    const username = generateTestUsername('nav');
    const password = 'testpassword123';
    const { token } = await registerUserViaApi(username, password);

    await homePage.goto();
    await setAuthToken(page, token);

    // Should see authenticated menu
    await homePage.expectAuthenticated(username);
    await homePage.expectAuthenticatedMenu();

    // Should not see login/register
    await expect(page.locator('a:has-text("Login")')).not.toBeVisible();
    await expect(page.locator('a:has-text("Register")')).not.toBeVisible();
  });

  test('can navigate to create draft room page', async ({ page }) => {
    const homePage = new HomePage(page);

    // Authenticate
    const username = generateTestUsername('nav');
    const { token } = await registerUserViaApi(username, 'testpassword123');

    await homePage.goto();
    await setAuthToken(page, token);

    // Navigate to create draft room
    await homePage.clickCreateDraftRoom();

    // Should be on create page
    expect(page.url()).toContain('/create');
    await expect(page.locator('text=Create Draft Room')).toBeVisible();
  });

  test('can navigate to join room page', async ({ page }) => {
    const homePage = new HomePage(page);

    // Authenticate
    const username = generateTestUsername('nav');
    const { token } = await registerUserViaApi(username, 'testpassword123');

    await homePage.goto();
    await setAuthToken(page, token);

    // Navigate to join room
    await homePage.clickJoinRoom();

    // Should be on join page
    expect(page.url()).toContain('/join');
  });

  test('can navigate to profile page', async ({ page }) => {
    const homePage = new HomePage(page);

    // Authenticate
    const username = generateTestUsername('nav');
    const { token } = await registerUserViaApi(username, 'testpassword123');

    await homePage.goto();
    await setAuthToken(page, token);

    // Navigate to profile
    await homePage.clickMyProfile();

    // Should be on profile page
    expect(page.url()).toContain('/profile');
  });

  test('can navigate to create lobby page', async ({ page }) => {
    const homePage = new HomePage(page);

    // Authenticate
    const username = generateTestUsername('nav');
    const { token } = await registerUserViaApi(username, 'testpassword123');

    await homePage.goto();
    await setAuthToken(page, token);

    // Navigate to create lobby
    await homePage.clickCreateLobby();

    // Should be on create lobby page
    expect(page.url()).toContain('/create-lobby');
    await expect(page.locator('text=Create 10-Man Lobby')).toBeVisible();
  });

  test('unauthenticated user cannot access protected routes', async ({ page }) => {
    // Try to access create lobby page directly
    await page.goto('/create-lobby');

    // Should be redirected to login or home
    await page.waitForURL(/\/(login)?$/);
  });
});
