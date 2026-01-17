import { api } from './client'
import { Lobby, LobbyPlayer, MatchOption, Room, PendingAction, TeamStats, VotingStatus, VotingMode } from '@/types'

interface CreateLobbyRequest {
  draftMode: 'pro_play' | 'fearless'
  timerDurationSeconds?: number
  votingEnabled?: boolean
  votingMode?: VotingMode
}

interface SwapRequest {
  player1Id: string
  player2Id: string
  swapType: 'players' | 'roles'
}

export const lobbyApi = {
  create: (data: CreateLobbyRequest): Promise<Lobby> =>
    api.post('/lobbies', data),

  get: (idOrCode: string): Promise<Lobby> =>
    api.get(`/lobbies/${idOrCode}`),

  join: (idOrCode: string): Promise<LobbyPlayer> =>
    api.post(`/lobbies/${idOrCode}/join`),

  leave: (idOrCode: string): Promise<{ success: boolean }> =>
    api.post(`/lobbies/${idOrCode}/leave`),

  setReady: (idOrCode: string, ready: boolean): Promise<{ ready: boolean }> =>
    api.post(`/lobbies/${idOrCode}/ready`, { ready }),

  generateTeams: (lobbyId: string): Promise<MatchOption[]> =>
    api.post(`/lobbies/${lobbyId}/generate-teams`),

  loadMoreTeams: (lobbyId: string): Promise<MatchOption[]> =>
    api.post(`/lobbies/${lobbyId}/load-more-teams`),

  getMatchOptions: (lobbyId: string): Promise<MatchOption[]> =>
    api.get(`/lobbies/${lobbyId}/match-options`),

  selectOption: (lobbyId: string, optionNumber: number): Promise<Lobby> =>
    api.post(`/lobbies/${lobbyId}/select-option`, { optionNumber }),

  startDraft: (lobbyId: string): Promise<Room> =>
    api.post<Room>(`/lobbies/${lobbyId}/start-draft`),

  // Captain management
  takeCaptain: (lobbyId: string): Promise<Lobby> =>
    api.post(`/lobbies/${lobbyId}/take-captain`),

  promoteCaptain: (lobbyId: string, userId: string): Promise<Lobby> =>
    api.post(`/lobbies/${lobbyId}/promote-captain`, { userId }),

  kickPlayer: (lobbyId: string, userId: string): Promise<Lobby> =>
    api.post(`/lobbies/${lobbyId}/kick`, { userId }),

  // Pending actions
  proposeSwap: (lobbyId: string, data: SwapRequest): Promise<PendingAction> =>
    api.post(`/lobbies/${lobbyId}/swap`, data),

  proposeMatchmake: (lobbyId: string): Promise<PendingAction> =>
    api.post(`/lobbies/${lobbyId}/propose-matchmake`),

  proposeSelectOption: (lobbyId: string, optionNumber: number): Promise<PendingAction> =>
    api.post(`/lobbies/${lobbyId}/propose-select-option`, { optionNumber }),

  proposeStartDraft: (lobbyId: string): Promise<PendingAction> =>
    api.post(`/lobbies/${lobbyId}/propose-start-draft`),

  getPendingAction: (lobbyId: string): Promise<PendingAction | null> =>
    api.get(`/lobbies/${lobbyId}/pending-action`),

  approvePendingAction: (lobbyId: string, actionId: string): Promise<Lobby> =>
    api.post(`/lobbies/${lobbyId}/pending-action/${actionId}/approve`),

  cancelPendingAction: (lobbyId: string, actionId: string): Promise<Lobby> =>
    api.post(`/lobbies/${lobbyId}/pending-action/${actionId}/cancel`),

  // Team stats
  getTeamStats: (lobbyId: string): Promise<TeamStats> =>
    api.get(`/lobbies/${lobbyId}/team-stats`),

  // Voting
  castVote: (lobbyId: string, optionNumber: number): Promise<VotingStatus> =>
    api.post(`/lobbies/${lobbyId}/vote`, { optionNumber }),

  getVotingStatus: (lobbyId: string): Promise<VotingStatus> =>
    api.get(`/lobbies/${lobbyId}/voting-status`),

  startVoting: (lobbyId: string, durationSeconds?: number): Promise<Lobby> =>
    api.post(`/lobbies/${lobbyId}/start-voting`, { durationSeconds: durationSeconds || 0 }),

  endVoting: (lobbyId: string, forceOption?: number): Promise<Lobby> =>
    api.post(`/lobbies/${lobbyId}/end-voting`, forceOption !== undefined ? { forceOption } : {}),
}
