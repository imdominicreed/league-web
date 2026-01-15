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
    await expect(page.locator('[data-testid="role-profile-TOP"]')).toBeVisible({
      timeout: TIMEOUTS.MEDIUM,
    });

    // Click on TOP role to edit
    await page.click('[data-testid="role-profile-TOP"]');

    // Should see edit modal or inline edit
    const rankSelect = page.locator('select[name="rank"]').or(page.locator('[data-testid="rank-select"]'));
    await expect(rankSelect).toBeVisible({ timeout: TIMEOUTS.SHORT });

    // Change rank to Gold IV
    await rankSelect.selectOption({ label: 'Gold IV' });

    // Save changes
    const saveButton = page.locator('button:has-text("Save")');
    if (await saveButton.isVisible()) {
      await saveButton.click();
    }

    // Verify the change persisted
    await expect(page.locator('text=Gold IV')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
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
    await expect(page.locator('[data-testid="role-profile-MID"]')).toBeVisible({
      timeout: TIMEOUTS.MEDIUM,
    });

    // Click on MID role to edit
    await page.click('[data-testid="role-profile-MID"]');

    // Should see comfort rating control
    const comfortInput = page
      .locator('input[name="comfort"]')
      .or(page.locator('[data-testid="comfort-slider"]'))
      .or(page.locator('select[name="comfort"]'));
    await expect(comfortInput).toBeVisible({ timeout: TIMEOUTS.SHORT });

    // Change comfort to 5 (highest)
    const isSelect = (await comfortInput.evaluate((el) => el.tagName)) === 'SELECT';
    if (isSelect) {
      await comfortInput.selectOption('5');
    } else {
      await comfortInput.fill('5');
    }

    // Save changes
    const saveButton = page.locator('button:has-text("Save")');
    if (await saveButton.isVisible()) {
      await saveButton.click();
    }

    // Verify the change (look for comfort indicator)
    await expect(page.locator('text=5').or(page.locator('[data-comfort="5"]'))).toBeVisible({
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
    const roles = ['TOP', 'JGL', 'MID', 'ADC', 'SUP'];
    for (const role of roles) {
      await expect(
        page.locator(`[data-testid="role-profile-${role}"]`).or(page.locator(`text=${role}`))
      ).toBeVisible({ timeout: TIMEOUTS.SHORT });
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
    await expect(page.locator('[data-testid="role-profile-ADC"]')).toBeVisible({
      timeout: TIMEOUTS.MEDIUM,
    });

    // Edit ADC role
    await page.click('[data-testid="role-profile-ADC"]');

    // Change rank
    const rankSelect = page.locator('select[name="rank"]').or(page.locator('[data-testid="rank-select"]'));
    if (await rankSelect.isVisible()) {
      await rankSelect.selectOption({ label: 'Platinum II' });

      // Save
      const saveButton = page.locator('button:has-text("Save")');
      if (await saveButton.isVisible()) {
        await saveButton.click();
      }
    }

    // Reload the page
    await page.reload();
    await expect(page.locator('text=Role Profiles')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });

    // Verify the change persisted
    await expect(page.locator('text=Platinum II')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  });
});
