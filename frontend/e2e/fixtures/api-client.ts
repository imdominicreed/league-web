import { Page } from '@playwright/test';
import { API_BASE, TEST_PASSWORD, generateTestUsername } from '../helpers/test-data';

/**
 * Fetch with retry and exponential backoff.
 * Retries on network errors and 5xx server errors.
 */
async function fetchWithRetry(
  url: string,
  options: RequestInit,
  maxRetries: number = 3
): Promise<Response> {
  let lastError: Error | null = null;

  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      const response = await fetch(url, options);
      // Only retry on 5xx errors, not 4xx
      if (response.ok || response.status < 500) {
        return response;
      }
      lastError = new Error(`HTTP ${response.status}: ${await response.text()}`);
    } catch (err) {
      lastError = err as Error;
    }

    if (attempt < maxRetries) {
      // Exponential backoff: 500ms, 1000ms, 2000ms
      const delay = 500 * Math.pow(2, attempt - 1);
      await new Promise((r) => setTimeout(r, delay));
    }
  }

  throw lastError;
}

/**
 * User registration response
 */
export interface RegisterResponse {
  userId: string;
  token: string;
  displayName: string;
}

/**
 * Lobby response
 */
export interface LobbyResponse {
  id: string;
  shortCode: string;
}

/**
 * Match option from matchmaking
 */
export interface MatchOption {
  optionNumber: number;
  algorithmType: string;
  balanceScore: number;
  blueTeamAvgMmr: number;
  redTeamAvgMmr: number;
}

/**
 * Register a user via API and return credentials.
 */
export async function registerUserViaApi(
  displayName: string,
  password: string = TEST_PASSWORD
): Promise<RegisterResponse> {
  const response = await fetchWithRetry(`${API_BASE}/auth/register`, {
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
    displayName,
  };
}

/**
 * Create a lobby via API.
 */
export async function createLobbyViaApi(
  token: string,
  draftMode: string = 'pro_play',
  timerDurationSeconds: number = 90
): Promise<LobbyResponse> {
  const response = await fetchWithRetry(`${API_BASE}/lobbies`, {
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
 * Join a lobby via API.
 */
export async function joinLobbyViaApi(token: string, lobbyId: string): Promise<void> {
  const response = await fetchWithRetry(`${API_BASE}/lobbies/${lobbyId}/join`, {
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
 * Set ready status via API.
 */
export async function setReadyViaApi(
  token: string,
  lobbyId: string,
  ready: boolean
): Promise<void> {
  const response = await fetchWithRetry(`${API_BASE}/lobbies/${lobbyId}/ready`, {
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
 * Generate team options via API.
 */
export async function generateTeamsViaApi(token: string, lobbyId: string): Promise<MatchOption[]> {
  const response = await fetchWithRetry(`${API_BASE}/lobbies/${lobbyId}/generate-teams`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error(`Generate teams failed: ${response.status}`);
  }

  return response.json();
}

/**
 * Select a match option via API.
 */
export async function selectOptionViaApi(
  token: string,
  lobbyId: string,
  optionNumber: number
): Promise<void> {
  const response = await fetchWithRetry(`${API_BASE}/lobbies/${lobbyId}/select-option`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ optionNumber }),
  });

  if (!response.ok) {
    throw new Error(`Select option failed: ${response.status}`);
  }
}

/**
 * Start draft via API.
 */
export async function startDraftViaApi(token: string, lobbyId: string): Promise<string> {
  const response = await fetchWithRetry(`${API_BASE}/lobbies/${lobbyId}/start-draft`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error(`Start draft failed: ${response.status}`);
  }

  const data = await response.json();
  return data.roomId;
}

/**
 * Set auth token in page localStorage.
 * Note: This requires a page reload to take effect.
 */
export async function setAuthToken(page: Page, token: string): Promise<void> {
  await page.evaluate((accessToken) => {
    localStorage.setItem('accessToken', accessToken);
  }, token);
}

/**
 * Set auth token and reload the page.
 * @deprecated Use setAuthTokenWithoutReload for better performance
 */
export async function setAuthTokenAndReload(page: Page, token: string): Promise<void> {
  await setAuthToken(page, token);
  await page.reload();
}

/**
 * Create an authenticated user and return credentials.
 * Convenience wrapper around registerUserViaApi with auto-generated username.
 */
export async function createTestUser(
  prefix: string = 'e2e'
): Promise<RegisterResponse> {
  const displayName = generateTestUsername(prefix);
  return registerUserViaApi(displayName);
}
