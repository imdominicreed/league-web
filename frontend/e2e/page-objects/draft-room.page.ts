import { expect } from '@playwright/test';
import { BasePage } from './base.page';
import { TIMEOUTS, POLL_INTERVALS } from '../helpers/wait-strategies';

/**
 * Draft Room page interactions
 */
export class DraftRoomPage extends BasePage {
  async goto(roomId: string) {
    await this.page.goto(`/draft/${roomId}`);
  }

  async waitForDraftLoaded() {
    await expect(
      this.page.locator('text=Waiting for Players').or(this.byTestIdOrText('draft-button-lock-in', 'Lock In'))
    ).toBeVisible({ timeout: TIMEOUTS.LONG });
  }

  async waitForWebSocketConnected() {
    await expect(this.byTestId('draft-disconnected')).not.toBeVisible({ timeout: TIMEOUTS.MEDIUM });
    await expect(this.page.locator('.text-lol-gold-light').first()).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  }

  async isSpectator(): Promise<boolean> {
    const readyButton = this.byTestIdOrText('draft-button-ready', 'Ready');
    const count = await readyButton.count();
    return count === 0;
  }

  async canClickReady(): Promise<boolean> {
    const readyButton = this.byTestIdOrText('draft-button-ready', 'Ready');
    const count = await readyButton.count();
    return count > 0 && (await readyButton.isVisible());
  }

  async waitForActiveState() {
    await expect(this.byTestIdOrText('draft-button-lock-in', 'Lock In')).toBeVisible({ timeout: TIMEOUTS.LONG });
    await this.waitForChampionsLoaded();
  }

  async waitForChampionsLoaded() {
    await expect(
      this.byTestId('champion-grid-items').locator('button img').first()
    ).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  }

  async waitForWaitingState() {
    await expect(this.page.locator('text=Waiting for Players')).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  }

  async isWaitingForPlayers(): Promise<boolean> {
    return this.page.locator('text=Waiting for Players').isVisible();
  }

  async clickReady() {
    await this.byTestIdOrText('draft-button-ready', 'Ready').click();
  }

  async clickStartDraft() {
    await this.byTestIdOrText('draft-button-start', 'Start Draft').click();
  }

  async expectStartDraftButton() {
    await expect(this.byTestIdOrText('draft-button-start', 'Start Draft')).toBeVisible({
      timeout: TIMEOUTS.MEDIUM,
    });
  }

  async selectChampion(championName: string) {
    const championButton = this.page.locator(`button:has(img[alt="${championName}"])`);
    await championButton.click({ force: true });
  }

  async selectChampionByIndex(index: number) {
    // Find buttons with images (champion buttons) in the champion grid items
    const championButtons = this.byTestId('champion-grid-items').locator('button:not([disabled])');

    // Wait for at least one enabled champion button
    await expect(championButtons.first()).toBeVisible({ timeout: TIMEOUTS.LONG });

    // Click the champion at the specified index
    // Use force: true because the img inside the button intercepts pointer events
    const targetButton = championButtons.nth(index);
    await targetButton.click({ force: true });
  }

  async clickLockIn() {
    await this.byTestIdOrText('draft-button-lock-in', 'Lock In').click();
  }

  async expectLockInEnabled() {
    const lockInButton = this.byTestIdOrText('draft-button-lock-in', 'Lock In');
    await expect(lockInButton).toBeEnabled();
  }

  async expectLockInDisabled() {
    const lockInButton = this.byTestIdOrText('draft-button-lock-in', 'Lock In');
    await expect(lockInButton).toBeDisabled();
  }

  async isYourTurn(): Promise<boolean> {
    try {
      // Check for enabled champion buttons specifically in the champion grid items
      // (not the filter buttons which are always enabled)
      const enabledButton = this.byTestId('champion-grid-items').locator('button:not([disabled])').first();
      await expect(enabledButton).toBeVisible({ timeout: 1000 });
      return true;
    } catch {
      return false;
    }
  }

  async waitForYourTurn(timeout: number = TIMEOUTS.LONG) {
    await expect(
      this.byTestId('champion-grid-items').locator('button:not([disabled])').first()
    ).toBeVisible({ timeout });
  }

  async waitForNotYourTurn() {
    await expect(
      this.byTestId('champion-grid-items').locator('button[disabled]').first()
    ).toBeVisible({ timeout: TIMEOUTS.LONG });
  }

  async expectPicksVisible() {
    await expect(this.page.locator('.text-lol-gold.font-beaufort').first()).toBeVisible();
  }

  async expectDraftComplete() {
    await expect(this.byTestIdOrText('draft-status-complete', 'Complete')).toBeVisible({ timeout: TIMEOUTS.LONG });
  }

  async getRoomCode(): Promise<string> {
    const codeElement = this.byTestIdOrText('draft-room-code', '');
    return (await codeElement.textContent()) || '';
  }

  async getYourSide(): Promise<'blue' | 'red' | 'spectator' | null> {
    return null;
  }

