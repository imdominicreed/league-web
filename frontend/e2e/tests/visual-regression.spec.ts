import { test, expect } from '@playwright/test';
import { generateTestUsername, registerUserViaApi } from '../fixtures';

const API_BASE = 'http://localhost:9999/api/v1';

/**
 * Visual Regression Tests
 *
 * These tests capture screenshots and compare them against baseline images.
 * - Run `npx playwright test --update-snapshots` to update baselines
 * - Screenshots are stored in e2e/__snapshots__
 * - Use threshold configuration in playwright.config.ts to adjust sensitivity
 *
 * Best Practices:
 * 1. Wait for all dynamic content to load before capturing
 * 2. Use consistent viewport sizes
 * 3. Avoid capturing time-sensitive data (use data masking if needed)
 * 4. Group related visual tests together
 */

async function loginViaApi(username: string, password: string): Promise<string> {
  const response = await fetch(`${API_BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ displayName: username, password }),
  });
  if (!response.ok) {
    throw new Error(`Login failed: ${response.status}`);
  }
  const data = await response.json();
  return data.accessToken;
}

test.describe('Visual Regression Tests', () => {
  test('home page - unauthenticated', async ({ page }) => {
    await page.goto('/');

    // Wait for content to stabilize
    await expect(page.locator('text=League Draft')).toBeVisible();

    // Take screenshot
    await expect(page).toHaveScreenshot('home-unauthenticated.png', {
      // Mask dynamic elements
      mask: [
        page.locator('[data-testid="dynamic-content"]'),
      ],
    });
  });

  test('login page', async ({ page }) => {
    await page.goto('/login');

    // Wait for form to be visible
    await expect(page.locator('form')).toBeVisible();

    // Take screenshot
    await expect(page).toHaveScreenshot('login-page.png');
  });

  test('register page', async ({ page }) => {
    await page.goto('/register');

    // Wait for form to be visible
    await expect(page.locator('form')).toBeVisible();

    // Take screenshot
    await expect(page).toHaveScreenshot('register-page.png');
  });

  test('home page - authenticated', async ({ page }) => {
    // Register and login
    const username = generateTestUsername('visual');
    const password = 'testpassword123';
    await registerUserViaApi(username, password);
    const token = await loginViaApi(username, password);

    // Set up auth
    await page.goto('/');
    await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
    await page.reload();

    // Wait for authenticated state
    await expect(page.locator(`text=${username}`)).toBeVisible();

    // Take screenshot (mask username to avoid test name differences)
    await expect(page).toHaveScreenshot('home-authenticated.png', {
      mask: [page.locator(`text=${username}`)],
    });
  });

  test('profile page layout', async ({ page }) => {
    // Register and login
    const username = generateTestUsername('profile');
    const password = 'testpassword123';
    await registerUserViaApi(username, password);
    const token = await loginViaApi(username, password);

    // Set up auth
    await page.goto('/');
    await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
    await page.reload();

    // Navigate to profile
    await page.goto('/profile');

    // Wait for profile content
    await expect(page.locator('text=Role Profiles')).toBeVisible();

    // Take screenshot (mask username)
    await expect(page).toHaveScreenshot('profile-page.png', {
      mask: [page.locator(`text=${username}`)],
    });
  });
});

test.describe('Component Visual Tests', () => {
  // Skipped: This test requires multi-user coordination to start the draft
  test.skip('champion grid styling', async ({ page }) => {
    // Register and login
    const username = generateTestUsername('champ');
    const password = 'testpassword123';
    await registerUserViaApi(username, password);
    const token = await loginViaApi(username, password);

    // Create a room via API
    const roomResponse = await fetch(`${API_BASE}/rooms`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ draftMode: 'pro_play', timerDurationSeconds: 30 }),
    });
    const room = await roomResponse.json();

    // Set up auth and navigate
    await page.goto('/');
    await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
    await page.reload();
    await page.goto(`/draft/${room.id}`);

    // Wait for champion grid to load
    await expect(page.locator('[data-testid="champion-grid"]')).toBeVisible();
    await expect(page.locator('[data-testid="champion-grid"] button').first()).toBeVisible();

    // Screenshot just the champion grid component
    const championGrid = page.locator('[data-testid="champion-grid"]');
    await expect(championGrid).toHaveScreenshot('champion-grid.png');
  });
});
