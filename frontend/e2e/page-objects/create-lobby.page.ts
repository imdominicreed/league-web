import { BasePage } from './base.page';

/**
 * Create Lobby page interactions
 */
export class CreateLobbyPage extends BasePage {
  async goto() {
    await this.page.goto('/create-lobby');
  }

  async selectDraftMode(mode: 'pro_play' | 'fearless') {
    await this.page.selectOption('select', mode);
  }

  async setTimerDuration(seconds: number) {
    await this.page.fill('input[type="number"]', String(seconds));
  }

  async submit() {
    await this.page.click('button:has-text("Create Lobby")');
  }

  async createLobby(mode: 'pro_play' | 'fearless' = 'pro_play', timerSeconds: number = 30) {
    await this.selectDraftMode(mode);
    await this.setTimerDuration(timerSeconds);
    await this.submit();
  }
}
