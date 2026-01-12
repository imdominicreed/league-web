import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 1,  // Add local retry for flaky tests
  workers: process.env.CI ? 1 : 2,  // Reduce workers to avoid resource contention
  reporter: 'html',
  timeout: 90000,  // Increase from 60s to 90s for multi-user tests

  use: {
    baseURL: 'http://localhost:3000',
    // Capture trace on first retry for debugging
    trace: 'on-first-retry',
    // Capture screenshot on failure
    screenshot: 'only-on-failure',
    // Retain video on failure for debugging
    video: 'retain-on-failure',
  },

  projects: [
    // Smoke tests: Quick sanity checks
    {
      name: 'smoke',
      testMatch: /@smoke/,
      use: { ...devices['Desktop Chrome'] },
    },
    // Full test suite
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
  },
});
