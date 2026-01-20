import { test, expect, Page } from '@playwright/test';
import { createTestUser, TestUser } from '../helpers/test-utils';

const API_BASE = 'http://localhost:9999/api/v1';

/**
 * Create a draft room via API
 * @param timerDurationSeconds - Timer duration in seconds
 */
async function createRoom(page: Page, token: string, timerDurationSeconds: number) {
  const res = await page.request.post(`${API_BASE}/rooms`, {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      draftMode: 'pro_play',
      timerDuration: timerDurationSeconds,
    },
  });
  expect(res.ok()).toBeTruthy();
  return await res.json();
}

/**
 * WebSocket helper for draft room interactions
 */
class DraftWebSocket {
  private ws: WebSocket | null = null;
  private messages: any[] = [];
  private messageHandlers: ((msg: any) => void)[] = [];
  private connected = false;
  private token: string;
  private page: Page;

  constructor(page: Page, token: string) {
    this.page = page;
    this.token = token;
  }

  async connect(): Promise<void> {
    return this.page.evaluate((token) => {
      return new Promise<void>((resolve, reject) => {
        const ws = new WebSocket(`ws://localhost:9999/api/v1/ws?token=${token}`);
        (window as any).__testWs = ws;
        (window as any).__testWsMessages = [];

        ws.onopen = () => resolve();
        ws.onerror = (e) => reject(e);
        ws.onmessage = (e) => {
          const msg = JSON.parse(e.data);
          (window as any).__testWsMessages.push(msg);
        };
      });
    }, this.token);
  }

  async sendCommand(action: string, payload?: any): Promise<void> {
    await this.page.evaluate(({ action, payload }) => {
      const ws = (window as any).__testWs;
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
          type: 'COMMAND',
          payload: { action, ...(payload ? { payload } : {}) },
          timestamp: Date.now(),
        }));
      }
    }, { action, payload });
  }

  async joinRoom(roomId: string, side: string): Promise<void> {
    await this.sendCommand('join_room', { roomId, side });
    // Wait a bit for the room to process
    await this.page.waitForTimeout(500);
  }

  async setReady(ready: boolean): Promise<void> {
    await this.sendCommand('set_ready', { ready });
    await this.page.waitForTimeout(200);
  }

  async startDraft(): Promise<void> {
    await this.sendCommand('start_draft');
    await this.page.waitForTimeout(500);
  }

  async pauseDraft(): Promise<void> {
    await this.sendCommand('pause_draft');
    await this.page.waitForTimeout(500);
  }

  async readyToResume(ready: boolean): Promise<void> {
    await this.sendCommand('resume_ready', { ready });
    await this.page.waitForTimeout(200);
  }

  async selectChampion(championId: string): Promise<void> {
    await this.sendCommand('select_champion', { championId });
    await this.page.waitForTimeout(100);
  }

  async lockIn(): Promise<void> {
    await this.sendCommand('lock_in');
    await this.page.waitForTimeout(500);
  }

  async getMessages(): Promise<any[]> {
    return this.page.evaluate(() => (window as any).__testWsMessages);
  }

  async clearMessages(): Promise<void> {
    await this.page.evaluate(() => {
      (window as any).__testWsMessages = [];
    });
  }

  async getLatestMessageOfType(type: string): Promise<any | null> {
    const messages = await this.getMessages();
    for (let i = messages.length - 1; i >= 0; i--) {
      if (messages[i].type === type) {
        return messages[i];
      }
    }
    return null;
  }

  async close(): Promise<void> {
    await this.page.evaluate(() => {
      const ws = (window as any).__testWs;
      if (ws) {
        ws.close();
      }
    });
  }
}

