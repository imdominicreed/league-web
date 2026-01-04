import { api } from './client'
import { Lobby, LobbyPlayer, MatchOption, Room } from '@/types'

interface CreateLobbyRequest {
  draftMode: 'pro_play' | 'fearless'
  timerDurationSeconds?: number
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

  getMatchOptions: (lobbyId: string): Promise<MatchOption[]> =>
    api.get(`/lobbies/${lobbyId}/match-options`),

  selectOption: (lobbyId: string, optionNumber: number): Promise<Lobby> =>
    api.post(`/lobbies/${lobbyId}/select-option`, { optionNumber }),

  startDraft: (lobbyId: string): Promise<Room> =>
    api.post<Room>(`/lobbies/${lobbyId}/start-draft`),
}
