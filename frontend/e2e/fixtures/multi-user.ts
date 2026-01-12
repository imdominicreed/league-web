import { test as base, BrowserContext, Page } from '@playwright/test';

const API_BASE = 'http://localhost:9999/api/v1';

export interface TestUser {
  id: string;
  displayName: string;
  token: string;
}

export interface UserSession {
  context: BrowserContext;
  page: Page;
  user: TestUser;
  token: string;
}

export interface Lobby {
  id: string;
  shortCode: string;
  createdBy: string;
  status: string;
  selectedMatchOption: number | null;
  draftMode: string;
  timerDurationSeconds: number;
  roomId: string | null;
  players: LobbyPlayer[];
}

export interface LobbyPlayer {
  id: string;
  userId: string;
  displayName: string;
  team: string | null;
  assignedRole: string | null;
  isReady: boolean;
  isCaptain: boolean;
  joinOrder: number;
}

export interface MatchOption {
  optionNumber: number;
  blueTeamAvgMmr: number;
  redTeamAvgMmr: number;
  mmrDifference: number;
  balanceScore: number;
  assignments: MatchAssignment[];
}

export interface MatchAssignment {
  userId: string;
  displayName: string;
  team: string;
  assignedRole: string;
  roleMmr: number;
  comfortRating: number;
}

export interface LobbyWithUsers {
  lobby: Lobby;
  users: UserSession[];
}

/**
 * API helper class for making authenticated requests with retry logic
 */
class ApiClient {
  constructor(private token?: string) {}

  async request<T>(
    endpoint: string,
    options: { method?: string; body?: unknown; maxRetries?: number } = {}
  ): Promise<T> {
    const { method = 'GET', body, maxRetries = 3 } = options;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    let lastError: Error | null = null;

    for (let attempt = 1; attempt <= maxRetries; attempt++) {
      try {
        const response = await fetch(`${API_BASE}${endpoint}`, {
          method,
          headers,
          body: body ? JSON.stringify(body) : undefined,
        });

        // Only retry on 5xx errors, not 4xx
        if (response.ok || response.status < 500) {
          if (!response.ok) {
            const error = await response.text();
            throw new Error(`API Error ${response.status}: ${error}`);
          }
          return response.json();
        }

        const errorText = await response.text();
        lastError = new Error(`API Error ${response.status}: ${errorText}`);
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

  async get<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint);
  }

  async post<T>(endpoint: string, body?: unknown): Promise<T> {
    return this.request<T>(endpoint, { method: 'POST', body });
  }
}

/**
 * Register a new user via the API
 */
async function registerUser(displayName: string, password: string): Promise<{ user: TestUser; token: string }> {
  const client = new ApiClient();
  const response = await client.post<{
    user: { id: string; displayName: string };
    accessToken: string;
    refreshToken: string;
  }>('/auth/register', { displayName, password });

  return {
    user: {
      id: response.user.id,
      displayName: response.user.displayName,
      token: response.accessToken,
    },
    token: response.accessToken,
  };
}

/**
 * Generate a unique username for testing
 */
function generateUsername(index: number): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(2, 8); // 6 chars for uniqueness
  return `e2e_${index}_${timestamp}_${random}`;
}

/**
 * Extended test fixture with multi-user support
 */
