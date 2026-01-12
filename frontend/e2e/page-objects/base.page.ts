import { Page, Locator, expect } from '@playwright/test';
import { TIMEOUTS } from '../helpers/wait-strategies';

/**
 * Base page object class with common utilities.
 * All page objects should extend this class.
 */
export abstract class BasePage {
  constructor(protected readonly page: Page) {}

  /**
   * Get the underlying Playwright Page object.
   * Use for low-level operations not covered by page object methods.
   */
  getPage(): Page {
    return this.page;
  }

  /**
   * Get element by data-testid attribute.
   * Preferred over text/class selectors for stability.
   */
  protected byTestId(testId: string): Locator {
    return this.page.locator(`[data-testid="${testId}"]`);
  }

  /**
   * Get element by data-testid with fallback to text selector.
   * Use during migration from text selectors to data-testid.
   */
  protected byTestIdOrText(testId: string, fallbackText: string): Locator {
    return this.byTestId(testId).or(this.page.locator(`text=${fallbackText}`)).first();
  }

  /**
   * Get element by data-testid with fallback to CSS selector.
   * Use during migration from class selectors to data-testid.
   */
  protected byTestIdOrSelector(testId: string, fallbackSelector: string): Locator {
    return this.byTestId(testId).or(this.page.locator(fallbackSelector)).first();
  }

  /**
   * Wait for page to be fully loaded (network idle).
   */
  async waitForPageLoad(timeout = TIMEOUTS.MEDIUM): Promise<void> {
    await this.page.waitForLoadState('networkidle', { timeout });
  }

  /**
   * Wait for element to be visible.
   */
  async waitForVisible(locator: Locator, timeout = TIMEOUTS.MEDIUM): Promise<void> {
    await expect(locator).toBeVisible({ timeout });
  }

  /**
   * Wait for element to be hidden.
   */
  async waitForHidden(locator: Locator, timeout = TIMEOUTS.MEDIUM): Promise<void> {
    await expect(locator).not.toBeVisible({ timeout });
  }

  /**
   * Wait for URL to match pattern.
   */
  async waitForUrl(pattern: string | RegExp, timeout = TIMEOUTS.MEDIUM): Promise<void> {
    await this.page.waitForURL(pattern, { timeout });
  }
}
