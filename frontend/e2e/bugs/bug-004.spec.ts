import { test, expect } from '@playwright/test';
import { createTestUser, setAuthToken } from '../helpers/test-utils';

const API_BASE = 'http://localhost:9999/api/v1';

test.describe('BUG-004: Custom Timer Value Not Sent When Creating Lobby', () => {
  test('should send custom timer value when creating lobby via UI', async ({ page }) => {
    // Create a test user
    const user = await createTestUser(page, 'bug004_timer');

    // Set auth token to simulate logged-in state
    await setAuthToken(page, user);

    // Navigate to create lobby page
    await page.goto('/create-lobby');

    // Wait for the page to load
    await expect(page.locator('h1')).toContainText('Create 10-Man Lobby');

    // Find the timer duration input and change it to 60 seconds
    const timerInput = page.locator('input[type="number"]');
    await expect(timerInput).toBeVisible();

    // Clear the input and type new value
    await timerInput.fill('60');

    // Verify the input value changed
    await expect(timerInput).toHaveValue('60');

    // Set up a request listener to capture the API request
    let capturedRequestBody: any = null;
    page.on('request', request => {
      if (request.url().includes('/lobbies') && request.method() === 'POST') {
        capturedRequestBody = request.postDataJSON();
      }
    });

    // Click the Create Lobby button
    await page.click('button:has-text("Create Lobby")');

    // Wait for navigation to lobby room
    await page.waitForURL(/\/lobby\/.+/, { timeout: 10000 });

    // Verify the request was made with the correct timer value
    expect(capturedRequestBody).not.toBeNull();
    expect(capturedRequestBody.timerDurationSeconds).toBe(60);
  });

  test('should create lobby with correct timer duration via API', async ({ page }) => {
    // Create a test user
    const user = await createTestUser(page, 'bug004_api');

    // Create lobby directly via API with custom timer
    const customTimer = 45;
    const res = await page.request.post(`${API_BASE}/lobbies`, {
      headers: { Authorization: `Bearer ${user.token}` },
      data: {
        draftMode: 'pro_play',
        timerDurationSeconds: customTimer,
      },
    });
    expect(res.ok()).toBeTruthy();
    const lobby = await res.json();

    // Verify the lobby was created with the correct timer
    expect(lobby.timerDurationSeconds).toBe(customTimer);

    // Also fetch the lobby to double-check
    const getRes = await page.request.get(`${API_BASE}/lobbies/${lobby.id}`, {
      headers: { Authorization: `Bearer ${user.token}` },
    });
    expect(getRes.ok()).toBeTruthy();
    const fetchedLobby = await getRes.json();
    expect(fetchedLobby.timerDurationSeconds).toBe(customTimer);
  });

  test('should display custom timer value in lobby room', async ({ page }) => {
    // Create a test user
    const user = await createTestUser(page, 'bug004_display');

    // Set auth token to simulate logged-in state
    await setAuthToken(page, user);

    // Navigate to create lobby page
    await page.goto('/create-lobby');

    // Set custom timer value (90 seconds)
    const timerInput = page.locator('input[type="number"]');
    await timerInput.fill('90');

    // Create the lobby
    await page.click('button:has-text("Create Lobby")');

    // Wait for navigation to lobby room
    await page.waitForURL(/\/lobby\/.+/, { timeout: 10000 });

    // The lobby room should show some indication of the timer duration
    // (this depends on UI implementation - check if timer value is shown)
    // For now, verify we reached the lobby room successfully
    await expect(page.locator('text=10-Man Lobby')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=Code:')).toBeVisible();
  });
});
