import { Page, expect } from '@playwright/test';

const API_BASE = 'http://localhost:9999/api/v1';

/**
 * Home page interactions
 */
export class HomePage {
  constructor(private page: Page) {}

  async goto() {
    await this.page.goto('/');
  }

  async clickLogin() {
    await this.page.click('a:has-text("Login")');
    await this.page.waitForURL('/login');
  }

  async clickRegister() {
    await this.page.click('a:has-text("Register")');
    await this.page.waitForURL('/register');
  }

  async clickCreateDraftRoom() {
    await this.page.click('a:has-text("Create Draft Room")');
    await this.page.waitForURL('/create');
  }

  async clickJoinRoom() {
    await this.page.click('a:has-text("Join Room")');
    await this.page.waitForURL('/join');
  }

  async clickMyProfile() {
    await this.page.click('a:has-text("My Profile")');
    await this.page.waitForURL('/profile');
  }

  async clickCreateLobby() {
    await this.page.click('a:has-text("Create 10-Man Lobby")');
    await this.page.waitForURL('/create-lobby');
  }

  async expectAuthenticated(displayName: string) {
    await expect(this.page.locator(`text=Welcome, ${displayName}`)).toBeVisible();
  }

  async expectUnauthenticated() {
    await expect(this.page.locator('a:has-text("Login")')).toBeVisible();
    await expect(this.page.locator('a:has-text("Register")')).toBeVisible();
  }

  async expectAuthenticatedMenu() {
    await expect(this.page.locator('a:has-text("Create Draft Room")')).toBeVisible();
    await expect(this.page.locator('a:has-text("Join Room")')).toBeVisible();
    await expect(this.page.locator('a:has-text("My Profile")')).toBeVisible();
    await expect(this.page.locator('a:has-text("Create 10-Man Lobby")')).toBeVisible();
  }
}

/**
 * Login page interactions
 */
export class LoginPage {
  constructor(private page: Page) {}

  async goto() {
    await this.page.goto('/login');
  }

  async fillUsername(username: string) {
    await this.page.fill('#displayName', username);
  }

  async fillPassword(password: string) {
    await this.page.fill('#password', password);
  }

  async submit() {
    await this.page.click('button:has-text("Login")');
  }

  async login(username: string, password: string) {
    await this.fillUsername(username);
    await this.fillPassword(password);
    await this.submit();
  }

  async expectError(errorText?: string) {
    const errorBox = this.page.locator('.bg-red-500\\/20');
    await expect(errorBox).toBeVisible();
    if (errorText) {
      await expect(errorBox).toContainText(errorText);
    }
  }

  async expectLoading() {
    await expect(this.page.locator('button:has-text("Logging in...")')).toBeVisible();
  }
}

/**
 * Register page interactions
 */
export class RegisterPage {
  constructor(private page: Page) {}

  async goto() {
    await this.page.goto('/register');
  }

  async fillUsername(username: string) {
    await this.page.fill('#displayName', username);
  }

  async fillPassword(password: string) {
    await this.page.fill('#password', password);
  }

  async submit() {
    await this.page.click('button:has-text("Register")');
  }

  async register(username: string, password: string) {
    await this.fillUsername(username);
    await this.fillPassword(password);
    await this.submit();
  }

  async expectError(errorText?: string) {
    const errorBox = this.page.locator('.bg-red-500\\/20');
    await expect(errorBox).toBeVisible();
    if (errorText) {
      await expect(errorBox).toContainText(errorText);
    }
  }
}

/**
 * Create Lobby page interactions
 */
export class CreateLobbyPage {
  constructor(private page: Page) {}

  async goto() {
    await this.page.goto('/create-lobby');
  }

  async selectDraftMode(mode: 'pro_play' | 'fearless') {
    await this.page.selectOption('select', mode);
  }

  async setTimerDuration(seconds: number) {
    await this.page.fill('input[type="number"]', String(seconds));
  }

  async submit() {
    await this.page.click('button:has-text("Create Lobby")');
  }

  async createLobby(mode: 'pro_play' | 'fearless' = 'pro_play', timerSeconds: number = 30) {
    await this.selectDraftMode(mode);
    await this.setTimerDuration(timerSeconds);
    await this.submit();
  }
}

/**
 * Lobby Room page interactions
 */
