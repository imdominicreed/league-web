import { Page, Locator, expect, Response as PlaywrightResponse } from '@playwright/test';

/**
 * Centralized timeout constants for consistent wait behavior across tests.
 */
export const TIMEOUTS = {
  /** Simple DOM updates, element visibility */
  SHORT: 8000,
  /** API calls, state sync, WebSocket messages */
  MEDIUM: 20000,
  /** Complex operations like team generation */
  LONG: 45000,
  /** Draft phase transitions with timer */
  DRAFT_PHASE: 60000,
  /** Multi-user tests with 10 browser contexts */
  MULTI_USER: 90000,
} as const;

/**
 * Faster polling intervals for expect.poll() calls.
 * Starts fast and backs off gradually.
 */
export const POLL_INTERVALS: number[] = [100, 200, 500, 1000];

/**
 * Wait for a condition to become true using Playwright's expect.poll().
 * Preferred over manual polling with waitForTimeout.
 */
export async function waitForCondition(
  conditionFn: () => Promise<boolean>,
  options: { timeout?: number; message?: string } = {}
): Promise<void> {
  const { timeout = TIMEOUTS.MEDIUM, message = 'Condition not met' } = options;
  await expect
    .poll(conditionFn, { timeout, message, intervals: POLL_INTERVALS })
    .toBe(true);
}

/**
 * Wait for a locator's text content to change from its current value.
 */
export async function waitForTextChange(
  locator: Locator,
  options: { timeout?: number } = {}
): Promise<string | null> {
  const initialText = await locator.textContent();
  await expect
    .poll(async () => (await locator.textContent()) !== initialText, {
      timeout: options.timeout ?? TIMEOUTS.MEDIUM,
      intervals: POLL_INTERVALS,
    })
    .toBe(true);
  return locator.textContent();
}

/**
 * Wait for an element count to reach a specific value.
 */
export async function waitForCount(
  locator: Locator,
  count: number,
  options: { timeout?: number } = {}
): Promise<void> {
  await expect(locator).toHaveCount(count, {
    timeout: options.timeout ?? TIMEOUTS.MEDIUM,
  });
}

/**
 * Wait for a specific API response matching a URL pattern.
 */
export async function waitForApiResponse(
  page: Page,
  urlPattern: string | RegExp,
  options: { timeout?: number } = {}
): Promise<PlaywrightResponse> {
  return page.waitForResponse(urlPattern, {
    timeout: options.timeout ?? TIMEOUTS.MEDIUM,
  });
}

/**
 * Wait for WebSocket to be connected by checking for absence of disconnect indicator.
 * This is a common pattern in the draft/lobby pages.
 */
export async function waitForWebSocketConnected(
  page: Page,
  options: { timeout?: number } = {}
): Promise<void> {
  const disconnectedIndicator = page.locator('text=Disconnected');
  await expect(disconnectedIndicator).not.toBeVisible({
    timeout: options.timeout ?? TIMEOUTS.MEDIUM,
  });
}

/**
 * Wait for navigation to complete and page to be ready.
 */
export async function waitForPageReady(
  page: Page,
  options: { timeout?: number } = {}
): Promise<void> {
  await page.waitForLoadState('networkidle', {
    timeout: options.timeout ?? TIMEOUTS.MEDIUM,
  });
}

/**
 * Wait for WebSocket to be connected by checking Redux state.
 * More reliable than checking UI text for connection status.
 */
export async function waitForWebSocketReady(
  page: Page,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.MEDIUM;
  await expect
    .poll(
      async () => {
        const wsState = await page.evaluate(() => {
          // Check if Redux store exists and has draft connected state
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const state = (window as any).__REDUX_STORE__?.getState?.();
          return state?.draft?.connected || state?.ws?.connected || false;
        });
        return wsState === true;
      },
      { timeout, intervals: POLL_INTERVALS }
    )
    .toBe(true);
}

/**
 * Wait for all users to navigate to a URL matching the pattern.
 * Useful for ensuring all users are on the same page before continuing.
 */
export async function waitForAllUsersAtUrl(
  users: { page: Page }[],
  urlPattern: RegExp,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.MULTI_USER;
  await Promise.all(
    users.map((user) => user.page.waitForURL(urlPattern, { timeout }))
  );
}

/**
 * Wait for all pages to show a specific element as visible.
 * Useful for synchronizing multi-user tests.
 */
export async function waitForAllUsersReady(
  pages: Page[],
  selector: string,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.MULTI_USER;
  await Promise.all(
    pages.map((page) =>
      expect(page.locator(selector)).toBeVisible({ timeout })
    )
  );
}

/**
 * Retry an async operation with exponential backoff.
 * Useful for flaky operations that may succeed on retry.
 */
export async function retryWithBackoff<T>(
  operation: () => Promise<T>,
  options: { maxRetries?: number; baseDelayMs?: number } = {}
): Promise<T> {
  const { maxRetries = 3, baseDelayMs = 500 } = options;
  let lastError: Error | null = null;

  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      return await operation();
    } catch (err) {
      lastError = err as Error;
      if (attempt < maxRetries) {
        const delay = baseDelayMs * Math.pow(2, attempt - 1);
        await new Promise((r) => setTimeout(r, delay));
      }
    }
  }

  throw lastError;
}
