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
