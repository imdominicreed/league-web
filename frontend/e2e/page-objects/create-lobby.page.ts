import { BasePage } from './base.page';

export type VotingMode = 'majority' | 'unanimous' | 'captain_override';

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

  async enableVoting(enable: boolean = true) {
    const checkbox = this.page.locator('#votingEnabled');
    const isChecked = await checkbox.isChecked();
    if (enable && !isChecked) {
      await checkbox.check();
    } else if (!enable && isChecked) {
      await checkbox.uncheck();
    }
  }

  async selectVotingMode(mode: VotingMode) {
    // Voting mode dropdown only visible when voting is enabled
    const votingModeSelect = this.page.locator('select').nth(1);
    await votingModeSelect.selectOption(mode);
  }

  async submit() {
    await this.page.click('button:has-text("Create Lobby")');
  }

  async createLobby(mode: 'pro_play' | 'fearless' = 'pro_play', timerSeconds: number = 30) {
    await this.selectDraftMode(mode);
    await this.setTimerDuration(timerSeconds);
    await this.submit();
  }

  async createLobbyWithVoting(
    mode: 'pro_play' | 'fearless' = 'pro_play',
    timerSeconds: number = 30,
    votingMode: VotingMode = 'majority'
  ) {
    await this.selectDraftMode(mode);
    await this.setTimerDuration(timerSeconds);
    await this.enableVoting(true);
    await this.selectVotingMode(votingMode);
    await this.submit();
  }
}
