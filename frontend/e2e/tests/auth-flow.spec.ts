import { test, expect } from '@playwright/test';
import { HomePage, LoginPage, RegisterPage } from '../page-objects';
import { generateTestUsername, registerUserViaApi } from '../fixtures';

test.describe('Authentication Flow', () => {
  test('user can register through the UI', async ({ page }) => {
    const homePage = new HomePage(page);
    const registerPage = new RegisterPage(page);

    const username = generateTestUsername('reg');
    const password = 'testpassword123';

    // Start at home page
    await homePage.goto();
    await homePage.expectUnauthenticated();

    // Navigate to register
    await homePage.clickRegister();

    // Fill and submit registration form
    await registerPage.register(username, password);

    // Should be redirected to home and authenticated
    await page.waitForURL('/');
    await homePage.expectAuthenticated(username);
    await homePage.expectAuthenticatedMenu();
  });

  test('user can login through the UI', async ({ page }) => {
    const homePage = new HomePage(page);
    const loginPage = new LoginPage(page);

    // First register a user via API
    const username = generateTestUsername('login');
    const password = 'testpassword123';
    await registerUserViaApi(username, password);

    // Start at home page
    await homePage.goto();
    await homePage.expectUnauthenticated();

    // Navigate to login
    await homePage.clickLogin();

    // Fill and submit login form
    await loginPage.login(username, password);

    // Should be redirected to home and authenticated
    await page.waitForURL('/');
    await homePage.expectAuthenticated(username);
    await homePage.expectAuthenticatedMenu();
  });

  test('shows error on invalid login credentials', async ({ page }) => {
    const loginPage = new LoginPage(page);

    await loginPage.goto();

    // Try to login with invalid credentials
    await loginPage.login('nonexistent_user_12345', 'wrongpassword');

    // Should show error message
    await loginPage.expectError();

    // Should still be on login page
    expect(page.url()).toContain('/login');
  });

  test('shows error on duplicate registration', async ({ page }) => {
    const registerPage = new RegisterPage(page);

    // First register a user via API
    const username = generateTestUsername('dup');
    const password = 'testpassword123';
    await registerUserViaApi(username, password);

    // Try to register with same username
    await registerPage.goto();
    await registerPage.register(username, password);

    // Should show error message
    await registerPage.expectError();

    // Should still be on register page
    expect(page.url()).toContain('/register');
  });

  test('login page has link to register', async ({ page }) => {
    await page.goto('/login');

    // Click the register link
    await page.click('a:has-text("Register")');

    // Should navigate to register page
    await page.waitForURL('/register');
  });

  test('register page has link to login', async ({ page }) => {
    await page.goto('/register');

    // Click the login link
    await page.click('a:has-text("Login")');

    // Should navigate to login page
    await page.waitForURL('/login');
  });
});
