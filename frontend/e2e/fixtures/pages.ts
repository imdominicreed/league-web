/**
 * @deprecated This file is kept for backward compatibility.
 * Please import from the new locations:
 *   - Page objects: import { HomePage, LoginPage, ... } from '../page-objects';
 *   - API helpers: import { registerUserViaApi, ... } from '../fixtures';
 *   - Test data: import { generateTestUsername, ... } from '../helpers/test-data';
 */

import { Page } from '@playwright/test';
import { setAuthToken as setAuthTokenBase } from './api-client';

// Re-export page objects
export {
  HomePage,
  LoginPage,
  RegisterPage,
  CreateLobbyPage,
  LobbyRoomPage,
  DraftRoomPage,
  waitForAnyTurn,
} from '../page-objects';

// Re-export API helpers
export {
  registerUserViaApi,
  createLobbyViaApi,
  joinLobbyViaApi,
  setReadyViaApi,
} from './api-client';

// Re-export test data helpers
export { generateTestUsername } from '../helpers/test-data';

/**
 * Set auth token and reload page.
 * Note: For better performance, consider using the auth.fixtures.ts
 * which uses storageState to avoid reloads.
 */
export async function setAuthToken(page: Page, token: string): Promise<void> {
  await setAuthTokenBase(page, token);
  await page.reload();
}
