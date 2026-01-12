import { expect } from '@playwright/test';
import { BasePage } from './base.page';

/**
 * Home page interactions
 */
export class HomePage extends BasePage {
  async goto() {
    await this.page.goto('/');
  }

  async clickLogin() {
    await this.byTestIdOrText('home-link-login', 'Login').click();
    await this.page.waitForURL('/login');
  }

  async clickRegister() {
    await this.byTestIdOrText('home-link-register', 'Register').click();
    await this.page.waitForURL('/register');
  }

  async clickCreateDraftRoom() {
    await this.byTestIdOrText('home-link-create-draft', 'Create Draft Room').click();
    await this.page.waitForURL('/create');
  }

  async clickJoinRoom() {
    await this.byTestIdOrText('home-link-join-room', 'Join Room').click();
    await this.page.waitForURL('/join');
  }

  async clickMyProfile() {
    await this.byTestIdOrText('home-link-profile', 'My Profile').click();
    await this.page.waitForURL('/profile');
  }

  async clickCreateLobby() {
    await this.byTestIdOrText('home-link-create-lobby', 'Create 10-Man Lobby').click();
    await this.page.waitForURL('/create-lobby');
  }

  async expectAuthenticated(displayName: string) {
    const welcomeMessage = this.byTestIdOrText('home-welcome-message', `Welcome, ${displayName}`);
    await expect(welcomeMessage).toBeVisible();
  }

  async expectUnauthenticated() {
    await expect(this.byTestIdOrText('home-link-login', 'Login')).toBeVisible();
    await expect(this.byTestIdOrText('home-link-register', 'Register')).toBeVisible();
  }

  async expectAuthenticatedMenu() {
    await expect(this.byTestIdOrText('home-link-create-draft', 'Create Draft Room')).toBeVisible();
    await expect(this.byTestIdOrText('home-link-join-room', 'Join Room')).toBeVisible();
    await expect(this.byTestIdOrText('home-link-profile', 'My Profile')).toBeVisible();
    await expect(this.byTestIdOrText('home-link-create-lobby', 'Create 10-Man Lobby')).toBeVisible();
  }
}
