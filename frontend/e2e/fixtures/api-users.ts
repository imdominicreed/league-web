import { API_BASE, TEST_PASSWORD, generateTestUsername } from '../helpers/test-data';

/**
 * Simulated user - API only, no browser context
 */
export interface SimulatedUser {
  id: string;
  displayName: string;
  token: string;
}

/**
 * Fetch with retry and exponential backoff.
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
      if (response.ok || response.status < 500) {
        return response;
      }
      lastError = new Error(`HTTP ${response.status}: ${await response.text()}`);
    } catch (err) {
      lastError = err as Error;
    }

    if (attempt < maxRetries) {
      const delay = 500 * Math.pow(2, attempt - 1);
      await new Promise((r) => setTimeout(r, delay));
    }
  }

  throw lastError;
}

/**
 * Create a simulated user (API only, no browser)
 */
export async function createSimulatedUser(index: number): Promise<SimulatedUser> {
  const displayName = generateTestUsername(`sim${index}`);
  const response = await fetchWithRetry(`${API_BASE}/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ displayName, password: TEST_PASSWORD }),
  });

  if (!response.ok) {
    throw new Error(`Registration failed: ${response.status}`);
  }

  const data = await response.json();
  return {
    id: data.user.id,
    displayName: data.user.displayName,
    token: data.accessToken,
  };
}

/**
 * Create multiple simulated users in parallel
 */
export async function createSimulatedUsers(count: number, startIndex: number = 0): Promise<SimulatedUser[]> {
  return Promise.all(
    Array.from({ length: count }, (_, i) => createSimulatedUser(startIndex + i))
  );
}

/**
 * Join lobby as simulated user
 */
export async function joinLobbyAsSimulated(user: SimulatedUser, lobbyId: string): Promise<void> {
  const response = await fetchWithRetry(`${API_BASE}/lobbies/${lobbyId}/join`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${user.token}` },
  });

  if (!response.ok) {
    throw new Error(`Join lobby failed: ${response.status}`);
  }
}

/**
 * Set ready status for simulated user
 */
export async function setReadyAsSimulated(user: SimulatedUser, lobbyId: string, ready: boolean = true): Promise<void> {
  const response = await fetchWithRetry(`${API_BASE}/lobbies/${lobbyId}/ready`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${user.token}`,
    },
    body: JSON.stringify({ ready }),
  });

  if (!response.ok) {
    throw new Error(`Set ready failed: ${response.status}`);
  }
}

/**
 * Initialize role profiles for simulated user
 */
export async function initializeRoleProfilesAsSimulated(user: SimulatedUser): Promise<void> {
  const response = await fetchWithRetry(`${API_BASE}/profile/roles/initialize`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${user.token}` },
  });

  if (!response.ok) {
    throw new Error(`Initialize profiles failed: ${response.status}`);
  }
}

/**
 * Generate teams as simulated user (lobby creator)
 */
export async function generateTeamsAsSimulated(user: SimulatedUser, lobbyId: string): Promise<void> {
  const response = await fetchWithRetry(`${API_BASE}/lobbies/${lobbyId}/generate-teams`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${user.token}` },
  });

  if (!response.ok) {
    throw new Error(`Generate teams failed: ${response.status}`);
  }
}

/**
 * Select match option as simulated user
 */
export async function selectOptionAsSimulated(
  user: SimulatedUser,
  lobbyId: string,
  optionNumber: number
): Promise<void> {
  const response = await fetchWithRetry(`${API_BASE}/lobbies/${lobbyId}/select-option`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${user.token}`,
    },
    body: JSON.stringify({ optionNumber }),
  });

  if (!response.ok) {
    throw new Error(`Select option failed: ${response.status}`);
  }
}

/**
 * Start draft as simulated user
 */
export async function startDraftAsSimulated(user: SimulatedUser, lobbyId: string): Promise<string> {
  const response = await fetchWithRetry(`${API_BASE}/lobbies/${lobbyId}/start-draft`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${user.token}` },
  });

  if (!response.ok) {
    throw new Error(`Start draft failed: ${response.status}`);
  }

  const data = await response.json();
  return data.id;
}

/**
 * Approve pending action as simulated user
 */
export async function approvePendingActionAsSimulated(
  user: SimulatedUser,
  lobbyId: string,
  actionId: string
): Promise<void> {
  const response = await fetchWithRetry(`${API_BASE}/lobbies/${lobbyId}/pending-action/${actionId}/approve`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${user.token}` },
  });

  if (!response.ok) {
    throw new Error(`Approve pending action failed: ${response.status}`);
  }
}

/**
 * Get pending action for lobby
 */
export async function getPendingActionAsSimulated(
  user: SimulatedUser,
  lobbyId: string
): Promise<{ id: string; actionType: string } | null> {
  const response = await fetch(`${API_BASE}/lobbies/${lobbyId}/pending-action`, {
    method: 'GET',
    headers: { Authorization: `Bearer ${user.token}` },
  });

  if (!response.ok) {
    return null;
  }

  return response.json();
}

/**
 * Wait for and approve pending action
 */
export async function waitForAndApprovePendingAction(
  user: SimulatedUser,
  lobbyId: string,
  timeout: number = 10000
): Promise<boolean> {
  const startTime = Date.now();

  while (Date.now() - startTime < timeout) {
    const pendingAction = await getPendingActionAsSimulated(user, lobbyId);
    if (pendingAction) {
      await approvePendingActionAsSimulated(user, lobbyId, pendingAction.id);
      return true;
    }
    await new Promise((r) => setTimeout(r, 500));
  }

  return false;
}
