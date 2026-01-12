import { test as base, BrowserContext, Page } from '@playwright/test';
import { registerUserViaApi, RegisterResponse } from './api-client';
import { generateTestUsername, TEST_PASSWORD, FRONTEND_BASE } from '../helpers/test-data';

/**
 * Auth fixture types
 */
export interface AuthFixtures {
  /** Pre-authenticated browser context (no reload needed) */
  authenticatedContext: BrowserContext;
  /** Pre-authenticated page ready to use */
  authenticatedPage: Page;
  /** Current user's credentials */
  testUser: RegisterResponse;
}

/**
 * Extended test with authentication fixtures.
 *
 * Usage:
 *   import { test } from '../fixtures/auth.fixtures';
 *
 *   test('authenticated test', async ({ authenticatedPage, testUser }) => {
 *     await authenticatedPage.goto('/');
 *     // User is already logged in - no reload needed
 *   });
 */
export const test = base.extend<AuthFixtures>({
  // Create a test user for this test
  testUser: async ({}, use) => {
    const displayName = generateTestUsername('e2e');
    const user = await registerUserViaApi(displayName, TEST_PASSWORD);
    await use(user);
  },

  // Create authenticated context using storageState (no reload needed)
  authenticatedContext: async ({ browser, testUser }, use) => {
    const context = await browser.newContext({
      storageState: {
        cookies: [],
        origins: [
          {
            origin: FRONTEND_BASE,
            localStorage: [
              {
                name: 'accessToken',
                value: testUser.token,
              },
            ],
          },
        ],
      },
    });

    await use(context);
    await context.close();
  },

  // Create authenticated page from the context
  authenticatedPage: async ({ authenticatedContext }, use) => {
    const page = await authenticatedContext.newPage();
    await use(page);
  },
});

export { expect } from '@playwright/test';
