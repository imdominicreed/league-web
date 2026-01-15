import { Page, expect } from '@playwright/test';
import { TIMEOUTS, POLL_INTERVALS } from './wait-strategies';

/**
 * WebSocket synchronization utilities for E2E tests.
 * These utilities help wait for state updates propagated via WebSocket
 * instead of using arbitrary timeouts or page reloads.
 */

/**
 * Wait for the lobby state to update via polling.
 * The lobby page polls every 3 seconds, so we wait for the next poll cycle.
 */
export async function waitForLobbyStateUpdate(
  page: Page,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.MEDIUM;
  // Wait for network to settle after API call
  await page.waitForLoadState('networkidle', { timeout });
}

/**
 * Wait for a specific element to appear after a state change.
 * More reliable than waitForTimeout as it waits for actual DOM changes.
 */
export async function waitForElementAfterAction(
  page: Page,
  selector: string,
  options: { timeout?: number; state?: 'visible' | 'hidden' } = {}
): Promise<void> {
  const { timeout = TIMEOUTS.MEDIUM, state = 'visible' } = options;
  const locator = page.locator(selector);

  if (state === 'visible') {
    await expect(locator).toBeVisible({ timeout });
  } else {
    await expect(locator).not.toBeVisible({ timeout });
  }
}

/**
 * Wait for the pending action banner to appear/disappear.
 * Uses data-testid for reliable selection.
 */
export async function waitForPendingActionBanner(
  page: Page,
  options: { visible?: boolean; timeout?: number } = {}
): Promise<void> {
  const { visible = true, timeout = TIMEOUTS.MEDIUM } = options;
  const banner = page.locator('[data-testid="pending-action-banner"]');

  if (visible) {
    await expect(banner).toBeVisible({ timeout });
  } else {
    await expect(banner).not.toBeVisible({ timeout });
  }
}

/**
 * Wait for the approve button to be visible in the pending action banner.
 */
export async function waitForApproveButton(
  page: Page,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.MEDIUM;
  await expect(
    page.locator('[data-testid="pending-action-approve-button"]')
  ).toBeVisible({ timeout });
}

/**
 * Wait for a specific lobby status to be displayed.
 */
export async function waitForLobbyStatus(
  page: Page,
  status: string,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.MEDIUM;
  await expect(
    page.locator('[data-testid="lobby-status-display"]')
  ).toContainText(status, { timeout });
}

/**
 * Wait for match options to be visible after team generation.
 */
export async function waitForMatchOptionsVisible(
  page: Page,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.LONG;
  await expect(
    page.locator('[data-testid="match-option-1"]')
  ).toBeVisible({ timeout });
}

/**
 * Wait for a player to appear/disappear from the lobby.
 */
export async function waitForPlayerInLobby(
  page: Page,
  displayName: string,
  options: { visible?: boolean; timeout?: number } = {}
): Promise<void> {
  const { visible = true, timeout = TIMEOUTS.MEDIUM } = options;
  const playerLocator = page.locator(`[data-testid^="lobby-player-"]`).filter({
    hasText: displayName
  });

  if (visible) {
    await expect(playerLocator).toBeVisible({ timeout });
  } else {
    await expect(playerLocator).not.toBeVisible({ timeout });
  }
}

/**
 * Wait for captain status to change for the current user.
 * Checks for the presence of "Captain Controls" vs "Player Actions".
 */
export async function waitForCaptainStatus(
  page: Page,
  isCaptain: boolean,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.MEDIUM;
  const expectedText = isCaptain ? 'Captain Controls' : 'Player Actions';

  await expect(page.locator(`text=${expectedText}`)).toBeVisible({ timeout });
}

/**
 * Wait for a player's ready status to update.
 */
export async function waitForPlayerReady(
  page: Page,
  displayName: string,
  isReady: boolean,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.MEDIUM;

  // Wait for the player card to show the correct ready state
  await expect.poll(
    async () => {
      const playerCard = page.locator(`[data-testid^="lobby-player-"]`).filter({
        hasText: displayName
      });
      const readyIndicator = playerCard.locator('[data-testid="player-ready-status"]');
      const text = await readyIndicator.textContent();
      return isReady ? text?.includes('Ready') : text?.includes('Not Ready');
    },
    { timeout, intervals: POLL_INTERVALS }
  ).toBe(true);
}

/**
 * Wait for the team column to show correct player count.
 */
export async function waitForTeamPlayerCount(
  page: Page,
  team: 'blue' | 'red',
  count: number,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.MEDIUM;
  const teamColumn = page.locator(`[data-testid="team-column-${team}"]`);

  await expect(teamColumn).toContainText(`${count}/5`, { timeout });
}

