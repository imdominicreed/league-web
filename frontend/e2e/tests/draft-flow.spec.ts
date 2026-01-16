import { test, expect } from '@playwright/test';
import { HomePage, DraftRoomPage } from '../page-objects';
import { generateTestUsername, registerUserViaApi } from '../fixtures';
import { TIMEOUTS } from '../helpers/wait-strategies';

const API_BASE = 'http://localhost:9999/api/v1';

/**
 * Helper to create a room via API
 */
async function createRoomViaApi(
  token: string,
  options: { draftMode?: string; timerDurationSeconds?: number } = {}
): Promise<{ id: string; shortCode: string }> {
  const response = await fetch(`${API_BASE}/rooms`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      draftMode: options.draftMode || 'pro_play',
      timerDurationSeconds: options.timerDurationSeconds || 30,
    }),
  });
  if (!response.ok) {
    throw new Error(`Create room failed: ${response.status}`);
  }
  return response.json();
}

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

test.describe('1v1 Draft Flow', () => {
  test.describe.configure({ mode: 'serial' });

  test('user can create a draft room', async ({ page }) => {
    // Register user via API
    const username = generateTestUsername('draft');
    const password = 'testpassword123';
    await registerUserViaApi(username, password);

    // Login and get token
    const token = await loginViaApi(username, password);

    // Set token in browser
    await page.goto('/');
    await page.evaluate((accessToken) => {
      localStorage.setItem('accessToken', accessToken);
    }, token);
    await page.reload();

    // Navigate to create draft
    const homePage = new HomePage(page);
    await homePage.goto();
    await homePage.expectAuthenticated(username);

    // Click Create Draft
    await page.click('a:has-text("Create Draft Room")');
    await page.waitForURL('/create');

    // Select Pro Play mode (default) and create room
    await page.click('button:has-text("Pro Play")');
    await page.click('button:has-text("Create Room")');

    // Should redirect to draft room
    await page.waitForURL(/\/draft\//, { timeout: TIMEOUTS.LONG });

    // Draft room page should load
    const draftPage = new DraftRoomPage(page);
    await draftPage.waitForDraftLoaded();
  });

  test('user can join a draft room via code', async ({ page }) => {
    // Create two users
    const creator = generateTestUsername('creator');
    const joiner = generateTestUsername('joiner');
    const password = 'testpassword123';

    await registerUserViaApi(creator, password);
    await registerUserViaApi(joiner, password);

    // Creator creates room via API
    const creatorToken = await loginViaApi(creator, password);
    const room = await createRoomViaApi(creatorToken);

    // Joiner logs in and joins via UI
    const joinerToken = await loginViaApi(joiner, password);

    await page.goto('/');
    await page.evaluate((accessToken) => {
      localStorage.setItem('accessToken', accessToken);
    }, joinerToken);
    await page.reload();

    // Navigate to join draft
    await page.click('a:has-text("Join Room")');
    await page.waitForURL('/join');

    // Enter room code
    await page.fill('input#code', room.shortCode);
    await page.click('button:has-text("Join Room")');

    // Should redirect to draft room
    await page.waitForURL(/\/draft\//, { timeout: TIMEOUTS.LONG });

    const draftPage = new DraftRoomPage(page);
    await draftPage.waitForDraftLoaded();
  });

  test('1v1 draft ready and start flow', async ({ browser }) => {
    test.setTimeout(120000);

    // Create two users
    const user1Name = generateTestUsername('blue');
    const user2Name = generateTestUsername('red');
    const password = 'testpassword123';

    await registerUserViaApi(user1Name, password);
    await registerUserViaApi(user2Name, password);

    // Get tokens
    const user1Token = await loginViaApi(user1Name, password);
    const user2Token = await loginViaApi(user2Name, password);

    // User 1 creates room
    const room = await createRoomViaApi(user1Token, { timerDurationSeconds: 30 });

    // Create two browser contexts
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const page1 = await context1.newPage();
    const page2 = await context2.newPage();

    try {
      // Set up auth for both
      await page1.goto('/');
      await page1.evaluate((t) => localStorage.setItem('accessToken', t), user1Token);
      await page1.reload();

      await page2.goto('/');
      await page2.evaluate((t) => localStorage.setItem('accessToken', t), user2Token);
      await page2.reload();

      // Both navigate to draft room
      await page1.goto(`/draft/${room.id}`);
      await page2.goto(`/draft/${room.id}`);

      const draftPage1 = new DraftRoomPage(page1);
      const draftPage2 = new DraftRoomPage(page2);

      // Wait for both to load
      await draftPage1.waitForDraftLoaded();
      await draftPage2.waitForDraftLoaded();

      // Wait for WebSocket connections
      await draftPage1.waitForWebSocketConnected();
      await draftPage2.waitForWebSocketConnected();

      // Both should be in waiting state initially
      await draftPage1.waitForWaitingState();
      await draftPage2.waitForWaitingState();

      // Both click Ready
      if (await draftPage1.canClickReady()) {
        await draftPage1.clickReady();
      }
      if (await draftPage2.canClickReady()) {
        await draftPage2.clickReady();
      }

      // One of them should see Start Draft button
      let starter: DraftRoomPage | null = null;
      for (const dp of [draftPage1, draftPage2]) {
        const startBtn = dp.getPage().locator('button:has-text("Start Draft")');
        if ((await startBtn.count()) > 0 && (await startBtn.isVisible())) {
          starter = dp;
          break;
        }
      }

      expect(starter).not.toBeNull();
      await starter!.clickStartDraft();

      // Draft should become active
      await draftPage1.waitForActiveState();

      // Champions should be loaded
      await draftPage1.waitForChampionsLoaded();
      await draftPage2.waitForChampionsLoaded();
    } finally {
      await context1.close();
      await context2.close();
    }
  });

  test('1v1 draft complete ban and pick phases', async ({ browser }) => {
    test.setTimeout(180000);

    // Create two users
    const user1Name = generateTestUsername('p1');
    const user2Name = generateTestUsername('p2');
    const password = 'testpassword123';

    await registerUserViaApi(user1Name, password);
    await registerUserViaApi(user2Name, password);

    const user1Token = await loginViaApi(user1Name, password);
    const user2Token = await loginViaApi(user2Name, password);

    // User 1 creates room with short timer
    const room = await createRoomViaApi(user1Token, { timerDurationSeconds: 30 });

    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const page1 = await context1.newPage();
    const page2 = await context2.newPage();

    try {
      // Set up auth
      await page1.goto('/');
      await page1.evaluate((t) => localStorage.setItem('accessToken', t), user1Token);
      await page1.reload();

      await page2.goto('/');
      await page2.evaluate((t) => localStorage.setItem('accessToken', t), user2Token);
      await page2.reload();

      // Navigate to draft
      await page1.goto(`/draft/${room.id}`);
      await page2.goto(`/draft/${room.id}`);

      const draftPage1 = new DraftRoomPage(page1);
      const draftPage2 = new DraftRoomPage(page2);

      await draftPage1.waitForDraftLoaded();
      await draftPage2.waitForDraftLoaded();
      await draftPage1.waitForWebSocketConnected();
      await draftPage2.waitForWebSocketConnected();

      // Ready up and start
      if (await draftPage1.canClickReady()) await draftPage1.clickReady();
      if (await draftPage2.canClickReady()) await draftPage2.clickReady();

      for (const dp of [draftPage1, draftPage2]) {
        const startBtn = dp.getPage().locator('button:has-text("Start Draft")');
        if ((await startBtn.count()) > 0 && (await startBtn.isVisible())) {
          await startBtn.click();
          break;
        }
      }

      await draftPage1.waitForActiveState();

      // Complete first 6 ban phases
      for (let phase = 0; phase < 6; phase++) {
        // Find who has their turn
        let turnPage: DraftRoomPage | null = null;
        for (const dp of [draftPage1, draftPage2]) {
          if (await dp.isYourTurn()) {
            turnPage = dp;
            break;
          }
        }

        expect(turnPage).not.toBeNull();
        await turnPage!.performBanOrPick(phase);
      }

      // After 6 bans, verify we're in pick phase
      const action = await draftPage1.getCurrentAction();
      expect(action.toLowerCase()).toContain('pick');

      // Complete first 2 pick phases
      for (let phase = 0; phase < 2; phase++) {
        let turnPage: DraftRoomPage | null = null;
        for (const dp of [draftPage1, draftPage2]) {
          if (await dp.isYourTurn()) {
            turnPage = dp;
            break;
          }
        }

        expect(turnPage).not.toBeNull();
        await turnPage!.performBanOrPick(10 + phase);
      }

      // Verify picks are visible
      await draftPage1.expectPicksVisible();
    } finally {
      await context1.close();
      await context2.close();
    }
  });

  test('draft room displays champion grid correctly', async ({ page }) => {
    // Register and login
    const username = generateTestUsername('grid');
    const password = 'testpassword123';
    await registerUserViaApi(username, password);
    const token = await loginViaApi(username, password);

    // Create room
    const room = await createRoomViaApi(token);

    // Set up auth
    await page.goto('/');
    await page.evaluate((t) => localStorage.setItem('accessToken', t), token);
    await page.reload();

    // Navigate to draft
    await page.goto(`/draft/${room.id}`);

    const draftPage = new DraftRoomPage(page);
    await draftPage.waitForDraftLoaded();
    await draftPage.waitForWebSocketConnected();

    // Ready up and check we can start
    if (await draftPage.canClickReady()) {
      await draftPage.clickReady();
    }

    // Champion grid should have champions
    await expect(page.locator('[data-testid="champion-grid"]')).toBeVisible();
    await expect(page.locator('[data-testid="champion-grid"] button').first()).toBeVisible();

    // There should be many champions available
    const championCount = await page.locator('[data-testid="champion-grid"] button').count();
    expect(championCount).toBeGreaterThan(100);
  });

  test('timer countdown is displayed correctly', async ({ browser }) => {
    test.setTimeout(120000);

    // Create two users
    const user1Name = generateTestUsername('timer1');
    const user2Name = generateTestUsername('timer2');
    const password = 'testpassword123';

    await registerUserViaApi(user1Name, password);
    await registerUserViaApi(user2Name, password);

    const user1Token = await loginViaApi(user1Name, password);
    const user2Token = await loginViaApi(user2Name, password);

    // Create room with 30 second timer
    const room = await createRoomViaApi(user1Token, { timerDurationSeconds: 30 });

    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const page1 = await context1.newPage();
    const page2 = await context2.newPage();

    try {
      // Set up auth
      await page1.goto('/');
      await page1.evaluate((t) => localStorage.setItem('accessToken', t), user1Token);
      await page1.reload();

      await page2.goto('/');
      await page2.evaluate((t) => localStorage.setItem('accessToken', t), user2Token);
      await page2.reload();

      // Navigate to draft
      await page1.goto(`/draft/${room.id}`);
      await page2.goto(`/draft/${room.id}`);

      const draftPage1 = new DraftRoomPage(page1);
      const draftPage2 = new DraftRoomPage(page2);

      await draftPage1.waitForDraftLoaded();
      await draftPage2.waitForDraftLoaded();
      await draftPage1.waitForWebSocketConnected();
      await draftPage2.waitForWebSocketConnected();

      // Ready up and start
      if (await draftPage1.canClickReady()) await draftPage1.clickReady();
      if (await draftPage2.canClickReady()) await draftPage2.clickReady();

      for (const dp of [draftPage1, draftPage2]) {
        const startBtn = dp.getPage().locator('button:has-text("Start Draft")');
        if ((await startBtn.count()) > 0 && (await startBtn.isVisible())) {
          await startBtn.click();
          break;
        }
      }

      await draftPage1.waitForActiveState();

      // Timer should be visible and counting down
      const initialTimer = await draftPage1.getTimerSeconds();
      expect(initialTimer).toBeGreaterThan(0);
      expect(initialTimer).toBeLessThanOrEqual(30);

      // Wait a moment and verify it decreased
      await draftPage1.waitForTimerBelow(initialTimer);

      const laterTimer = await draftPage1.getTimerSeconds();
      expect(laterTimer).toBeLessThan(initialTimer);
    } finally {
      await context1.close();
      await context2.close();
    }
  });
});
