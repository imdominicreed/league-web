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
 * API helper class for making authenticated requests
 */
class ApiClient {
  constructor(private token?: string) {}

  async request<T>(endpoint: string, options: { method?: string; body?: unknown } = {}): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
      method: options.method || 'GET',
      headers,
      body: options.body ? JSON.stringify(options.body) : undefined,
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`API Error ${response.status}: ${error}`);
    }

    return response.json();
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
  const random = Math.random().toString(36).substring(2, 6);
  return `e2e_user_${index}_${timestamp}_${random}`;
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
        timerDurationSeconds: 30,
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