  async performBanOrPick(championIndex: number = 0) {
    const lockInButton = this.byTestIdOrText('draft-button-lock-in', 'Lock In');

    const bannedBefore = (await this.getBannedChampionNames()).length;
    const pickedBefore = await this.page
      .locator('.bg-lol-dark-blue .text-lol-gold.text-sm.font-beaufort')
      .count();

    // Wait for champion grid to be ready with enabled buttons
    const enabledButtons = this.byTestId('champion-grid-items').locator('button:not([disabled])');
    await expect(enabledButtons.first()).toBeVisible({ timeout: TIMEOUTS.LONG });

    // Try clicking champion and wait for Lock In to enable (retry if needed)
    for (let attempt = 0; attempt < 3; attempt++) {
      await this.selectChampionByIndex(championIndex + attempt);

      // Wait for Lock In to be enabled (indicates champion was selected)
      try {
        await expect(lockInButton).toBeEnabled({ timeout: 5000 });
        break; // Success, exit retry loop
      } catch {
        if (attempt === 2) throw new Error('Failed to select champion after 3 attempts');
        // Try next champion
      }
    }

    await lockInButton.click();

    await expect
      .poll(
        async () => {
          const bannedAfter = (await this.getBannedChampionNames()).length;
          const pickedAfter = await this.page
            .locator('.bg-lol-dark-blue .text-lol-gold.text-sm.font-beaufort')
            .count();
          return bannedAfter > bannedBefore || pickedAfter > pickedBefore;
        },
        { timeout: TIMEOUTS.MEDIUM, intervals: POLL_INTERVALS }
      )
      .toBe(true);
  }

  // ========== Edge Case Test Helpers ==========

  async isChampionDisabled(championName: string): Promise<boolean> {
    const championButton = this.byTestId('champion-grid-items').locator(`button:has(img[alt="${championName}"])`);
    const isDisabled = await championButton.getAttribute('disabled');
    return isDisabled !== null;
  }

  async isChampionDisabledByIndex(index: number): Promise<boolean> {
    const championButtons = this.byTestId('champion-grid-items').locator('button');
    const button = championButtons.nth(index);
    const isDisabled = await button.getAttribute('disabled');
    return isDisabled !== null;
  }

  async getTimerSeconds(): Promise<number> {
    // Use specific testid - don't use fallback text to avoid matching wrong elements
    const timerElement = this.byTestId('draft-timer-value');
    const timerText = await timerElement.textContent();
    return timerText ? parseInt(timerText, 10) : 0;
  }

  async getCurrentTeam(): Promise<string> {
    // Use specific testid - don't use fallback text to avoid matching wrong elements
    const teamElement = this.byTestId('draft-current-team');
    const teamText = await teamElement.textContent();
    return teamText || '';
  }

  async getCurrentAction(): Promise<string> {
    // Match the phase indicator banner: "🚫 Banning Phase" or "⚔️ Picking Phase"
    const actionText = await this.page.locator('.text-center .font-beaufort.uppercase').first().textContent();
    return actionText || '';
  }

  async waitForPhaseChange(fromAction: string, timeout: number = TIMEOUTS.LONG): Promise<void> {
    await expect(
      this.page.locator('.text-center .font-beaufort.uppercase').first()
    ).not.toHaveText(fromAction, { timeout });
  }

  async waitForTimerBelow(seconds: number, timeout: number = TIMEOUTS.LONG): Promise<void> {
    await expect
      .poll(
        async () => {
          const current = await this.getTimerSeconds();
          return current > 0 && current < seconds;
        },
        { timeout, intervals: POLL_INTERVALS }
      )
      .toBe(true);
  }

  async getBannedChampionNames(): Promise<string[]> {
    const banSlots = this.page.locator('.grayscale.opacity-60');
    const count = await banSlots.count();
    const names: string[] = [];
    for (let i = 0; i < count; i++) {
      const alt = await banSlots.nth(i).getAttribute('alt');
      if (alt) names.push(alt);
    }
    return names;
  }

  async getEnabledChampionCount(): Promise<number> {
    const enabledButtons = this.byTestId('champion-grid-items').locator('button:not([disabled])');
    return enabledButtons.count();
  }

  async getDisabledChampionCount(): Promise<number> {
    const disabledButtons = this.byTestId('champion-grid-items').locator('button[disabled]');
    return disabledButtons.count();
  }

  async isDraftComplete(): Promise<boolean> {
    const completeText = this.byTestIdOrText('draft-status-complete', 'Complete');
    const count = await completeText.count();
    return count > 0;
  }

  async reloadAndReconnect(): Promise<void> {
    await this.page.reload();
    await this.waitForDraftLoaded();
    await this.waitForWebSocketConnected();
  }

  // ========== Ban Slot Methods ==========

  async getBanSlot(side: 'blue' | 'red', index: number) {
    return this.byTestId(`draft-ban-slot-${side}-${index}`);
  }

  async getPickSlot(side: 'blue' | 'red', index: number) {
    return this.byTestId(`draft-pick-slot-${side}-${index}`);
  }

  async getTeamPanel(side: 'blue' | 'red') {
    return this.byTestId(`draft-team-panel-${side}`);
  }
}

/**
 * Wait for any of the given draft pages to get their turn.
 */
export async function waitForAnyTurn(
  draftPages: DraftRoomPage[],
  timeout: number = TIMEOUTS.MEDIUM
): Promise<DraftRoomPage> {
  return new Promise((resolve, reject) => {
    let resolved = false;
    let rejectionCount = 0;

    // Set up the timeout
    const timeoutId = setTimeout(() => {
      if (!resolved) {
        resolved = true;
        reject(new Error(`No page received turn within ${timeout}ms`));
      }
    }, timeout);

    // Try each page
    draftPages.forEach(async (draftPage) => {
      try {
        await draftPage.waitForYourTurn(timeout);
        if (!resolved) {
          resolved = true;
          clearTimeout(timeoutId);
          resolve(draftPage);
        }
      } catch {
        rejectionCount++;
        // If all pages rejected and we haven't resolved yet, reject
        if (rejectionCount === draftPages.length && !resolved) {
          resolved = true;
          clearTimeout(timeoutId);
          reject(new Error(`No page received turn within ${timeout}ms`));
        }
      }
    });
  });
}
