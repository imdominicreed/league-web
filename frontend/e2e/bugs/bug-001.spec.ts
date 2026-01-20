import { test, expect } from '@playwright/test';
import { createTestUser, setAuthToken } from '../helpers/test-utils';

test.describe('BUG-001: No Logout Button on Home Page', () => {
  test('should show logout button when authenticated', async ({ page }) => {
    // Create a test user
    const user = await createTestUser(page, 'bug001_logout');

    // Set auth token to simulate logged-in state
    await setAuthToken(page, user);

    // Navigate to home page
    await page.goto('/');

    // Verify the welcome message is shown (confirms we're authenticated)
    await expect(page.getByTestId('home-welcome-message')).toBeVisible();
    await expect(page.getByTestId('home-welcome-message')).toContainText(user.displayName);

    // Verify the logout button is visible
    const logoutButton = page.getByTestId('home-logout-button');
    await expect(logoutButton).toBeVisible();
    await expect(logoutButton).toHaveText('Logout');
  });

  test('should log out user when logout button is clicked', async ({ page }) => {
    // Create a test user
    const user = await createTestUser(page, 'bug001_logoutclick');

    // Set auth token to simulate logged-in state
    await setAuthToken(page, user);

    // Navigate to home page
    await page.goto('/');

    // Verify we're logged in
    await expect(page.getByTestId('home-welcome-message')).toBeVisible();
    const logoutButton = page.getByTestId('home-logout-button');
    await expect(logoutButton).toBeVisible();

    // Click the logout button
    await logoutButton.click();

    // Verify we're logged out - login and register buttons should appear
    await expect(page.getByTestId('home-link-login')).toBeVisible({ timeout: 5000 });
    await expect(page.getByTestId('home-link-register')).toBeVisible();

    // The welcome message should no longer be visible
    await expect(page.getByTestId('home-welcome-message')).not.toBeVisible();

    // The logout button should no longer be visible
    await expect(page.getByTestId('home-logout-button')).not.toBeVisible();
  });

  test('should clear localStorage when logging out', async ({ page }) => {
    // Create a test user
    const user = await createTestUser(page, 'bug001_storage');

    // Set auth token to simulate logged-in state
    await setAuthToken(page, user);

    // Navigate to home page
    await page.goto('/');

    // Verify we're logged in
    await expect(page.getByTestId('home-welcome-message')).toBeVisible();

    // Verify the token is in localStorage before logout
    const tokenBefore = await page.evaluate(() => localStorage.getItem('accessToken'));
    expect(tokenBefore).toBeTruthy();

    // Click the logout button
    await page.getByTestId('home-logout-button').click();

    // Verify the token is removed from localStorage after logout
    const tokenAfter = await page.evaluate(() => localStorage.getItem('accessToken'));
    expect(tokenAfter).toBeNull();
  });
});
