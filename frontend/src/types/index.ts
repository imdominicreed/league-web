export interface User {
  id: string
  displayName: string
}

export interface Champion {
  id: string
  key: string
  name: string
  title: string
  imageUrl: string
  tags: string[]
}

export interface Room {
  id: string
  shortCode: string
  draftMode: 'pro_play' | 'fearless'
  timerDurationSeconds: number
  status: 'waiting' | 'in_progress' | 'completed'
  blueSideUserId?: string
  redSideUserId?: string
}

export interface DraftState {
  currentPhase: number
  currentTeam: 'blue' | 'red' | null
  actionType: 'ban' | 'pick' | null
  timerRemainingMs: number
  blueBans: string[]
  redBans: string[]
  bluePicks: string[]
  redPicks: string[]
  isComplete: boolean
  teamPlayers?: TeamPlayer[]
  isTeamDraft?: boolean
}

export interface Player {
  userId: string
  displayName: string
  ready: boolean
}

export interface RoomState {
  room: Room | null
  players: {
    blue: Player | null
    red: Player | null
  }
  yourSide: 'blue' | 'red' | 'spectator' | null
  spectatorCount: number
}

export type Side = 'blue' | 'red' | 'spectator'

export type Role = 'top' | 'jungle' | 'mid' | 'adc' | 'support'

export const ALL_ROLES: Role[] = ['top', 'jungle', 'mid', 'adc', 'support']

export const ROLE_DISPLAY_NAMES: Record<Role, string> = {
  top: 'Top',
  jungle: 'Jungle',
  mid: 'Mid',
  adc: 'ADC',
  support: 'Support',
}

export const ROLE_ABBREVIATIONS: Record<Role, string> = {
  top: 'TOP',
  jungle: 'JGL',
  mid: 'MID',
  adc: 'ADC',
  support: 'SUP',
}

export interface TeamPlayer {
  id: string
  displayName: string
  team: 'blue' | 'red'
  assignedRole: Role
  isCaptain: boolean
}

export type LeagueRank =
  | 'Unranked'
  | 'Iron IV' | 'Iron III' | 'Iron II' | 'Iron I'
  | 'Bronze IV' | 'Bronze III' | 'Bronze II' | 'Bronze I'
  | 'Silver IV' | 'Silver III' | 'Silver II' | 'Silver I'
  | 'Gold IV' | 'Gold III' | 'Gold II' | 'Gold I'
  | 'Platinum IV' | 'Platinum III' | 'Platinum II' | 'Platinum I'
  | 'Emerald IV' | 'Emerald III' | 'Emerald II' | 'Emerald I'
  | 'Diamond IV' | 'Diamond III' | 'Diamond II' | 'Diamond I'
  | 'Master' | 'Grandmaster' | 'Challenger'

export const ALL_RANKS: LeagueRank[] = [
  'Unranked',
  'Iron IV', 'Iron III', 'Iron II', 'Iron I',
  'Bronze IV', 'Bronze III', 'Bronze II', 'Bronze I',
  'Silver IV', 'Silver III', 'Silver II', 'Silver I',
  'Gold IV', 'Gold III', 'Gold II', 'Gold I',
  'Platinum IV', 'Platinum III', 'Platinum II', 'Platinum I',
  'Emerald IV', 'Emerald III', 'Emerald II', 'Emerald I',
  'Diamond IV', 'Diamond III', 'Diamond II', 'Diamond I',
  'Master', 'Grandmaster', 'Challenger',
]

export interface RoleProfile {
  role: Role
  leagueRank: LeagueRank
  mmr: number
  comfortRating: number
}

export interface UserProfile {
  user: User
  roleProfiles: RoleProfile[]
}

// Lobby types
export type LobbyStatus = 'waiting_for_players' | 'matchmaking' | 'team_selected' | 'drafting' | 'completed'

export interface Lobby {
  id: string
  shortCode: string
  createdBy: string
  status: LobbyStatus
  selectedMatchOption: number | null
  draftMode: 'pro_play' | 'fearless'
  timerDurationSeconds: number
  roomId: string | null
  players: LobbyPlayer[]
}

export interface LobbyPlayer {
  id: string
  userId: string
  displayName: string
  team: Side | null
  assignedRole: Role | null
  isReady: boolean
}

export interface MatchOption {
  optionNumber: number
  blueTeamAvgMmr: number
  redTeamAvgMmr: number
  mmrDifference: number
  balanceScore: number
  assignments: MatchAssignment[]
}

export interface MatchAssignment {
  userId: string
  displayName: string
  team: Side
  assignedRole: Role
  roleMmr: number
  comfortRating: number
}

// WebSocket message types
export interface WSMessage<T = unknown> {
  type: string
  payload: T
  timestamp: number
}

export interface StateSyncPayload {
  room: {
    id: string
    shortCode: string
    draftMode: string
    status: string
    timerDuration: number
  }
  draft: DraftState
  players: {
    blue: Player | null
    red: Player | null
  }
  yourSide: Side
  spectatorCount: number
  fearlessBans?: string[]
}