test.describe('BUG-011: Draft Timer Resets on Unpause Instead of Resuming', () => {
  test('timer should resume from paused value, not reset to full duration', async ({ page, browser }) => {
    // Create two users
    const blueUser = await createTestUser(page, 'bug011_blue');
    const redUser = await createTestUser(page, 'bug011_red');

    // Create a room with a 10 second timer
    const timerDurationSeconds = 10;
    const timerDurationMs = timerDurationSeconds * 1000;
    const room = await createRoom(page, blueUser.token, timerDurationSeconds);
    const roomId = room.id;

    // Create a second context for red player
    const context2 = await browser.newContext();
    const page2 = await context2.newPage();

    try {
      // Navigate both pages to set up WebSocket context
      await page.goto('/');
      await page2.goto('/');

      // Set up WebSocket connections
      const blueWs = new DraftWebSocket(page, blueUser.token);
      const redWs = new DraftWebSocket(page2, redUser.token);

      await blueWs.connect();
      await redWs.connect();

      // Join room
      await blueWs.joinRoom(roomId, 'blue');
      await redWs.joinRoom(roomId, 'red');

      // Both players ready
      await blueWs.setReady(true);
      await redWs.setReady(true);

      // Blue starts the draft
      await blueWs.startDraft();

      // Wait for draft to start and timer to tick down a bit
      await page.waitForTimeout(2000); // 2 seconds pass

      // Blue player pauses the draft
      await blueWs.pauseDraft();

      // Get the paused message to check frozen timer value
      const pausedMsg = await blueWs.getLatestMessageOfType('DRAFT_PAUSED');
      expect(pausedMsg).not.toBeNull();
      console.log('Paused message:', pausedMsg);

      // The frozen timer should be around 8000ms (10000 - 2000)
      const frozenTimerMs = pausedMsg.payload.timerFrozenAt;
      expect(frozenTimerMs).toBeLessThan(timerDurationMs);
      expect(frozenTimerMs).toBeGreaterThan(5000); // Should be > 5 seconds remaining

      // Both players ready to resume
      await blueWs.readyToResume(true);
      await redWs.readyToResume(true);

      // Wait for resume countdown (5 seconds) plus a bit
      await page.waitForTimeout(6000);

      // Get the resumed message
      const resumedMsg = await blueWs.getLatestMessageOfType('DRAFT_RESUMED');
      expect(resumedMsg).not.toBeNull();
      console.log('Resumed message:', resumedMsg);

      // The resumed timer should match the frozen timer
      const resumedTimerMs = resumedMsg.payload.timerRemainingMs;
      expect(resumedTimerMs).toBe(frozenTimerMs);

      // Clear messages to track new ones
      await blueWs.clearMessages();

      // Blue player selects and locks in a champion to advance to phase 1
      await blueWs.selectChampion('Ezreal');
      await blueWs.lockIn();

      // Check for phase changed message
      const phaseChangedMsg = await blueWs.getLatestMessageOfType('PHASE_CHANGED');
      expect(phaseChangedMsg).not.toBeNull();
      console.log('Phase changed message:', phaseChangedMsg);

      // BUG-011 FIX: The new phase timer should be the FULL duration (10000ms)
      // Before the fix, it would be the remaining time from the pause (~8000ms)
      const newPhaseTimerMs = phaseChangedMsg.payload.timerRemainingMs;
      expect(newPhaseTimerMs).toBe(timerDurationMs);

      // Clean up
      await blueWs.close();
      await redWs.close();
    } finally {
      await context2.close();
    }
  });

  test('multiple phases after resume should all have correct full timer duration', async ({ page, browser }) => {
    // Create two users
    const blueUser = await createTestUser(page, 'bug011b_blue');
    const redUser = await createTestUser(page, 'bug011b_red');

    // Create a room with a 5 second timer
    const timerDurationSeconds = 5;
    const timerDurationMs = timerDurationSeconds * 1000;
    const room = await createRoom(page, blueUser.token, timerDurationSeconds);
    const roomId = room.id;

    // Create a second context for red player
    const context2 = await browser.newContext();
    const page2 = await context2.newPage();

    try {
      // Navigate both pages
      await page.goto('/');
      await page2.goto('/');

      // Set up WebSocket connections
      const blueWs = new DraftWebSocket(page, blueUser.token);
      const redWs = new DraftWebSocket(page2, redUser.token);

      await blueWs.connect();
      await redWs.connect();

      // Join room
      await blueWs.joinRoom(roomId, 'blue');
      await redWs.joinRoom(roomId, 'red');

      // Both players ready
      await blueWs.setReady(true);
      await redWs.setReady(true);

      // Blue starts the draft
      await blueWs.startDraft();

      // Wait 1 second then pause
      await page.waitForTimeout(1000);
      await blueWs.pauseDraft();

      // Both players resume
      await blueWs.readyToResume(true);
      await redWs.readyToResume(true);
      await page.waitForTimeout(6000);

      // Complete a few phases and verify each has full timer duration
      const champions = ['Ezreal', 'Jinx', 'Ahri', 'Zed'];

      for (let i = 0; i < 4; i++) {
        await blueWs.clearMessages();

        // Alternate between blue and red based on phase (simplified)
        // Phase 0 = Blue ban, Phase 1 = Red ban, Phase 2 = Blue ban, Phase 3 = Red ban
        const activeWs = i % 2 === 0 ? blueWs : redWs;

        await activeWs.selectChampion(champions[i]);
        await activeWs.lockIn();

        // Check phase changed message has full timer
        // Note: Messages go to both clients, so check blue's messages
        const phaseChangedMsg = await blueWs.getLatestMessageOfType('PHASE_CHANGED');

        if (phaseChangedMsg) {
          console.log(`Phase ${i + 1} timer: ${phaseChangedMsg.payload.timerRemainingMs}ms (expected ${timerDurationMs}ms)`);
          // Each new phase should start with full timer duration
          expect(phaseChangedMsg.payload.timerRemainingMs).toBe(timerDurationMs);
        }
      }

      // Clean up
      await blueWs.close();
      await redWs.close();
    } finally {
      await context2.close();
    }
  });

  test('timer tick values should be consistent after resume', async ({ page, browser }) => {
    // Create two users
    const blueUser = await createTestUser(page, 'bug011c_blue');
    const redUser = await createTestUser(page, 'bug011c_red');

    // Create a room with 10 second timer
    const timerDurationSeconds = 10;
    const timerDurationMs = timerDurationSeconds * 1000;
    const room = await createRoom(page, blueUser.token, timerDurationSeconds);
    const roomId = room.id;

    // Create a second context for red player
    const context2 = await browser.newContext();
    const page2 = await context2.newPage();

    try {
      await page.goto('/');
      await page2.goto('/');

      const blueWs = new DraftWebSocket(page, blueUser.token);
      const redWs = new DraftWebSocket(page2, redUser.token);

      await blueWs.connect();
      await redWs.connect();

      await blueWs.joinRoom(roomId, 'blue');
      await redWs.joinRoom(roomId, 'red');

      await blueWs.setReady(true);
      await redWs.setReady(true);

      await blueWs.startDraft();

      // Wait 3 seconds
      await page.waitForTimeout(3000);

      // Pause
      await blueWs.pauseDraft();

      const pausedMsg = await blueWs.getLatestMessageOfType('DRAFT_PAUSED');
      const frozenTimer = pausedMsg?.payload?.timerFrozenAt || 0;
      console.log('Frozen timer:', frozenTimer);

      // Resume
      await blueWs.readyToResume(true);
      await redWs.readyToResume(true);
      await page.waitForTimeout(6000); // Wait for resume countdown

      // Clear and collect timer ticks
      await blueWs.clearMessages();
      await page.waitForTimeout(2000); // Collect 2 seconds of ticks

      const messages = await blueWs.getMessages();
      const timerTicks = messages.filter(m => m.type === 'TIMER_TICK');

      // Verify timer ticks are counting down from the resumed value
      // The ticks should be around (frozenTimer - elapsed)
      if (timerTicks.length > 0) {
        console.log('Timer ticks after resume:', timerTicks.map(t => t.payload.remainingMs));

        // First tick after our 2 second wait should be less than the frozen timer
        const firstTick = timerTicks[0].payload.remainingMs;
        expect(firstTick).toBeLessThan(frozenTimer);
        expect(firstTick).toBeGreaterThan(0); // Should still have time left
      }

      await blueWs.close();
      await redWs.close();
    } finally {
      await context2.close();
    }
  });
});
