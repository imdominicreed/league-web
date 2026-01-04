import { api } from './client'
import { Champion } from '@/types'

interface ChampionsResponse {
  champions: Champion[]
  version: string
}

export const championsApi = {
  getAll: (): Promise<ChampionsResponse> =>
    api.get('/champions'),

  get: (id: string): Promise<Champion> =>
    api.get(`/champions/${id}`),

  sync: (): Promise<{ synced: number; version: string }> =>
    api.post('/champions/sync'),
}