export class LobbyRoomPage {
  constructor(private page: Page) {}

  async goto(lobbyId: string) {
    await this.page.goto(`/lobby/${lobbyId}`);
  }

  async expectLobbyCode(code: string) {
    await expect(this.page.locator(`text=Code: ${code}`)).toBeVisible();
  }

  async expectPlayerCount(current: number, total: number = 10) {
    await expect(this.page.locator(`text=${current}/${total}`)).toBeVisible();
  }

  async clickReadyUp() {
    await this.page.click('button:has-text("Ready Up")');
  }

  async clickCancelReady() {
    await this.page.click('button:has-text("Cancel Ready")');
  }

  async expectReadyButton() {
    await expect(this.page.locator('button:has-text("Ready Up")')).toBeVisible();
  }

  async expectCancelReadyButton() {
    await expect(this.page.locator('button:has-text("Cancel Ready")')).toBeVisible();
  }

  async clickGenerateTeams() {
    // Try creator-only button first (for creator), then captain proposal
    const creatorButton = this.page.locator('button:has-text("Generate Team Options (Creator Only)")');
    if (await creatorButton.count() > 0 && await creatorButton.isVisible()) {
      await creatorButton.click();
    } else {
      // Fall back to any generate button
      await this.page.click('button:has-text("Generate Team Options")');
    }
  }

  async expectGenerateTeamsButton() {
    // Accept either the creator-only button or any generate button
    await expect(
      this.page.locator('button:has-text("Generate Team Options")')
    ).toBeVisible();
  }

  async expectGeneratingTeams() {
    await expect(this.page.locator('button:has-text("Generating Teams...")')).toBeVisible();
  }

  async selectOption(optionNumber: number) {
    // Click the match option card to select it
    // The button inside is just a visual indicator - the card div has the onClick
    const optionCard = this.page.locator(`[data-testid="match-option-${optionNumber}"]`);
    await optionCard.click();
  }

  async clickConfirmSelection() {
    await this.page.click('button:has-text("Confirm Selection")');
  }

  async clickStartDraft() {
    // Try the creator-only button first
    const creatorButton = this.page.locator('button:has-text("Start Draft (Creator Only)")');
    if (await creatorButton.count() > 0 && await creatorButton.isVisible()) {
      await creatorButton.click();
    } else {
      // Fall back to propose start draft or any start draft button
      const proposeButton = this.page.locator('button:has-text("Propose Start Draft")');
      if (await proposeButton.count() > 0 && await proposeButton.isVisible()) {
        await proposeButton.click();
      } else {
        await this.page.click('button:has-text("Start Draft")');
      }
    }
  }

  async expectStartDraftButton() {
    // Check for either the creator-only button or the propose button (use first() to handle both being visible)
    await expect(
      this.page.locator('button:has-text("Start Draft (Creator Only)")').or(
        this.page.locator('button:has-text("Propose Start Draft")')
      ).first()
    ).toBeVisible({ timeout: 10000 });
  }

