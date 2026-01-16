import { api } from './client'
import { MatchHistoryItem, MatchDetail } from '@/types'

interface ListParams {
  limit?: number
  offset?: number
}

export const matchHistoryApi = {
  list: (params?: ListParams): Promise<MatchHistoryItem[]> => {
    const queryParams = new URLSearchParams()
    if (params?.limit) queryParams.set('limit', params.limit.toString())
    if (params?.offset) queryParams.set('offset', params.offset.toString())
    const query = queryParams.toString()
    return api.get(`/match-history${query ? `?${query}` : ''}`)
  },

  getDetail: (roomId: string): Promise<MatchDetail> =>
    api.get(`/match-history/${roomId}`),
}