/**
 * Poll-based wait for a condition without page reload.
 * Useful when waiting for state that updates via polling (not WebSocket).
 */
export async function waitForConditionWithoutReload(
  page: Page,
  conditionFn: () => Promise<boolean>,
  options: { timeout?: number; message?: string } = {}
): Promise<void> {
  const { timeout = TIMEOUTS.MEDIUM, message = 'Condition not met' } = options;

  await expect.poll(conditionFn, {
    timeout,
    message,
    intervals: POLL_INTERVALS
  }).toBe(true);
}

/**
 * Wait for the "Approved" badge to appear for a captain.
 */
export async function waitForApprovedBadge(
  page: Page,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.MEDIUM;
  await expect(
    page.locator('[data-testid="pending-action-approved-badge"]')
  ).toBeVisible({ timeout });
}

/**
 * Wait for the Start Draft button to become visible.
 */
export async function waitForStartDraftButton(
  page: Page,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.MEDIUM;
  const startButton = page.locator('[data-testid="lobby-button-start-draft"]')
    .or(page.locator('[data-testid="captain-button-start-draft"]'));

  await expect(startButton.first()).toBeVisible({ timeout });
}

/**
 * Reload page and wait for lobby to load.
 * This is a helper for cases where reload is still needed,
 * but ensures proper waiting after reload.
 */
export async function reloadAndWaitForLobby(
  page: Page,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.MEDIUM;
  await page.reload();
  await expect(page.locator('text=10-Man Lobby')).toBeVisible({ timeout });
}

/**
 * Wait for any of the specified elements to become visible.
 * Useful for cases where different UI states may appear.
 */
export async function waitForAnyElement(
  page: Page,
  selectors: string[],
  options: { timeout?: number } = {}
): Promise<string> {
  const timeout = options.timeout ?? TIMEOUTS.MEDIUM;

  const locators = selectors.map(s => page.locator(s));
  const combined = locators.reduce((a, b) => a.or(b));

  await expect(combined.first()).toBeVisible({ timeout });

  // Return which selector matched
  for (const selector of selectors) {
    if (await page.locator(selector).isVisible()) {
      return selector;
    }
  }
  return selectors[0];
}

/**
 * Wait for a captain to see the approve button after an action.
 * This replaces reload-based detection with proper polling.
 */
export async function waitForCaptainApproveButton(
  page: Page,
  options: { timeout?: number } = {}
): Promise<boolean> {
  const timeout = options.timeout ?? TIMEOUTS.MEDIUM;

  try {
    await expect(
      page.locator('[data-testid="pending-action-approve-button"]')
    ).toBeVisible({ timeout });
    return true;
  } catch {
    return false;
  }
}

/**
 * Check if a page has the approve button visible (non-waiting).
 */
export async function hasApproveButton(page: Page): Promise<boolean> {
  const button = page.locator('[data-testid="pending-action-approve-button"]');
  return await button.isVisible().catch(() => false);
}

/**
 * Wait for state change after an action by polling for element updates.
 * Uses expect.poll to avoid arbitrary timeouts.
 */
export async function waitForStateUpdate(
  page: Page,
  checkFn: () => Promise<boolean>,
  options: { timeout?: number; message?: string } = {}
): Promise<void> {
  const { timeout = TIMEOUTS.MEDIUM, message = 'State did not update' } = options;

  await expect.poll(checkFn, {
    timeout,
    message,
    intervals: POLL_INTERVALS
  }).toBe(true);
}

/**
 * Wait for match options to be fully loaded (cards visible).
 */
export async function waitForMatchOptionsLoaded(
  page: Page,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.LONG;

  // Wait for at least one match option card to be visible
  await expect(
    page.locator('[data-testid="match-option-1"]')
  ).toBeVisible({ timeout });

  // Also wait for the content to be populated
  await expect(
    page.locator('[data-testid="match-option-1"]').locator('text=Blue Team')
  ).toBeVisible({ timeout: TIMEOUTS.SHORT });
}

/**
 * Wait for the lobby page to settle after navigation or reload.
 */
export async function waitForLobbyPageReady(
  page: Page,
  options: { timeout?: number } = {}
): Promise<void> {
  const timeout = options.timeout ?? TIMEOUTS.MEDIUM;

  // Wait for the main lobby heading
  await expect(page.locator('text=10-Man Lobby')).toBeVisible({ timeout });

  // Wait for at least one team column to be visible
  await expect(
    page.locator('[data-testid="team-column-blue"]').or(
      page.locator('[data-testid="team-column-red"]')
    ).first()
  ).toBeVisible({ timeout });
}
