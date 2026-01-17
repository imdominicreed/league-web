import { LobbyPlayer, MatchOption, TeamStats, VotingStatus, PendingAction, Role } from './index'

// Lobby WebSocket Message Types
export type LobbyMessageType =
  | 'lobby_state_sync'
  | 'player_joined'
  | 'player_left'
  | 'player_ready_changed'
  | 'status_changed'
  | 'match_options_generated'
  | 'team_selected'
  | 'vote_cast'
  | 'action_proposed'
  | 'action_approved'
  | 'action_executed'
  | 'action_cancelled'
  | 'draft_starting'
  | 'captain_changed'
  | 'player_kicked'
  | 'team_stats_updated'
  | 'voting_status_updated'
  | 'error'
  | 'join_lobby'

// Base message envelope
export interface LobbyWSMessage<T = unknown> {
  type: LobbyMessageType
  payload: T
  timestamp: number
}

// ============== Payload Types ==============

export interface LobbyInfo {
  id: string
  shortCode: string
  createdBy: string
  status: string
  selectedMatchOption: number | null
  draftMode: string
  timerDurationSeconds: number
  roomId: string | null
  votingEnabled: boolean
  votingMode: string
  votingDeadline?: string
}

export interface LobbyPlayerInfo {
  id: string
  userId: string
  displayName: string
  team: string | null
  assignedRole: string | null
  isReady: boolean
  isCaptain: boolean
  joinOrder: number
}

export interface MatchOptionInfo {
  optionNumber: number
  algorithmType: string
  blueTeamAvgMmr: number
  redTeamAvgMmr: number
  mmrDifference: number
  balanceScore: number
  avgBlueComfort: number
  avgRedComfort: number
  maxLaneDiff: number
  usedMmrThreshold: number
  assignments: AssignmentInfo[]
}

export interface AssignmentInfo {
  userId: string
  displayName: string
  team: string
  assignedRole: string
  roleMmr: number
  comfortRating: number
}

export interface TeamStatsInfo {
  blueTeamAvgMmr: number
  redTeamAvgMmr: number
  mmrDifference: number
  avgBlueComfort: number
  avgRedComfort: number
  laneDiffs: Record<string, number>
}

export interface VotingStatusInfo {
  votingEnabled: boolean
  votingMode: string
  deadline?: string
  totalPlayers: number
  votesCast: number
  voteCounts: Record<number, number>
  voters: Record<number, VoterInfoPayload[]>
  winningOption?: number
  canFinalize: boolean
}

export interface VoterInfoPayload {
  userId: string
  displayName: string
}

export interface InMemoryActionInfo {
  id: string
  actionType: string
  status: string
  proposedByUser: string
  proposedBySide: string
  player1Id?: string
  player2Id?: string
  matchOptionNum?: number
  approvedByBlue: boolean
  approvedByRed: boolean
  expiresAt: string
}

// ============== Event Payloads ==============

export interface LobbyStateSyncPayload {
  lobby: LobbyInfo
  players: LobbyPlayerInfo[]
  matchOptions?: MatchOptionInfo[]
  teamStats?: TeamStatsInfo
  votingStatus?: VotingStatusInfo
  votes: Record<string, number>
  pendingAction: InMemoryActionInfo | null
}

export interface PlayerJoinedPayload {
  player: LobbyPlayerInfo
}

export interface PlayerLeftPayload {
  userId: string
  displayName: string
}

export interface PlayerReadyChangedPayload {
  userId: string
  isReady: boolean
}

export interface StatusChangedPayload {
  oldStatus: string
  newStatus: string
}

export interface MatchOptionsGeneratedPayload {
  options: MatchOptionInfo[]
}

export interface TeamSelectedPayload {
  optionNumber: number
  assignments: LobbyPlayerInfo[]
  teamStats?: TeamStatsInfo
}

export interface VoteCastPayload {
  userId: string
  displayName: string
  optionNumber: number
  voteAdded: boolean // true if vote was added, false if removed
  voteCounts: Record<number, number>
  votesCast: number
  voters: Record<number, VoterInfoPayload[]>
}

export interface ActionProposedPayload {
  action: InMemoryActionInfo
}

export interface ActionApprovedPayload {
  actionId: string
  approvedBySide: string
  approvedByBlue: boolean
  approvedByRed: boolean
}

export interface ActionExecutedPayload {
  actionType: string
  result?: unknown
}

