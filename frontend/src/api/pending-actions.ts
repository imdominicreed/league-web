import { api } from './client'
import { PendingAction } from '@/types'

export interface LobbyPendingAction {
  lobbyId: string
  lobbyCode: string
  lobbyName?: string
  action: PendingAction
  needsYourApproval: boolean
}

export interface DraftPendingAction {
  roomId: string
  roomCode: string
  actionType: 'pick' | 'ban' | 'pending_edit' | 'ready_to_resume'
  isYourTurn: boolean
  currentPhase?: number
  timerRemaining?: number
}

export interface UnifiedPendingActions {
  lobbyActions: LobbyPendingAction[]
  draftActions: DraftPendingAction[]
}

export const pendingActionsApi = {
  getAll: (): Promise<UnifiedPendingActions> =>
    api.get('/pending-actions'),
}
