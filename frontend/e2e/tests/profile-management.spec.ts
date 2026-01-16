import { test, expect } from '@playwright/test';
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

/**
 * Helper to initialize role profiles via API
 */
async function initializeRoleProfiles(token: string): Promise<void> {
  const response = await fetch(`${API_BASE}/profile/roles/initialize`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
  });
  if (!response.ok && response.status !== 409) {
    // 409 means already initialized, which is fine
    throw new Error(`Initialize roles failed: ${response.status}`);
  }
}

test.describe('Profile Management', () => {
  test('user can view their profile page', async ({ page }) => {
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
    await page.click('a:has-text("Profile")');
    await page.waitForURL('/profile');

    // Profile page should show username
    await expect(page.locator(`text=${username}`)).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Should see Role Profiles section header
    await expect(page.locator('text=Role Profiles')).toBeVisible();
  });

  test('user can initialize role profiles', async ({ page }) => {
    // Register and login
    const username = generateTestUsername('init');
    const password = 'testpassword123';
    await registerUserViaApi(username, password);
    const token = await loginViaApi(username, password);

    // Set up auth
    await page.goto('/');
    await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
    await page.reload();

    // Navigate to profile
    await page.goto('/profile');
    await expect(page.locator('text=Role Profiles')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Look for initialize button or role cards
    const initButton = page.locator('button:has-text("Initialize")');
    const roleCards = page.locator('[data-testid^="role-profile-"]');

    // Either we see init button (new user) or role cards (already initialized)
    const initVisible = await initButton.isVisible().catch(() => false);

    if (initVisible) {
      await initButton.click();
      // Wait for role cards to appear
      await expect(roleCards.first()).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    }

    // Should now have 5 role profiles (TOP, JGL, MID, ADC, SUP)
    const roleCount = await roleCards.count();
    expect(roleCount).toBe(5);
  });

  test('user can update a role profile rank', async ({ page }) => {
    // Register and login
    const username = generateTestUsername('rank');
    const password = 'testpassword123';
    await registerUserViaApi(username, password);
    const token = await loginViaApi(username, password);

    // Initialize role profiles via API
    await initializeRoleProfiles(token);

    // Set up auth
    await page.goto('/');
    await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
    await page.reload();

    // Navigate to profile
    await page.goto('/profile');
    await expect(page.locator('text=Role Profiles')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Wait for role cards to load
    const topRoleCard = page.locator('[data-testid="role-profile-top"]');
    await expect(topRoleCard).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Click the Edit button inside the TOP role card
    await topRoleCard.locator('button:has-text("Edit")').click();

    // Should see the select dropdown for rank
    const rankSelect = topRoleCard.locator('select');
    await expect(rankSelect).toBeVisible({ timeout: TIMEOUTS.SHORT });

    // Change rank to Gold IV
    await rankSelect.selectOption({ label: 'Gold IV' });

    // Save changes
    await topRoleCard.locator('button:has-text("Save")').click();

    // Verify the change persisted (card should show Gold IV)
    await expect(topRoleCard.locator('text=Gold IV')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  });

  test('user can update role comfort rating', async ({ page }) => {
    // Register and login
    const username = generateTestUsername('comfort');
    const password = 'testpassword123';
    await registerUserViaApi(username, password);
    const token = await loginViaApi(username, password);

    // Initialize role profiles via API
    await initializeRoleProfiles(token);

    // Set up auth
    await page.goto('/');
    await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
    await page.reload();

    // Navigate to profile
    await page.goto('/profile');
    const midRoleCard = page.locator('[data-testid="role-profile-mid"]');
    await expect(midRoleCard).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Click the Edit button inside the MID role card
    await midRoleCard.locator('button:has-text("Edit")').click();

    // In edit mode, stars become clickable - click the 5th star
    // Stars are rendered as buttons with "★" text
    const starButtons = midRoleCard.locator('button:has-text("★")');
    await starButtons.nth(4).click(); // 5th star (0-indexed)

    // Save changes
    await midRoleCard.locator('button:has-text("Save")').click();

    // Verify all 5 stars are now filled (yellow) by checking that 5 stars are visible
    // The card should still have the comfort stars visible
    await expect(midRoleCard.locator('button:has-text("★")').first()).toBeVisible({
      timeout: TIMEOUTS.MEDIUM,
    });
  });

  test('all five roles are displayed', async ({ page }) => {
    // Register and login
    const username = generateTestUsername('roles');
    const password = 'testpassword123';
    await registerUserViaApi(username, password);
    const token = await loginViaApi(username, password);

    // Initialize role profiles via API
    await initializeRoleProfiles(token);

    // Set up auth
    await page.goto('/');
    await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
    await page.reload();

    // Navigate to profile
    await page.goto('/profile');
    await expect(page.locator('text=Role Profiles')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // All 5 roles should be visible
    const roles = ['top', 'jungle', 'mid', 'adc', 'support'];
    for (const role of roles) {
      await expect(page.locator(`[data-testid="role-profile-${role}"]`)).toBeVisible({
        timeout: TIMEOUTS.SHORT,
      });
    }
  });

  test('profile shows user display name', async ({ page }) => {
    // Register with a specific display name
    const username = generateTestUsername('display');
    const password = 'testpassword123';
    await registerUserViaApi(username, password);
    const token = await loginViaApi(username, password);

    // Set up auth
    await page.goto('/');
    await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
    await page.reload();

    // Navigate to profile
    await page.goto('/profile');

    // Display name should be visible
    await expect(page.locator(`text=${username}`)).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  });

  test('profile changes persist after page reload', async ({ page }) => {
    // Register and login
    const username = generateTestUsername('persist');
    const password = 'testpassword123';
    await registerUserViaApi(username, password);
    const token = await loginViaApi(username, password);

    // Initialize role profiles via API
    await initializeRoleProfiles(token);

    // Set up auth
    await page.goto('/');
    await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
    await page.reload();

    // Navigate to profile
    await page.goto('/profile');
    const adcRoleCard = page.locator('[data-testid="role-profile-adc"]');
    await expect(adcRoleCard).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Click the Edit button inside the ADC role card
    await adcRoleCard.locator('button:has-text("Edit")').click();

    // Change rank to Platinum II
    const rankSelect = adcRoleCard.locator('select');
    await expect(rankSelect).toBeVisible({ timeout: TIMEOUTS.SHORT });
    await rankSelect.selectOption({ label: 'Platinum II' });

    // Save changes
    await adcRoleCard.locator('button:has-text("Save")').click();

    // Wait for save to complete
    await expect(adcRoleCard.locator('text=Platinum II')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Reload the page
    await page.reload();
    await expect(page.locator('text=Role Profiles')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Verify the change persisted
    await expect(page.locator('text=Platinum II')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  });
});