export const test = base.extend<{
  createUsers: (count: number) => Promise<UserSession[]>;
  lobbyWithUsers: (count: number) => Promise<LobbyWithUsers>;
}>({
  /**
   * Creates N browser contexts with authenticated users
   */
  createUsers: async ({ browser }, use) => {
    const createdSessions: UserSession[] = [];

    const createUsers = async (count: number): Promise<UserSession[]> => {
      // Register all users in parallel via API
      const registrations = await Promise.all(
        Array.from({ length: count }, (_, i) =>
          registerUser(generateUsername(i), 'testpassword123')
        )
      );

      // Create browser contexts and set up auth in parallel
      const sessions = await Promise.all(
        registrations.map(async ({ user, token }) => {
          const context = await browser.newContext();
          const page = await context.newPage();

          // Navigate to the app and set the token in localStorage
          await page.goto('/');
          await page.evaluate((accessToken) => {
            localStorage.setItem('accessToken', accessToken);
          }, token);

          // Reload to apply authentication
          await page.reload();

          const session: UserSession = {
            context,
            page,
            user,
            token,
          };

          createdSessions.push(session);
          return session;
        })
      );

      return sessions;
    };

    await use(createUsers);

    // Cleanup: close all browser contexts in parallel
    await Promise.all(createdSessions.map((session) => session.context.close()));
  },

  /**
   * Creates a lobby with N users where:
   * - First user creates the lobby
   * - All other users join the lobby in parallel
   */
  lobbyWithUsers: async ({ createUsers }, use) => {
    const lobbyWithUsers = async (count: number): Promise<LobbyWithUsers> => {
      if (count < 2) {
        throw new Error('lobbyWithUsers requires at least 2 users');
      }

      // Create all users
      const users = await createUsers(count);

      // First user creates the lobby
      const creatorClient = new ApiClient(users[0].token);
      const lobby = await creatorClient.post<Lobby>('/lobbies', {
        draftMode: 'pro_play',
        timerDurationSeconds: 90, // Increased for test reliability
      });

      // All other users join the lobby in parallel
      await Promise.all(
        users.slice(1).map((user) => {
          const joinerClient = new ApiClient(user.token);
          return joinerClient.post(`/lobbies/${lobby.id}/join`);
        })
      );

      // Fetch the updated lobby state
      const updatedLobby = await creatorClient.get<Lobby>(`/lobbies/${lobby.id}`);

      return {
        lobby: updatedLobby,
        users,
      };
    };

    await use(lobbyWithUsers);
  },
});

export { expect } from '@playwright/test';

/**
 * Helper to get an API client for a user session
 */
export function getApiClient(session: UserSession): ApiClient {
  return new ApiClient(session.token);
}

/**
 * Helper to set user ready status
 */
export async function setUserReady(session: UserSession, lobbyId: string, ready: boolean): Promise<void> {
  const client = new ApiClient(session.token);
  await client.post(`/lobbies/${lobbyId}/ready`, { ready });
}

/**
 * Helper to generate teams
 */
export async function generateTeams(session: UserSession, lobbyId: string): Promise<MatchOption[]> {
  const client = new ApiClient(session.token);
  return client.post<MatchOption[]>(`/lobbies/${lobbyId}/generate-teams`);
}

/**
 * Helper to select a match option
 */
export async function selectMatchOption(session: UserSession, lobbyId: string, optionNumber: number): Promise<Lobby> {
  const client = new ApiClient(session.token);
  return client.post<Lobby>(`/lobbies/${lobbyId}/select-option`, { optionNumber });
}

/**
 * Helper to get current lobby state
 */
export async function getLobby(session: UserSession, lobbyId: string): Promise<Lobby> {
  const client = new ApiClient(session.token);
  return client.get<Lobby>(`/lobbies/${lobbyId}`);
}

// ========== Captain Management Helpers ==========

/**
 * Take captain status from current captain
 */
export async function takeCaptain(session: UserSession, lobbyId: string): Promise<Lobby> {
  const client = new ApiClient(session.token);
  return client.post<Lobby>(`/lobbies/${lobbyId}/take-captain`);
}

/**
 * Promote a teammate to captain
 */
export async function promoteCaptain(
  session: UserSession,
  lobbyId: string,
  targetUserId: string
): Promise<Lobby> {
  const client = new ApiClient(session.token);
  return client.post<Lobby>(`/lobbies/${lobbyId}/promote-captain`, { userId: targetUserId });
}

/**
 * Kick a player from the lobby
 */
export async function kickPlayer(
  session: UserSession,
  lobbyId: string,
  targetUserId: string
): Promise<Lobby> {
  const client = new ApiClient(session.token);
  return client.post<Lobby>(`/lobbies/${lobbyId}/kick`, { userId: targetUserId });
}

// ========== Pending Action Helpers ==========