export interface ActionCancelledPayload {
  actionId: string
  cancelledBy: string
}

export interface DraftStartingPayload {
  roomId: string
  shortCode: string
}

export interface CaptainChangedPayload {
  team: string
  newCaptainId: string
  newCaptainName: string
  oldCaptainId?: string
}

export interface PlayerKickedPayload {
  userId: string
  displayName: string
  kickedBy: string
}

export interface TeamStatsUpdatedPayload {
  stats: TeamStatsInfo
}

export interface VotingStatusUpdatedPayload {
  status: VotingStatusInfo
}

export interface LobbyErrorPayload {
  code: string
  message: string
}

// Client -> Server
export interface JoinLobbyPayload {
  lobbyId: string
}

// ============== Converters ==============

export function toLobbyPlayer(info: LobbyPlayerInfo): LobbyPlayer {
  return {
    id: info.id,
    userId: info.userId,
    displayName: info.displayName,
    team: info.team as 'blue' | 'red' | 'spectator' | null,
    assignedRole: info.assignedRole as Role | null,
    isReady: info.isReady,
    isCaptain: info.isCaptain,
    joinOrder: info.joinOrder,
  }
}

export function toMatchOption(info: MatchOptionInfo): MatchOption {
  return {
    optionNumber: info.optionNumber,
    algorithmType: info.algorithmType as 'mmr_balanced' | 'role_comfort' | 'hybrid' | 'lane_balanced' | 'comfort_first',
    blueTeamAvgMmr: info.blueTeamAvgMmr,
    redTeamAvgMmr: info.redTeamAvgMmr,
    mmrDifference: info.mmrDifference,
    balanceScore: info.balanceScore,
    avgBlueComfort: info.avgBlueComfort,
    avgRedComfort: info.avgRedComfort,
    maxLaneDiff: info.maxLaneDiff,
    usedMmrThreshold: info.usedMmrThreshold,
    assignments: info.assignments.map(a => ({
      userId: a.userId,
      displayName: a.displayName,
      team: a.team as 'blue' | 'red' | 'spectator',
      assignedRole: a.assignedRole as Role,
      roleMmr: a.roleMmr,
      comfortRating: a.comfortRating,
    })),
  }
}

export function toTeamStats(info: TeamStatsInfo): TeamStats {
  const laneDiffs: Record<Role, number> = {
    top: 0,
    jungle: 0,
    mid: 0,
    adc: 0,
    support: 0,
  }
  for (const [role, diff] of Object.entries(info.laneDiffs)) {
    if (role in laneDiffs) {
      laneDiffs[role as Role] = diff
    }
  }
  return {
    blueTeamAvgMmr: info.blueTeamAvgMmr,
    redTeamAvgMmr: info.redTeamAvgMmr,
    mmrDifference: info.mmrDifference,
    avgBlueComfort: info.avgBlueComfort,
    avgRedComfort: info.avgRedComfort,
    laneDiffs,
  }
}

export function toPendingAction(info: InMemoryActionInfo | null): PendingAction | null {
  if (!info) return null
  return {
    id: info.id,
    actionType: info.actionType as 'swap_players' | 'swap_roles' | 'matchmake' | 'select_option' | 'start_draft',
    status: info.status as 'pending' | 'approved' | 'cancelled' | 'expired',
    proposedByUser: info.proposedByUser,
    proposedBySide: info.proposedBySide as 'blue' | 'red' | 'spectator',
    player1Id: info.player1Id,
    player2Id: info.player2Id,
    matchOptionNum: info.matchOptionNum,
    approvedByBlue: info.approvedByBlue,
    approvedByRed: info.approvedByRed,
    expiresAt: info.expiresAt,
  }
}

export function toVotingStatus(info: VotingStatusInfo): VotingStatus {
  const voters: Record<number, { userId: string; displayName: string }[]> = {}
  for (const [optNum, voterList] of Object.entries(info.voters)) {
    voters[parseInt(optNum)] = voterList.map(v => ({
      userId: v.userId,
      displayName: v.displayName,
    }))
  }
  return {
    votingEnabled: info.votingEnabled,
    votingMode: info.votingMode as 'majority' | 'unanimous' | 'captain_override',
    deadline: info.deadline,
    totalPlayers: info.totalPlayers,
    votesCast: info.votesCast,
    voteCounts: info.voteCounts,
    voters,
    winningOption: info.winningOption,
    canFinalize: info.canFinalize,
  }
}
