import { expect } from '@playwright/test';
import { BasePage } from './base.page';
import { TIMEOUTS } from '../helpers/wait-strategies';

/**
 * Lobby Room page interactions
 */
export class LobbyRoomPage extends BasePage {
  async goto(lobbyId: string) {
    await this.page.goto(`/lobby/${lobbyId}`);
  }

  async expectLobbyCode(code: string) {
    const codeDisplay = this.byTestIdOrText('lobby-code-display', code);
    await expect(codeDisplay).toBeVisible();
  }

  async expectPlayerCount(current: number, total: number = 10) {
    const countDisplay = this.byTestIdOrText('lobby-player-count', `${current}/${total}`);
    await expect(countDisplay).toContainText(`${current}/${total}`);
  }

  async clickReadyUp() {
    await this.byTestIdOrText('captain-button-ready', 'Ready Up').click();
  }

  async clickCancelReady() {
    await this.byTestIdOrText('captain-button-ready', 'Cancel Ready').click();
  }

  async expectReadyButton() {
    await expect(this.page.locator('button:has-text("Ready Up")')).toBeVisible();
  }

  async expectCancelReadyButton() {
    await expect(this.page.locator('button:has-text("Cancel Ready")')).toBeVisible();
  }

  async clickGenerateTeams() {
    // Use the captain's Propose Matchmake button
    await this.page.click('[data-testid="captain-button-matchmake"]');
  }

  async expectGenerateTeamsButton() {
    await expect(
      this.page.locator('[data-testid="captain-button-matchmake"]')
    ).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  }

  async expectGeneratingTeams() {
    await expect(this.page.locator('button:has-text("Generating Teams...")')).toBeVisible();
  }

  async selectOption(optionNumber: number) {
    // Click the "Select This Option" button within the option card
    const optionCard = this.byTestId(`match-option-${optionNumber}`);
    await optionCard.locator('button:has-text("Select This Option")').click();
  }

  async clickConfirmSelection() {
    // This method is deprecated - selection happens via selectOption
    // Keeping for backwards compatibility but it's a no-op
  }

  async clickStartDraft() {
    const startButton = this.byTestIdOrText('lobby-button-start-draft', 'Start Draft');
    if (await startButton.count() > 0 && await startButton.isVisible()) {
      await startButton.click();
    } else {
      // Fall back to propose start draft
      const proposeButton = this.byTestIdOrText('captain-button-start-draft', 'Propose Start Draft');
      if (await proposeButton.count() > 0 && await proposeButton.isVisible()) {
        await proposeButton.click();
      } else {
        await this.page.click('button:has-text("Start Draft")');
      }
    }
  }

  async expectStartDraftButton() {
    await expect(
      this.byTestIdOrText('lobby-button-start-draft', 'Start Draft').or(
        this.byTestIdOrText('captain-button-start-draft', 'Propose Start Draft')
      ).first()
    ).toBeVisible({ timeout: TIMEOUTS.MEDIUM });
  }

