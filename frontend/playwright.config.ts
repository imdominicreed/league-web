import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E Test Configuration
 *
 * Test Categories:
 * - smoke: Quick sanity checks (< 30s per test)
 * - fast: Single-user tests that run in parallel
 * - serial: Multi-user tests that need exclusive browser resources
 *
 * Parallelization Strategy:
 * - Single-user tests run fully parallel
 * - Multi-user tests (10-man lobbies, draft flows) run serially via test.describe.configure({ mode: 'serial' })
 * - WebSocket tests are grouped to avoid connection contention
 *
 * Visual Regression:
 * - Use toHaveScreenshot() for visual comparisons
 * - Snapshots stored in e2e/__snapshots__
 * - Run with --update-snapshots to update baselines
 */
export default defineConfig({
  testDir: './e2e',

  // Parallelization: Enable for single-user tests, serial tests opt-out via describe.configure
  fullyParallel: true,

  // Forbid test.only in CI to prevent accidental skipping
  forbidOnly: !!process.env.CI,

  // Retry configuration for flaky test handling
  retries: process.env.CI ? 2 : 1,

  // Worker configuration:
  // - CI: 1 worker to avoid resource contention with containers
  // - Local: 2 workers for faster execution while maintaining stability
  workers: process.env.CI ? 1 : 2,

  // Reporter configuration
  reporter: process.env.CI
    ? [['html', { open: 'never' }], ['github']]
    : [['html', { open: 'on-failure' }], ['list']],

  // Global test timeout (increased for multi-user tests)
  timeout: 90000,

  // Expect timeout for assertions
  expect: {
    timeout: 10000,
    // Visual regression snapshot settings
    toHaveScreenshot: {
      // Allow slight pixel differences (anti-aliasing, etc.)
      maxDiffPixels: 100,
      // Threshold for pixel comparison (0-1, where 0 is exact match)
      threshold: 0.2,
    },
    toMatchSnapshot: {
      // Threshold for snapshot comparison
      threshold: 0.2,
    },
  },

  // Global test settings
  use: {
    baseURL: 'http://localhost:3000',

    // Capture trace on first retry for debugging
    trace: 'on-first-retry',

    // Capture screenshot on failure for debugging
    screenshot: 'only-on-failure',

    // Retain video on failure for debugging
    video: 'retain-on-failure',

    // Viewport settings
    viewport: { width: 1280, height: 720 },

    // Ignore HTTPS errors (useful for local development)
    ignoreHTTPSErrors: true,

    // Action timeout
    actionTimeout: 15000,

    // Navigation timeout
    navigationTimeout: 30000,
  },

  projects: [
    // ====== SMOKE TESTS ======
    // Quick sanity checks that should pass before running full suite
    {
      name: 'smoke',
      testMatch: /@smoke/,
      use: {
        ...devices['Desktop Chrome'],
      },
      // Smoke tests should be fast
      timeout: 30000,
    },

    // ====== FAST TESTS ======
    // Single-user tests (auth, navigation, basic flows)
    // These run in parallel for speed
    {
      name: 'fast',
      testMatch: /\/(auth-flow|navigation|error-handling)\.spec\.ts$/,
      use: {
        ...devices['Desktop Chrome'],
      },
      // Fast tests can use more parallelism
      fullyParallel: true,
    },

    // ====== SERIAL TESTS ======
    // Multi-user tests (lobbies, drafts with multiple browsers)
    // These run serially to avoid WebSocket/resource contention
    {
      name: 'serial',
      testMatch: /\/(multi-user|lobby-captain|draft-edge|draft-flow).*\.spec\.ts$/,
      use: {
        ...devices['Desktop Chrome'],
      },
      // Serial tests need longer timeouts
      timeout: 180000,
      // Force serial execution for multi-user tests
      fullyParallel: false,
    },

    // ====== FULL SUITE ======
    // All tests - default project
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
      },
    },
  ],

  // Snapshot output directory for visual regression
  snapshotDir: './e2e/__snapshots__',

  // Output directory for test artifacts
  outputDir: './e2e/test-results',

  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
    // Ensure server is ready before tests
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
