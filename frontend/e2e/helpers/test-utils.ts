import { Page, expect } from '@playwright/test';

const API_BASE = 'http://localhost:9999/api/v1';

export interface TestUser {
  id: string;
  username: string;
  displayName: string;
  token: string;
}

/**
 * Register and login a test user, returns user info with token
 */
export async function createTestUser(page: Page, suffix: string): Promise<TestUser> {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(2, 8);
  const username = `testuser_${suffix}_${timestamp}_${random}`;
  const displayName = `Test${suffix}${random}`;
  const password = 'testpass123';

  // Register
  const registerRes = await page.request.post(`${API_BASE}/auth/register`, {
    data: { username, displayName, password },
  });
  expect(registerRes.ok()).toBeTruthy();
  const registerData = await registerRes.json();

  // The register response includes accessToken directly
  return {
    id: registerData.user.id,
    username,
    displayName,
    token: registerData.accessToken,
  };
}

/**
 * Login via UI
 */
export async function loginViaUI(page: Page, username: string, password: string) {
  await page.goto('/login');
  await page.fill('input[name="username"]', username);
  await page.fill('input[name="password"]', password);
  await page.click('button[type="submit"]');
  await page.waitForURL('/');
}

/**
 * Set auth token in localStorage to simulate logged-in state
 */
export async function setAuthToken(page: Page, user: TestUser) {
  await page.goto('/');
  await page.evaluate(({ token, user }) => {
    localStorage.setItem('accessToken', token);
    localStorage.setItem('user', JSON.stringify(user));
  }, { token: user.token, user: { id: user.id, username: user.username, displayName: user.displayName } });
  await page.reload();
}

/**
 * Create a lobby via API
 */
export async function createLobby(page: Page, token: string, options?: { timerDurationSeconds?: number }) {
  const res = await page.request.post(`${API_BASE}/lobbies`, {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      name: `Test Lobby ${Date.now()}`,
      draftMode: 'pro_play',
      timerDurationSeconds: options?.timerDurationSeconds ?? 30,
    },
  });
  expect(res.ok()).toBeTruthy();
  return await res.json();
}

/**
 * Join a lobby via API
 */
export async function joinLobby(page: Page, token: string, lobbyId: string) {
  const res = await page.request.post(`${API_BASE}/lobbies/${lobbyId}/join`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(res.ok()).toBeTruthy();
  return await res.json();
}

/**
 * Wait for element with text
 */
export async function waitForText(page: Page, text: string, timeout = 10000) {
  await page.waitForSelector(`text=${text}`, { timeout });
}

/**
 * Create a lobby with voting enabled via API
 */
export async function createLobbyWithVoting(
  page: Page,
  token: string,
  options?: {
    timerDurationSeconds?: number;
    votingMode?: 'majority' | 'captain_override';
  }
) {
  const res = await page.request.post(`${API_BASE}/lobbies`, {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      name: `Test Voting Lobby ${Date.now()}`,
      draftMode: 'pro_play',
      timerDurationSeconds: options?.timerDurationSeconds ?? 30,
      votingEnabled: true,
      votingMode: options?.votingMode ?? 'majority',
    },
  });
  expect(res.ok()).toBeTruthy();
  return await res.json();
}

/**
 * Initialize role profiles for a user via API
 */
export async function initializeRoleProfiles(page: Page, token: string) {
  const res = await page.request.post(`${API_BASE}/profile/roles/initialize`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  // Might already be initialized, so accept 400 as well
  return res.ok() || res.status() === 400;
}

/**
 * Generate teams for a lobby via API
 */
export async function generateTeams(page: Page, token: string, lobbyId: string) {
  const res = await page.request.post(`${API_BASE}/lobbies/${lobbyId}/generate-teams`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(res.ok()).toBeTruthy();
  return await res.json();
}

/**
 * Cast a vote for a match option via API
 */
export async function castVote(page: Page, token: string, lobbyId: string, optionNumber: number) {
  const res = await page.request.post(`${API_BASE}/lobbies/${lobbyId}/vote`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { optionNumber },
  });
  expect(res.ok()).toBeTruthy();
  return await res.json();
}

/**
 * Get voting status via API
 */
export async function getVotingStatus(page: Page, token: string, lobbyId: string) {
  const res = await page.request.get(`${API_BASE}/lobbies/${lobbyId}/voting-status`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(res.ok()).toBeTruthy();
  return await res.json();
}

/**
 * Kick a player from lobby via API
 */
export async function kickPlayer(page: Page, token: string, lobbyId: string, userId: string) {
  const res = await page.request.post(`${API_BASE}/lobbies/${lobbyId}/kick`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { userId },
  });
  expect(res.ok()).toBeTruthy();
  return await res.json();
}

/**
 * Get lobby data via API
 */
export async function getLobby(page: Page, token: string, lobbyId: string) {
  const res = await page.request.get(`${API_BASE}/lobbies/${lobbyId}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(res.ok()).toBeTruthy();
  return await res.json();
}
