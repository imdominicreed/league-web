import { expect } from '@playwright/test';
import { BasePage } from './base.page';

/**
 * Register page interactions
 */
export class RegisterPage extends BasePage {
  async goto() {
    await this.page.goto('/register');
  }

  async fillUsername(username: string) {
    await this.byTestIdOrSelector('register-input-username', '#displayName').fill(username);
  }

  async fillPassword(password: string) {
    await this.byTestIdOrSelector('register-input-password', '#password').fill(password);
  }

  async submit() {
    // Click the submit button directly by test ID
    await this.page.click('[data-testid="register-button-submit"]');
    // Wait for API response (loading state or error)
    await this.page.waitForTimeout(2000);
  }

  async register(username: string, password: string) {
    await this.fillUsername(username);
    await this.fillPassword(password);
    await this.submit();
  }

  async expectError(errorText?: string) {
    const errorBox = this.byTestIdOrSelector('register-error-message', '.bg-red-500\\/20');
    await expect(errorBox).toBeVisible({ timeout: 10000 });
    if (errorText) {
      await expect(errorBox).toContainText(errorText);
    }
  }
}
