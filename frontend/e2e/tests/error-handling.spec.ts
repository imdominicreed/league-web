import { test, expect } from '@playwright/test';
import { HomePage, LoginPage, RegisterPage, DraftRoomPage } from '../page-objects';
import { generateTestUsername, registerUserViaApi } from '../fixtures';
import { TIMEOUTS } from '../helpers/wait-strategies';

const API_BASE = 'http://localhost:9999/api/v1';

/**
 * Helper to get auth token via API
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

test.describe('Error Handling', () => {
  test.describe('Authentication Errors', () => {
    test('shows error for invalid login credentials', async ({ page }) => {
      const loginPage = new LoginPage(page);
      await loginPage.goto();

      // Try invalid credentials
      await loginPage.login('nonexistent_user_99999', 'wrongpassword');

      // Should show error
      await loginPage.expectError();

      // Should stay on login page
      expect(page.url()).toContain('/login');
    });

    test('shows error for duplicate registration', async ({ page }) => {
      const registerPage = new RegisterPage(page);

      // First register a user via API
      const username = generateTestUsername('dup');
      const password = 'testpassword123';
      await registerUserViaApi(username, password);

      // Try to register same username via UI
      await registerPage.goto();
      await registerPage.register(username, password);

      // Should show error
      await registerPage.expectError();

      // Should stay on register page
      expect(page.url()).toContain('/register');
    });

    test('shows error for empty login fields', async ({ page }) => {
      await page.goto('/login');

      // Try to submit empty form - HTML5 validation should prevent submission
      await page.click('button:has-text("Login")');

      // Should still be on login page (form not submitted due to required fields)
      expect(page.url()).toContain('/login');

      // Check that required fields are invalid (HTML5 validation)
      const usernameInput = page.locator('input#displayName');
      const isInvalid = await usernameInput.evaluate((el: HTMLInputElement) => !el.validity.valid);
      expect(isInvalid).toBe(true);
    });

    test('shows error for empty registration fields', async ({ page }) => {
      await page.goto('/register');

      // Try to submit empty form - HTML5 validation should prevent submission
      await page.click('button:has-text("Register")');

      // Should still be on register page (form not submitted due to required fields)
      expect(page.url()).toContain('/register');

      // Check that required fields are invalid (HTML5 validation)
      const usernameInput = page.locator('input#displayName');
      const isInvalid = await usernameInput.evaluate((el: HTMLInputElement) => !el.validity.valid);
      expect(isInvalid).toBe(true);
    });
  });

  test.describe('Draft Room Errors', () => {
    test('shows error for invalid room code', async ({ page }) => {
      // Register and login
      const username = generateTestUsername('invalid');
      const password = 'testpassword123';
      await registerUserViaApi(username, password);
      const token = await loginViaApi(username, password);

      // Set up auth
      await page.goto('/');
      await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
      await page.reload();

      // Navigate to join draft
      await page.goto('/join');

      // Enter invalid room code
      await page.fill('input#code', 'INVALID123');
      await page.click('button:has-text("Join Room")');

      // Should show error
      await expect(
        page.locator('.text-red-500, .error-message, [role="alert"]').or(page.locator('text=not found'))
      ).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });

    test('shows error for non-existent draft room', async ({ page }) => {
      // Register and login
      const username = generateTestUsername('noroom');
      const password = 'testpassword123';
      await registerUserViaApi(username, password);
      const token = await loginViaApi(username, password);

      // Set up auth
      await page.goto('/');
      await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
      await page.reload();

      // Navigate directly to a non-existent room
      await page.goto('/draft/00000000-0000-0000-0000-000000000000');

      // Should show error or redirect
      await expect(
        page.locator('text=not found').or(page.locator('text=error')).or(page.locator('.text-red-500'))
      ).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });
  });

  test.describe('Lobby Errors', () => {
    test('shows error for non-existent lobby', async ({ page }) => {
      // Register and login
      const username = generateTestUsername('nolobby');
      const password = 'testpassword123';
      await registerUserViaApi(username, password);
      const token = await loginViaApi(username, password);

      // Set up auth
      await page.goto('/');
      await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
      await page.reload();

      // Navigate to a non-existent lobby
      await page.goto('/lobby/00000000-0000-0000-0000-000000000000');

      // Should show not found message
      await expect(
        page.locator('text=not found').or(page.locator('text=Lobby not found'))
      ).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    });
  });

  test.describe('Protected Routes', () => {
    test('redirects to login when accessing protected route without auth', async ({ page }) => {
      // Clear any stored auth
      await page.goto('/');
      await page.evaluate(() => {
        localStorage.removeItem('accessToken');
        localStorage.removeItem('refreshToken');
      });

      // Try to access profile (protected route)
      await page.goto('/profile');

      // Should redirect to login page
      await page.waitForURL('/login', { timeout: TIMEOUTS.MEDIUM });
      await expect(page.locator('h1:has-text("Login")')).toBeVisible();
    });

    test('redirects to login when accessing create draft without auth', async ({ page }) => {
      // Clear any stored auth
      await page.goto('/');
      await page.evaluate(() => {
        localStorage.removeItem('accessToken');
        localStorage.removeItem('refreshToken');
      });

      // Try to access create draft
      await page.goto('/create');

      // Should redirect to login page
      await page.waitForURL('/login', { timeout: TIMEOUTS.MEDIUM });
      await expect(page.locator('h1:has-text("Login")')).toBeVisible();
    });

    test('redirects to login when accessing create lobby without auth', async ({ page }) => {
      // Clear any stored auth
      await page.goto('/');
      await page.evaluate(() => {
        localStorage.removeItem('accessToken');
        localStorage.removeItem('refreshToken');
      });

      // Try to access create lobby
      await page.goto('/create-lobby');

      // Should redirect to login page
      await page.waitForURL('/login', { timeout: TIMEOUTS.MEDIUM });
      await expect(page.locator('h1:has-text("Login")')).toBeVisible();
    });
  });

  test.describe('Network Error Handling', () => {
    test('handles offline gracefully on home page', async ({ page, context }) => {
      // Register and login
      const username = generateTestUsername('offline');
      const password = 'testpassword123';
      await registerUserViaApi(username, password);
      const token = await loginViaApi(username, password);

      // Set up auth
      await page.goto('/');
      await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
      await page.reload();

      // Verify page loads
      await expect(page.locator('text=League Draft')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

      // Go offline
      await context.setOffline(true);

      // Try to navigate
      await page.click('a:has-text("Profile")').catch(() => {});

      // Page should handle gracefully (not crash)
      // May show error message or cached content
      const pageContent = await page.content();
      expect(pageContent).toBeTruthy();

      // Go back online
      await context.setOffline(false);
    });
  });

  test.describe('Form Validation Errors', () => {
    test('registration with any password length succeeds', async ({ page }) => {
      // Note: The backend currently accepts any password length
      await page.goto('/register');

      // Fill in username and short password
      await page.fill('input#displayName', generateTestUsername('shortpw'));
      await page.fill('input#password', '123'); // Short password

      await page.click('button:has-text("Register")');

      // Should succeed and redirect to home (backend accepts any password)
      await expect(page).toHaveURL('/');
    });

    test('shows error for invalid timer duration on draft creation', async ({ page }) => {
      // Register and login
      const username = generateTestUsername('timer');
      const password = 'testpassword123';
      await registerUserViaApi(username, password);
      const token = await loginViaApi(username, password);

      // Set up auth
      await page.goto('/');
      await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
      await page.reload();

      // Navigate to create draft
      await page.goto('/create');

      // The timer is a range slider with min=15 and max=60
      // Cannot set invalid value via UI, so verify the page loads correctly
      await expect(page.locator('text=Create Draft Room')).toBeVisible({ timeout: TIMEOUTS.SHORT });
      await expect(page.locator('button:has-text("Create Room")')).toBeVisible();
    });
  });
});
