/**
 * Fixtures Index
 *
 * Re-exports all fixtures for easy importing.
 *
 * Usage:
 *   import { test, registerUserViaApi, createLobbyViaApi } from '../fixtures';
 */

// Auth fixtures
export { test, expect } from './auth.fixtures';
export type { AuthFixtures } from './auth.fixtures';

// API client functions
export {
  registerUserViaApi,
  createLobbyViaApi,
  joinLobbyViaApi,
  setReadyViaApi,
  generateTeamsViaApi,
  selectOptionViaApi,
  startDraftViaApi,
  setAuthToken,
  setAuthTokenAndReload,
  createTestUser,
} from './api-client';
export type { RegisterResponse, LobbyResponse, MatchOption as ApiMatchOption } from './api-client';

// Types
export type {
  UserSession,
  LobbyPlayer,
  Lobby,
  MatchOption,
  Side,
  Role,
} from './types';

// Helpers
export {
  generateTestUsername,
  generateTestUsernames,
  TEST_PASSWORD,
  API_BASE,
  FRONTEND_BASE,
} from '../helpers/test-data';

export {
  TIMEOUTS,
  POLL_INTERVALS,
  waitForCondition,
  waitForTextChange,
  waitForCount,
  waitForApiResponse,
  waitForWebSocketConnected,
  waitForPageReady,
} from '../helpers/wait-strategies';
