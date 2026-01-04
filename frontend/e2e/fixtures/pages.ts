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
    await this.page.click('button:has-text("Generate Team Options")');
  }

  async expectGenerateTeamsButton() {
    await expect(this.page.locator('button:has-text("Generate Team Options")')).toBeVisible();
  }

  async expectGeneratingTeams() {
    await expect(this.page.locator('button:has-text("Generating Teams...")')).toBeVisible();
  }

  async selectOption(optionNumber: number) {
    // Click the "Select This Option" button for the specific option card
    const optionCard = this.page.locator(`[data-testid="match-option-${optionNumber}"]`);
    await optionCard.locator('button:has-text("Select This Option")').click();
  }

  async clickConfirmSelection() {
    await this.page.click('button:has-text("Confirm Selection")');
  }

  async clickStartDraft() {
    await this.page.click('button:has-text("Start Draft")');
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
    // Extra wait for state to fully settle
    await this.page.waitForTimeout(500);
  }

  async isSpectator(): Promise<boolean> {
    // Spectators don't see the Ready button after state sync
    // Use exact text match to avoid matching "Cancel Ready"
    const readyButton = this.page.locator('button:text-is("Ready")');
    return !(await readyButton.isVisible({ timeout: 3000 }).catch(() => false));
  }

  async canClickReady(): Promise<boolean> {
    // Check if Ready button is visible and clickable (non-spectator)
    // Use exact text match to avoid matching "Cancel Ready"
    const readyButton = this.page.locator('button:text-is("Ready")');
    return await readyButton.isVisible({ timeout: 3000 }).catch(() => false);
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
    ).toBeVisible({ timeout: 5000 });
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

  async waitForYourTurn() {
    // Wait until champion buttons become clickable
    await expect(
      this.page.locator('[data-testid="champion-grid"] button:not([disabled])').first()
    ).toBeVisible({ timeout: 30000 });
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
    await expect(this.page.locator('text=Draft Complete')).toBeVisible({ timeout: 30000 });
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
    // Select a champion and lock in
    // Fail fast if we can't complete within 5 seconds
    await this.selectChampionByIndex(championIndex);
    await this.page.waitForTimeout(300);

    // Verify Lock In button is enabled before clicking
    const lockInButton = this.page.locator('button:has-text("Lock In")');
    const isEnabled = await lockInButton.isEnabled({ timeout: 3000 }).catch(() => false);
    if (!isEnabled) {
      throw new Error(`Lock In button not enabled after selecting champion ${championIndex}`);
    }

    await this.clickLockIn();
    await this.page.waitForTimeout(500);
  }
}