  async expectOnDraftPage() {
    await this.page.waitForURL(/\/draft\//);
  }

  async waitForMatchOptions() {
    await expect(this.page.locator('text=Option 1')).toBeVisible({ timeout: 30000 });
  }

  async leave() {
    await this.page.click('a:has-text("Leave")');
    await this.page.waitForURL('/');
  }

  // Captain-related methods
  async isCaptain(): Promise<boolean> {
    // Check if the Captain badge is visible in controls
    const captainBadge = this.page.locator('text=Captain').first();
    return await captainBadge.count() > 0;
  }

  async clickTakeCaptain() {
    await this.page.click('button:has-text("Take Captain")');
  }

  async expectTakeCaptainButton() {
    await expect(this.page.locator('button:has-text("Take Captain")')).toBeVisible();
  }

  async clickProposeMatchmake() {
    await this.page.click('button:has-text("Propose Matchmake")');
  }

  async clickProposeStartDraft() {
    await this.page.click('button:has-text("Propose Start Draft")');
  }

  async expectPendingActionBanner() {
    // Check for the pending action banner (yellow background)
    await expect(this.page.locator('.bg-yellow-900\\/30')).toBeVisible();
  }

  async clickApprovePendingAction() {
    await this.page.click('button:has-text("Approve")');
  }

  async clickCancelPendingAction() {
    await this.page.click('button:has-text("Cancel")');
  }

  async expectTeamColumn(side: 'blue' | 'red') {
    const teamText = side === 'blue' ? 'Blue Team' : 'Red Team';
    await expect(this.page.locator(`text=${teamText}`)).toBeVisible();
  }

  // ========== Captain Modal Methods ==========

  async clickPromoteCaptain() {
    await this.page.click('button:has-text("Promote Captain")');
  }

  async clickKickPlayer() {
    await this.page.click('button:has-text("Kick Player")');
  }

  async clickProposeSwap() {
    await this.page.click('button:has-text("Propose Swap")');
  }

  async selectPlayerInModal(displayName: string) {
    // Click a player button in promote/kick modal
    await this.page.click(`button:has-text("${displayName}")`);
  }

  async cancelModal() {
    // Click Cancel button to close any open modal
    // Use last() since there may be multiple Cancel buttons (modal + controls)
    const cancelButtons = this.page.locator('button:has-text("Cancel")');
    const count = await cancelButtons.count();
    if (count > 0) {
      await cancelButtons.last().click();
    }
  }

  // ========== Swap Modal Configuration ==========

  async selectSwapType(type: 'players' | 'roles') {
    if (type === 'players') {
      await this.page.click('button:has-text("Between Teams")');
    } else {
      await this.page.click('button:has-text("Swap Roles")');
    }
  }

  async selectPlayer1InSwap(displayName: string) {
    // First select dropdown - find option containing the display name
    const select1 = this.page.locator('select').first();
    const options = await select1.locator('option').allTextContents();
    const matchingOption = options.find((opt) => opt.includes(displayName));
    if (matchingOption) {
      await select1.selectOption({ label: matchingOption });
    }
  }

  async selectPlayer2InSwap(displayName: string) {
    // Second select dropdown - find option containing the display name
    const select2 = this.page.locator('select').nth(1);
    const options = await select2.locator('option').allTextContents();
    const matchingOption = options.find((opt) => opt.includes(displayName));
    if (matchingOption) {
      await select2.selectOption({ label: matchingOption });
    }
  }

  async confirmSwapProposal() {
    // Click Propose button inside the swap modal (modal has bg-lol-gold class)
    const modal = this.page.locator('.fixed.inset-0.bg-black\\/70');
    await modal.locator('button.bg-lol-gold:has-text("Propose")').click();
  }

  // ========== Pending Action Verification ==========

  async expectNoPendingActionBanner() {
    await expect(this.page.locator('.bg-yellow-900\\/30')).not.toBeVisible();
  }

  async getPendingActionType(): Promise<string> {
    const label = this.page.locator('.bg-yellow-900\\/30 .text-yellow-400.font-semibold');
    return (await label.textContent()) || '';
  }

  async expectApproveButton() {
    await expect(this.page.locator('button:has-text("Approve")')).toBeVisible();
  }

  async expectApprovedBadge() {
    await expect(this.page.locator('.text-green-400:has-text("Approved")')).toBeVisible();
  }

  // ========== Player Verification ==========

  async expectPlayerOnTeam(displayName: string, team: 'blue' | 'red') {
    const teamSection = team === 'blue'
      ? this.page.locator('.bg-blue-900\\/30, [class*="blue"]').first()
      : this.page.locator('.bg-red-900\\/30, [class*="red"]').first();
    await expect(teamSection.locator(`text=${displayName}`)).toBeVisible();
  }

  async expectPlayerNotInLobby(displayName: string) {
    // Check specifically in the team column containers (not match options)
    // TeamColumn root has bg-blue-900/30 or bg-red-900/30 with an h3 header
    // MatchOptionCard has h4 headers, so we specifically target TeamColumn
    const blueTeamColumn = this.page.locator('.bg-blue-900\\/30').filter({ has: this.page.locator('h3') });
    const redTeamColumn = this.page.locator('.bg-red-900\\/30').filter({ has: this.page.locator('h3') });

    // Check both team columns don't contain the player name
    await expect(blueTeamColumn.locator(`text=${displayName}`)).not.toBeVisible();
    await expect(redTeamColumn.locator(`text=${displayName}`)).not.toBeVisible();
  }

  async expectCaptainControls() {
    await expect(this.page.locator('text=Captain Controls')).toBeVisible();
  }

  async expectPlayerActions() {
    await expect(this.page.locator('text=Player Actions')).toBeVisible();
  }
}

/**
 * Helper to register a user via API and return credentials
 */
export async function registerUserViaApi(
  displayName: string,
  password: string
): Promise<{ userId: string; token: string }> {
  const response = await fetch(`${API_BASE}/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ displayName, password }),
  });

  if (!response.ok) {
    throw new Error(`Registration failed: ${response.status}`);
  }

  const data = await response.json();
  return {
    userId: data.user.id,
    token: data.accessToken,
  };
}

/**
 * Helper to create a lobby via API
 */
export async function createLobbyViaApi(
  token: string,
  draftMode: string = 'pro_play',
  timerDurationSeconds: number = 30
): Promise<{ id: string; shortCode: string }> {
  const response = await fetch(`${API_BASE}/lobbies`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ draftMode, timerDurationSeconds }),
  });

  if (!response.ok) {
    throw new Error(`Create lobby failed: ${response.status}`);
  }

  return response.json();
}

/**
 * Helper to join a lobby via API
 */
export async function joinLobbyViaApi(token: string, lobbyId: string): Promise<void> {
  const response = await fetch(`${API_BASE}/lobbies/${lobbyId}/join`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error(`Join lobby failed: ${response.status}`);
  }
}

/**
 * Helper to set ready status via API
 */
export async function setReadyViaApi(token: string, lobbyId: string, ready: boolean): Promise<void> {
  const response = await fetch(`${API_BASE}/lobbies/${lobbyId}/ready`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ ready }),
  });

  if (!response.ok) {
    throw new Error(`Set ready failed: ${response.status}`);
  }
}

/**
 * Generate a unique test username
 */
export function generateTestUsername(prefix: string = 'e2e'): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(2, 6);
  return `${prefix}_${timestamp}_${random}`;
}

/**
 * Set auth token in page localStorage
 */
export async function setAuthToken(page: Page, token: string): Promise<void> {
  await page.evaluate((accessToken) => {
    localStorage.setItem('accessToken', accessToken);
  }, token);
  await page.reload();
}

/**
 * Draft Room page interactions
 */
export class DraftRoomPage {
  constructor(public readonly page: Page) {}

  async goto(roomId: string) {
    await this.page.goto(`/draft/${roomId}`);
  }

  async waitForDraftLoaded() {
    // Wait for either waiting state or active draft
    await expect(
      this.page.locator('text=Waiting for Players').or(this.page.locator('text=Lock In'))
    ).toBeVisible({ timeout: 30000 });
  }

  async waitForWebSocketConnected() {
    // Wait for WebSocket to connect by checking that "Disconnected" is not visible
    // and that player names appear in team panels
    await expect(this.page.locator('text=Disconnected')).not.toBeVisible({ timeout: 10000 });
    // Wait for at least one player name to appear (indicates state sync)
    await expect(this.page.locator('.text-lol-gold-light').first()).toBeVisible({ timeout: 10000 });
  }

  async isSpectator(): Promise<boolean> {
    // Spectators don't see the Ready button after state sync
    // Use count() which returns 0 without throwing if element doesn't exist
    const readyButton = this.page.locator('button:text-is("Ready")');
    const count = await readyButton.count();
    return count === 0;
  }

  async canClickReady(): Promise<boolean> {
    // Check if Ready button is visible and clickable (non-spectator)
    // Use count() to check existence without throwing
    const readyButton = this.page.locator('button:text-is("Ready")');
    const count = await readyButton.count();
    return count > 0 && (await readyButton.isVisible());
  }

  async waitForActiveState() {
    // Wait for the champion grid (Lock In button) to appear
    await expect(this.page.locator('button:has-text("Lock In")')).toBeVisible({ timeout: 30000 });
    // Also wait for champions to be loaded in the grid
    await this.waitForChampionsLoaded();
  }

  async waitForChampionsLoaded() {
    // Wait for at least one champion image to appear in the grid
    await expect(
      this.page.locator('[data-testid="champion-grid"] button img').first()
    ).toBeVisible({ timeout: 15000 });
  }

  async waitForWaitingState() {
    await expect(this.page.locator('text=Waiting for Players')).toBeVisible({ timeout: 15000 });
  }

  async isWaitingForPlayers(): Promise<boolean> {
    return this.page.locator('text=Waiting for Players').isVisible();
  }

  async clickReady() {
    await this.page.click('button:text-is("Ready")');
  }

  async clickStartDraft() {
    await this.page.click('button:has-text("Start Draft")');
  }

  async expectStartDraftButton() {
    await expect(this.page.locator('button:has-text("Start Draft")')).toBeVisible({
      timeout: 10000,
    });
  }

  async selectChampion(championName: string) {
    // Click on a champion in the grid by name
    // Use force:true because the img element intercepts pointer events
    const championButton = this.page.locator(`button:has(img[alt="${championName}"])`);
    await championButton.click({ force: true });
  }

  async selectChampionByIndex(index: number) {
    // Select the nth available champion in the grid
    // Use force:true because the img element intercepts pointer events
    const championButtons = this.page.locator(
      '[data-testid="champion-grid"] button:not([disabled])'
    );
    await championButtons.nth(index).click({ force: true });
  }

  async clickLockIn() {
    await this.page.click('button:has-text("Lock In")');
  }

  async expectLockInEnabled() {
    const lockInButton = this.page.locator('button:has-text("Lock In")');
    await expect(lockInButton).toBeEnabled();
  }

  async expectLockInDisabled() {
    const lockInButton = this.page.locator('button:has-text("Lock In")');
    await expect(lockInButton).toBeDisabled();
  }

  async isYourTurn(): Promise<boolean> {
    // When it's your turn, champion buttons are clickable (not disabled)
    const championButtons = this.page.locator('[data-testid="champion-grid"] button');
    const firstButton = championButtons.first();
    const isChampDisabled = await firstButton.getAttribute('disabled');
    return isChampDisabled === null;
  }

  async waitForYourTurn(timeout: number = 30000) {
    // Wait until champion buttons become clickable
    await expect(
      this.page.locator('[data-testid="champion-grid"] button:not([disabled])').first()
    ).toBeVisible({ timeout });
  }

  async waitForNotYourTurn() {
    // Wait until champion buttons become disabled (opponent's turn)
    await expect(
      this.page.locator('[data-testid="champion-grid"] button[disabled]').first()
    ).toBeVisible({ timeout: 30000 });
  }

  async expectPicksVisible() {
    // Check that team panels show picked champions
    await expect(this.page.locator('.text-lol-gold.font-beaufort').first()).toBeVisible();
  }

  async expectDraftComplete() {
    // BanBar shows "Complete" (styled as uppercase "COMPLETE" via CSS)
    await expect(this.page.locator('text=Complete')).toBeVisible({ timeout: 30000 });
  }

  async getRoomCode(): Promise<string> {
    const codeElement = this.page.locator('text=Room:').locator('span.font-mono');
    return (await codeElement.textContent()) || '';
  }

  async getYourSide(): Promise<'blue' | 'red' | 'spectator' | null> {
    // Determine side by looking at team panel highlighting or other indicators
    // This is simplified - in practice we'd need to check the actual UI state
    return null;
  }

  async performBanOrPick(championIndex: number = 0) {
    const lockInButton = this.page.locator('button:has-text("Lock In")');

    // Get state before action for verification
    const bannedBefore = (await this.getBannedChampionNames()).length;
    // Count picked champions by looking at champion name labels in team panels
    // (text-lol-gold.text-sm.font-beaufort elements are champion names shown after picks)
    const pickedBefore = await this.page
      .locator('.bg-lol-dark-blue .text-lol-gold.text-sm.font-beaufort')
      .count();

    // Select champion - Playwright auto-retries clicks
    await this.selectChampionByIndex(championIndex);

    // Wait for Lock In to enable (confirms WebSocket registered selection)
    await expect(lockInButton).toBeEnabled({ timeout: 5000 });

    // Click Lock In
    await lockInButton.click();

    // Wait for phase to ACTUALLY advance by checking either:
    // 1. Ban count increases (ban phase)
    // 2. Pick count increases (pick phase)
    // This handles consecutive same-side phases (e.g., R-R in B-R-R-B pick sequence)
    await expect
      .poll(
        async () => {
          const bannedAfter = (await this.getBannedChampionNames()).length;
          const pickedAfter = await this.page
            .locator('.bg-lol-dark-blue .text-lol-gold.text-sm.font-beaufort')
            .count();
          return bannedAfter > bannedBefore || pickedAfter > pickedBefore;
        },
        { timeout: 10000, intervals: [200, 500, 1000] }
      )
      .toBe(true);
  }

  // ========== Edge Case Test Helpers ==========

  async isChampionDisabled(championName: string): Promise<boolean> {
    // Check if a specific champion button is disabled (banned or picked)
    const championButton = this.page.locator(
      `[data-testid="champion-grid"] button:has(img[alt="${championName}"])`
    );
    const isDisabled = await championButton.getAttribute('disabled');
    return isDisabled !== null;
  }

  async isChampionDisabledByIndex(index: number): Promise<boolean> {
    // Check if the nth champion button is disabled
    const championButtons = this.page.locator('[data-testid="champion-grid"] button');
    const button = championButtons.nth(index);
    const isDisabled = await button.getAttribute('disabled');
    return isDisabled !== null;
  }

  async getTimerSeconds(): Promise<number> {
    // Get the timer value from the BanBar (shows as large number in center)
    const timerText = await this.page.locator('.font-beaufort.text-3xl').textContent();
    return timerText ? parseInt(timerText, 10) : 0;
  }

  async getCurrentAction(): Promise<string> {
    // Get current action text like "Blue Ban" or "Red Pick"
    // This is the small text below the timer in the BanBar's center section
    // Use .text-center to target only the timer section, not team headers
    const actionText = await this.page.locator('.text-center .text-xs.uppercase.tracking-wider').textContent();
    return actionText || '';
  }

  async waitForPhaseChange(fromAction: string, timeout: number = 30000): Promise<void> {
    // Wait for the action text to change from the current action
    await expect(
      this.page.locator('.text-center .text-xs.uppercase.tracking-wider')
    ).not.toHaveText(fromAction, { timeout });
  }

  async waitForTimerBelow(seconds: number, timeout: number = 30000): Promise<void> {
    // Wait until timer drops below a certain value using Playwright's auto-retry
    await expect
      .poll(
        async () => {
          const current = await this.getTimerSeconds();
          return current > 0 && current <= seconds;
        },
        { timeout, intervals: [500, 1000, 1000] }
      )
      .toBe(true);
  }

  async getBannedChampionNames(): Promise<string[]> {
    // Get all banned champion names from the ban bar
    // Champions in ban bar have grayscale + red X overlay
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
    // Count how many champions are currently enabled (clickable)
    const enabledButtons = this.page.locator(
      '[data-testid="champion-grid"] button:not([disabled])'
    );
    return enabledButtons.count();
  }

  async getDisabledChampionCount(): Promise<number> {
    // Count how many champions are currently disabled
    const disabledButtons = this.page.locator(
      '[data-testid="champion-grid"] button[disabled]'
    );
    return disabledButtons.count();
  }

  async isDraftComplete(): Promise<boolean> {
    // Check if "Complete" text is shown in the timer area
    // Use count() to check existence without throwing
    const completeText = this.page.locator('text=Complete');
    const count = await completeText.count();
    return count > 0;
  }

  async reloadAndReconnect(): Promise<void> {
    // Reload the page and wait for WebSocket reconnection
    await this.page.reload();
    await this.waitForDraftLoaded();
    await this.waitForWebSocketConnected();
  }
}

/**
 * Wait for any of the given draft pages to get their turn.
 * Uses Promise.race with proper Playwright waits instead of manual polling.
 */
export async function waitForAnyTurn(
  draftPages: DraftRoomPage[],
  timeout: number = 15000
): Promise<DraftRoomPage> {
  // Create a promise for each page that resolves when it gets its turn
  const turnPromises = draftPages.map(async (draftPage, index) => {
    try {
      await draftPage.waitForYourTurn(timeout);
      return { draftPage, index };
    } catch {
      // This page didn't get turn within timeout - return never-resolving promise
      // so Promise.race continues waiting for others
      return new Promise<never>(() => {});
    }
  });

  // Race all pages - first to get their turn wins
  const result = await Promise.race(turnPromises);

  if (!result || !result.draftPage) {
    throw new Error(`No page received turn within ${timeout}ms`);
  }

  return result.draftPage;
}