  async expectOnDraftPage() {
    await this.page.waitForURL(/\/draft\//);
  }

  async waitForMatchOptions() {
    await expect(this.page.locator('text=Option 1')).toBeVisible({ timeout: TIMEOUTS.LONG });
  }

  async leave() {
    await this.byTestIdOrText('lobby-link-leave', 'Leave').click();
    await this.page.waitForURL('/');
  }

  // Captain-related methods
  async isCaptain(): Promise<boolean> {
    const captainBadge = this.byTestIdOrText('captain-badge', 'Captain');
    return await captainBadge.count() > 0;
  }

  async clickTakeCaptain() {
    await this.byTestIdOrText('captain-button-take', 'Take Captain').click();
  }

  async expectTakeCaptainButton() {
    await expect(this.byTestIdOrText('captain-button-take', 'Take Captain')).toBeVisible();
  }

  async clickProposeMatchmake() {
    await this.byTestIdOrText('captain-button-matchmake', 'Propose Matchmake').click();
  }

  async clickProposeStartDraft() {
    await this.byTestIdOrText('captain-button-start-draft', 'Propose Start Draft').click();
  }

  async expectPendingActionBanner() {
    await expect(this.page.locator('.bg-yellow-900\\/30')).toBeVisible();
  }

  async clickApprovePendingAction() {
    await this.page.click('button:has-text("Approve")');
  }

  async clickCancelPendingAction() {
    await this.page.click('button:has-text("Cancel")');
  }

  async expectTeamColumn(side: 'blue' | 'red') {
    const teamText = side === 'blue' ? 'Blue Team' : 'Red Team';
    await expect(this.page.locator(`text=${teamText}`)).toBeVisible();
  }

  // ========== Captain Modal Methods ==========

  async clickPromoteCaptain() {
    await this.byTestIdOrText('captain-button-promote', 'Promote Captain').click();
  }

  async clickKickPlayer() {
    await this.byTestIdOrText('captain-button-kick', 'Kick Player').click();
  }

  async clickProposeSwap() {
    await this.byTestIdOrText('captain-button-swap', 'Swap').click();
  }

  async selectPlayerInModal(displayName: string) {
    await this.page.click(`button:has-text("${displayName}")`);
  }

  async cancelModal() {
    const cancelButtons = this.page.locator('button:has-text("Cancel")');
    const count = await cancelButtons.count();
    if (count > 0) {
      await cancelButtons.last().click();
    }
  }

  // ========== Swap Modal Configuration ==========

  async selectSwapType(type: 'players' | 'roles') {
    if (type === 'players') {
      await this.page.click('button:has-text("Between Teams")');
    } else {
      await this.page.click('button:has-text("Swap Roles")');
    }
  }

  async selectPlayer1InSwap(displayName: string) {
    const select1 = this.page.locator('select').first();
    const options = await select1.locator('option').allTextContents();
    const matchingOption = options.find((opt) => opt.includes(displayName));
    if (matchingOption) {
      await select1.selectOption({ label: matchingOption });
    }
  }

  async selectPlayer2InSwap(displayName: string) {
    const select2 = this.page.locator('select').nth(1);
    const options = await select2.locator('option').allTextContents();
    const matchingOption = options.find((opt) => opt.includes(displayName));
    if (matchingOption) {
      await select2.selectOption({ label: matchingOption });
    }
  }

  async confirmSwapProposal() {
    const modal = this.page.locator('.fixed.inset-0.bg-black\\/70');
    await modal.locator('button.bg-lol-gold:has-text("Propose")').click();
  }

  // ========== Pending Action Verification ==========

  async expectNoPendingActionBanner() {
    await expect(this.page.locator('.bg-yellow-900\\/30')).not.toBeVisible();
  }

  async getPendingActionType(): Promise<string> {
    const label = this.page.locator('.bg-yellow-900\\/30 .text-yellow-400.font-semibold');
    return (await label.textContent()) || '';
  }

  async expectApproveButton() {
    await expect(this.page.locator('button:has-text("Approve")')).toBeVisible();
  }

  async expectApprovedBadge() {
    await expect(this.page.locator('.text-green-400:has-text("Approved")')).toBeVisible();
  }

  // ========== Player Verification ==========

  async expectPlayerOnTeam(displayName: string, team: 'blue' | 'red') {
    const teamSection = team === 'blue'
      ? this.page.locator('.bg-blue-900\\/30, [class*="blue"]').first()
      : this.page.locator('.bg-red-900\\/30, [class*="red"]').first();
    await expect(teamSection.locator(`text=${displayName}`)).toBeVisible();
  }

  async expectPlayerNotInLobby(displayName: string) {
    const blueTeamColumn = this.page.locator('.bg-blue-900\\/30').filter({ has: this.page.locator('h3') });
    const redTeamColumn = this.page.locator('.bg-red-900\\/30').filter({ has: this.page.locator('h3') });

    await expect(blueTeamColumn.locator(`text=${displayName}`)).not.toBeVisible();
    await expect(redTeamColumn.locator(`text=${displayName}`)).not.toBeVisible();
  }

  async expectCaptainControls() {
    await expect(this.page.locator('text=Captain Controls')).toBeVisible();
  }

  async expectPlayerActions() {
    await expect(this.page.locator('text=Player Actions')).toBeVisible();
  }
}
