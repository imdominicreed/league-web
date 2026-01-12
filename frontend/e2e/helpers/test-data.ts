/**
 * Test data generators and factories for E2E tests.
 */

/**
 * Generate a unique test username with timestamp and random suffix.
 */
export function generateTestUsername(prefix = 'testuser'): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(2, 8);
  return `${prefix}_${timestamp}_${random}`;
}

/**
 * Generate multiple unique usernames.
 */
export function generateTestUsernames(
  count: number,
  prefix = 'testuser'
): string[] {
  return Array.from({ length: count }, (_, i) =>
    generateTestUsername(`${prefix}${i + 1}`)
  );
}

/**
 * Default test password used across all test users.
 */
export const TEST_PASSWORD = 'password123';

/**
 * API base URL for direct API calls in tests.
 */
export const API_BASE = 'http://localhost:9999/api/v1';

/**
 * Frontend base URL.
 */
export const FRONTEND_BASE = 'http://localhost:3000';
