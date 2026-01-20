import { test, expect } from '@playwright/test';
import { createTestUser, setAuthToken } from '../helpers/test-utils';

test.describe('BUG-002: Login Page Accessible When Authenticated', () => {
  test('should redirect authenticated user from /login to home', async ({ page }) => {
    // Create a test user and get token
    const user = await createTestUser(page, 'bug002_login');

    // Set auth token to simulate logged-in state
    await setAuthToken(page, user);

    // Navigate directly to login page
    await page.goto('/login');

    // Should be redirected to home page
    await page.waitForURL('/', { timeout: 5000 });

    // Verify we're on the home page
    await expect(page.locator('text=League Draft')).toBeVisible({ timeout: 5000 });
  });

  test('should redirect authenticated user from /register to home', async ({ page }) => {
    // Create a test user and get token
    const user = await createTestUser(page, 'bug002_register');

    // Set auth token to simulate logged-in state
    await setAuthToken(page, user);

    // Navigate directly to register page
    await page.goto('/register');

    // Should be redirected to home page
    await page.waitForURL('/', { timeout: 5000 });

    // Verify we're on the home page
    await expect(page.locator('text=League Draft')).toBeVisible({ timeout: 5000 });
  });

  test('should allow unauthenticated user to access /login', async ({ page }) => {
    // Clear any existing auth tokens
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.removeItem('accessToken');
    });

    // Navigate to login page
    await page.goto('/login');

    // Should stay on login page
    await expect(page).toHaveURL('/login');
    await expect(page.locator('h1:has-text("Login")')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('[data-testid="login-button-submit"]')).toBeVisible();
  });

  test('should allow unauthenticated user to access /register', async ({ page }) => {
    // Clear any existing auth tokens
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.removeItem('accessToken');
    });

    // Navigate to register page
    await page.goto('/register');

    // Should stay on register page
    await expect(page).toHaveURL('/register');
    await expect(page.locator('h1:has-text("Register")')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('[data-testid="register-button-submit"]')).toBeVisible();
  });
});
