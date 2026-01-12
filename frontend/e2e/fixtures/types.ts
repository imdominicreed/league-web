/**
 * Shared type definitions for E2E tests.
 */

import { Page, BrowserContext } from '@playwright/test';

/**
 * Test user session with page and credentials.
 */
export interface UserSession {
  page: Page;
  context: BrowserContext;
  userId: string;
  token: string;
  displayName: string;
}

/**
 * Lobby player data.
 */
export interface LobbyPlayer {
  id: string;
  lobbyId: string;
  userId: string;
  displayName: string;
  team: 'blue' | 'red' | 'spectator' | null;
  assignedRole: string | null;
  isReady: boolean;
  isCaptain: boolean;
}

/**
 * Lobby data.
 */
export interface Lobby {
  id: string;
  shortCode: string;
  creatorId: string;
  status: string;
  draftMode: string;
  timerDurationSeconds: number;
  players: LobbyPlayer[];
  selectedMatchOption?: number;
  roomId?: string;
}

/**
 * Match option from matchmaking.
 */
export interface MatchOption {
  optionNumber: number;
  algorithmType: string;
  balanceScore: number;
  blueTeamAvgMmr: number;
  redTeamAvgMmr: number;
  blueTeamAvgComfort: number;
  redTeamAvgComfort: number;
  maxLaneGap: number;
}

/**
 * Team side.
 */
export type Side = 'blue' | 'red' | 'spectator';

/**
 * Role in a team.
 */
export type Role = 'top' | 'jungle' | 'mid' | 'adc' | 'support';