export interface PendingAction {
  id: string;
  actionType: 'swap_players' | 'swap_roles' | 'matchmake' | 'start_draft';
  status: 'pending' | 'approved' | 'cancelled' | 'expired';
  proposedByUser: string;
  proposedBySide: 'blue' | 'red';
  player1Id?: string;
  player2Id?: string;
  approvedByBlue: boolean;
  approvedByRed: boolean;
  expiresAt: string;
}

/**
 * Propose a player or role swap
 */
export async function proposeSwap(
  session: UserSession,
  lobbyId: string,
  player1Id: string,
  player2Id: string,
  swapType: 'players' | 'roles'
): Promise<PendingAction> {
  const client = new ApiClient(session.token);
  return client.post<PendingAction>(`/lobbies/${lobbyId}/swap`, {
    player1Id,
    player2Id,
    swapType,
  });
}

/**
 * Get current pending action (if any)
 */
export async function getPendingAction(
  session: UserSession,
  lobbyId: string
): Promise<PendingAction | null> {
  const client = new ApiClient(session.token);
  try {
    return await client.get<PendingAction>(`/lobbies/${lobbyId}/pending-action`);
  } catch {
    return null;
  }
}

/**
 * Approve a pending action
 */
export async function approvePendingAction(
  session: UserSession,
  lobbyId: string,
  actionId: string
): Promise<Lobby> {
  const client = new ApiClient(session.token);
  return client.post<Lobby>(`/lobbies/${lobbyId}/pending-action/${actionId}/approve`);
}

/**
 * Cancel a pending action
 */
export async function cancelPendingAction(
  session: UserSession,
  lobbyId: string,
  actionId: string
): Promise<Lobby> {
  const client = new ApiClient(session.token);
  return client.post<Lobby>(`/lobbies/${lobbyId}/pending-action/${actionId}/cancel`);
}

// ========== Advanced Fixtures ==========

/**
 * Helper to create a lobby with 10 users in waiting_for_players status.
 * Teams are auto-assigned as players join (first 5 to blue, next 5 to red).
 * This is suitable for testing captain management actions (take, promote, kick, swap).
 */
export async function setupLobbyWithPlayers(
  createUsers: (count: number) => Promise<UserSession[]>
): Promise<LobbyWithUsers> {
  const users = await createUsers(10);

  // Creator creates lobby (auto-joins as blue captain)
  const creatorClient = new ApiClient(users[0].token);
  const lobby = await creatorClient.post<Lobby>('/lobbies', {
    draftMode: 'pro_play',
    timerDurationSeconds: 90,
  });

  // Other users join sequentially to ensure proper team assignment order
  // (Blue: users 0-4, Red: users 5-9)
  for (const user of users.slice(1)) {
    const client = new ApiClient(user.token);
    await client.post(`/lobbies/${lobby.id}/join`);
  }

  // Fetch the updated lobby state
  const updatedLobby = await creatorClient.get<Lobby>(`/lobbies/${lobby.id}`);

  return { lobby: updatedLobby, users };
}

/**
 * Helper to create a lobby with 10 users and teams already selected.
 * This creates users, creates lobby, all join, all ready, generates teams, selects option 1.
 * Use this for testing swap workflows that require pending action approval.
 */
export async function setupLobbyWithTeams(
  createUsers: (count: number) => Promise<UserSession[]>
): Promise<LobbyWithUsers> {
  const users = await createUsers(10);

  // Creator creates lobby
  const creatorClient = new ApiClient(users[0].token);
  const lobby = await creatorClient.post<Lobby>('/lobbies', {
    draftMode: 'pro_play',
    timerDurationSeconds: 90,
  });

  // All other users join in parallel
  await Promise.all(
    users.slice(1).map((user) => {
      const client = new ApiClient(user.token);
      return client.post(`/lobbies/${lobby.id}/join`);
    })
  );

  // All users ready up in parallel
  await Promise.all(
    users.map((user) => setUserReady(user, lobby.id, true))
  );

  // Generate teams
  await generateTeams(users[0], lobby.id);

  // Select first option
  const updatedLobby = await selectMatchOption(users[0], lobby.id, 1);

  return { lobby: updatedLobby, users };
}
